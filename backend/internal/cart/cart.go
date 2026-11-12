// Package cart serves the guest cart APIs (docs/API.md §2): cart lines
// hold one variant each with quantities only — every price, total and
// stock bound is recomputed from current database data on every read
// and write (never trusted from the client). Cart mutations run in a
// single BEGIN IMMEDIATE transaction that also creates the guest
// session on the first successful write.
package cart

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/session"
)

// MaxQuantityPerLine is the approved quantity bound (§4.4).
const MaxQuantityPerLine = 99

// ErrNotFound marks an unknown cart line (404 NOT_FOUND).
var ErrNotFound = errors.New("cart: line not found")

// FieldError is one §4.3 validation detail entry.
type FieldError struct {
	Field string
	Code  string
}

// ValidationError carries field-level §4.3 details (400).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "cart: validation failed" }

// StockExceeded is a §4.3 409 STOCK_UNAVAILABLE with per-variant
// details.
type StockExceeded struct {
	VariantID   int64
	ProductName string // localized at error time
	Requested   int
	Available   int
}

func (e *StockExceeded) Error() string { return "cart: stock exceeded" }

// Line is one §4.4 cart line: a variant with quantity only — all
// money and bounds come from current database data.
type Line struct {
	ID           int64  `json:"id"`
	VariantID    int64  `json:"variantId"`
	ProductSlug  string `json:"productSlug"`
	ProductName  string `json:"productName"`
	ColorKey     string `json:"colorKey"`
	ColorName    string `json:"colorName"`
	StorageGb    int    `json:"storageGb"`
	UnitPriceIrr int64  `json:"unitPriceIrr"`
	Quantity     int    `json:"quantity"`
	LineTotalIrr int64  `json:"lineTotalIrr"`
	MaxQuantity  int    `json:"maxQuantity"`
}

// Cart is the response of every cart endpoint (§4.4).
type Cart struct {
	Items       []Line `json:"items"`
	SubtotalIrr int64  `json:"subtotalIrr"`
	ItemCount   int    `json:"itemCount"`
	Currency    string `json:"currency"`
}

func emptyCart() *Cart {
	return &Cart{Items: []Line{}, Currency: "IRR"}
}

// Service owns cart reads and writes.
type Service struct {
	db       *database.DB
	sessions *session.Service
}

// New builds a cart Service.
func New(db *database.DB, sessions *session.Service) *Service {
	return &Service{db: db, sessions: sessions}
}

func nameCol(prefix, locale string) string {
	if locale == i18n.LocaleEn {
		return prefix + "_en"
	}
	return prefix + "_fa"
}

// loadLines reads the session's cart lines with CURRENT variant
// prices and stock (§4.4: totals recomputed on every read).
func (s *Service) loadLines(ctx context.Context, q database.Querier, sessionID int64, locale string) (*Cart, error) {
	cart := emptyCart()
	rows, err := q.QueryContext(ctx, fmt.Sprintf(`
		SELECT ci.id, ci.variant_id, ci.quantity, v.color_key, v.%s AS color_name,
		       v.storage_gb, v.price_irr, v.stock_qty, p.slug, p.%s AS product_name
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE ci.cart_id = (SELECT id FROM carts WHERE session_id = ?)
		ORDER BY ci.id`, nameCol("color_name", locale), nameCol("name", locale)),
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("cart: load lines: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var l Line
		if err := rows.Scan(&l.ID, &l.VariantID, &l.Quantity, &l.ColorKey, &l.ColorName,
			&l.StorageGb, &l.UnitPriceIrr, &l.MaxQuantity, &l.ProductSlug, &l.ProductName); err != nil {
			return nil, fmt.Errorf("cart: line scan: %w", err)
		}
		l.LineTotalIrr = l.UnitPriceIrr * int64(l.Quantity)
		cart.SubtotalIrr += l.LineTotalIrr
		cart.ItemCount += l.Quantity
		cart.Items = append(cart.Items, l)
	}
	return cart, rows.Err()
}

// GetCart returns the session's cart. No cookie, unknown or expired
// session → an empty cart (§4.5: never 401); no cookie is issued.
func (s *Service) GetCart(ctx context.Context, sessionToken, locale string) (*Cart, error) {
	sessionID, err := s.sessions.Resolve(ctx, sessionToken)
	if err != nil {
		return nil, err
	}
	if sessionID == 0 {
		return emptyCart(), nil
	}
	return s.loadLines(ctx, s.db, sessionID, locale)
}

// AddItem adds quantity of a variant to the session's cart,
// incrementing an existing line. Inside one transaction it creates
// the guest session (first successful write → Set-Cookie) and the
// cart row, validates the variant and the stock bound
// (existing + requested ≤ current stock), and recomputes the cart.
func (s *Service) AddItem(ctx context.Context, sessionToken string, variantID int64, quantity int, locale string) (*Cart, string, error) {
	if variantID <= 0 || quantity < 1 || quantity > MaxQuantityPerLine {
		fields := make([]FieldError, 0, 2)
		if variantID <= 0 {
			fields = append(fields, FieldError{Field: "variantId", Code: "VARIANT_NOT_FOUND"})
		}
		if quantity < 1 || quantity > MaxQuantityPerLine {
			fields = append(fields, FieldError{Field: "quantity", Code: "INVALID_QUANTITY"})
		}
		return nil, "", &ValidationError{Fields: fields}
	}

	result := emptyCart()
	newToken := ""

	err := s.db.WithTx(ctx, func(tx database.Querier) error {
		sessionID, err := s.sessions.ResolveWith(ctx, tx, sessionToken)
		if err != nil {
			return err
		}
		if sessionID == 0 {
			// First successful write issues the session.
			raw, sid, err := s.sessions.CreateIn(ctx, tx)
			if err != nil {
				return err
			}
			sessionToken, sessionID = raw, sid
			newToken = raw
		}

		var cartID int64
		err = tx.QueryRowContext(ctx,
			"SELECT id FROM carts WHERE session_id = ?", sessionID).Scan(&cartID)
		if errors.Is(err, sql.ErrNoRows) {
			if cartID, err = insertCart(ctx, tx, sessionID); err != nil {
				return err
			}
		} else if err != nil {
			return fmt.Errorf("cart: lookup: %w", err)
		}

		// Variant must exist; its CURRENT stock bounds the line.
		var stock int
		var productName string
		err = tx.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT v.stock_qty, p.%s FROM product_variants v
			JOIN products p ON p.id = v.product_id WHERE v.id = ?`,
			nameCol("name", locale)), variantID).Scan(&stock, &productName)
		if errors.Is(err, sql.ErrNoRows) {
			return &ValidationError{Fields: []FieldError{{Field: "variantId", Code: "VARIANT_NOT_FOUND"}}}
		}
		if err != nil {
			return fmt.Errorf("cart: variant lookup: %w", err)
		}

		existing := 0
		var lineID int64
		err = tx.QueryRowContext(ctx,
			"SELECT id, quantity FROM cart_items WHERE cart_id = ? AND variant_id = ?",
			cartID, variantID).Scan(&lineID, &existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("cart: line lookup: %w", err)
		}

		requested := quantity + existing
		if requested > stock {
			return &StockExceeded{
				VariantID:   variantID,
				ProductName: productName,
				Requested:   requested,
				Available:   stock,
			}
		}

		if errors.Is(err, sql.ErrNoRows) {
			if _, err = tx.ExecContext(ctx,
				"INSERT INTO cart_items (cart_id, variant_id, quantity) VALUES (?, ?, ?)",
				cartID, variantID, quantity); err != nil {
				return fmt.Errorf("cart: insert line: %w", err)
			}
		} else {
			if _, err = tx.ExecContext(ctx,
				"UPDATE cart_items SET quantity = ? WHERE id = ?", requested, lineID); err != nil {
				return fmt.Errorf("cart: update line: %w", err)
			}
		}

		result, err = s.loadLines(ctx, tx, sessionID, locale)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	return result, newToken, nil
}

// UpdateItem sets a line's quantity (1–99) against current stock.
// Unknown line (or no/expired session) → ErrNotFound.
func (s *Service) UpdateItem(ctx context.Context, sessionToken string, lineID, quantity int64, locale string) (*Cart, error) {
	if quantity < 1 || quantity > MaxQuantityPerLine {
		return nil, &ValidationError{Fields: []FieldError{{Field: "quantity", Code: "INVALID_QUANTITY"}}}
	}

	result := emptyCart()
	err := s.db.WithTx(ctx, func(tx database.Querier) error {
		sessionID, err := s.sessions.ResolveWith(ctx, tx, sessionToken)
		if err != nil {
			return err
		}
		if sessionID == 0 {
			return ErrNotFound
		}

		var variantID int64
		var stock int
		var productName string
		err = tx.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT v.id, v.stock_qty, p.%s
			FROM cart_items ci
			JOIN product_variants v ON v.id = ci.variant_id
			JOIN products p ON p.id = v.product_id
			WHERE ci.id = ? AND ci.cart_id = (SELECT id FROM carts WHERE session_id = ?)`,
			nameCol("name", locale)), lineID, sessionID).Scan(&variantID, &stock, &productName)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("cart: line lookup: %w", err)
		}

		if int(quantity) > stock {
			return &StockExceeded{
				VariantID:   variantID,
				ProductName: productName,
				Requested:   int(quantity),
				Available:   stock,
			}
		}
		if _, err = tx.ExecContext(ctx,
			"UPDATE cart_items SET quantity = ? WHERE id = ?", quantity, lineID); err != nil {
			return fmt.Errorf("cart: update: %w", err)
		}
		result, err = s.loadLines(ctx, tx, sessionID, locale)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RemoveItem deletes a line from the session's cart. Unknown line
// (or no/expired session) → ErrNotFound.
func (s *Service) RemoveItem(ctx context.Context, sessionToken string, lineID int64, locale string) (*Cart, error) {
	result := emptyCart()
	err := s.db.WithTx(ctx, func(tx database.Querier) error {
		sessionID, err := s.sessions.ResolveWith(ctx, tx, sessionToken)
		if err != nil {
			return err
		}
		if sessionID == 0 {
			return ErrNotFound
		}
		res, err := tx.ExecContext(ctx,
			"DELETE FROM cart_items WHERE id = ? AND cart_id = (SELECT id FROM carts WHERE session_id = ?)",
			lineID, sessionID)
		if err != nil {
			return fmt.Errorf("cart: delete: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("cart: delete rows affected: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		result, err = s.loadLines(ctx, tx, sessionID, locale)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func insertCart(ctx context.Context, tx database.Querier, sessionID int64) (int64, error) {
	res, err := tx.ExecContext(ctx, "INSERT INTO carts (session_id) VALUES (?)", sessionID)
	if err != nil {
		return 0, fmt.Errorf("cart: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("cart: insert id: %w", err)
	}
	return id, nil
}
