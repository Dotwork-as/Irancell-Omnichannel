// Package checkout serves the shipping methods API and the
// transactional checkout with the mock payment (docs/API.md §2/§4.6):
// the cart is re-derived from the guest session, every price and the
// stock bound are revalidated inside one BEGIN IMMEDIATE transaction,
// stock is deducted and the order (always `paid` in the MVP) is
// created atomically — any failure rolls back everything.
package checkout

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/internal/orders"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/session"
)

// ErrEmptyCart marks a missing/expired session or a cart without
// lines → 409 CART_EMPTY (§4.3: never 401).
var ErrEmptyCart = errors.New("checkout: empty cart")

// TurnstileRequired marks a checkout that must pass Cloudflare
// Turnstile human verification but carried no token (400).
type TurnstileRequired struct{}

func (e *TurnstileRequired) Error() string { return "checkout: turnstile token required" }

// TurnstileFailed marks failed human verification — verification is
// FAIL-CLOSED: network or validation errors never let a checkout
// proceed (403).
type TurnstileFailed struct{}

func (e *TurnstileFailed) Error() string { return "checkout: turnstile verification failed" }

// FieldError is one §4.3 validation detail entry.
type FieldError struct {
	Field string
	Code  string
}

// ValidationError carries field-level §4.3 details (400).
type ValidationError struct {
	Fields []FieldError
}

func (e *ValidationError) Error() string { return "checkout: validation failed" }

// StockExceeded is a §4.3 409 STOCK_UNAVAILABLE with per-variant
// details.
type StockExceeded struct {
	VariantID   int64
	ProductName string // localized at error time
	Requested   int
	Available   int
}

func (e *StockExceeded) Error() string { return "checkout: stock exceeded" }

// ShippingMethod is one localized, database-owned delivery option
// (§4.4).
type ShippingMethod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceIrr    int64  `json:"priceIrr"`
}

// CheckoutRequest accepts only the approved fields (§4.4) plus the
// optional Turnstile token used when human verification is enabled.
type Recipient struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

type CheckoutRequest struct {
	Recipient        Recipient `json:"recipient"`
	ShippingMethodID string    `json:"shippingMethodId"`
	TurnstileToken   string    `json:"turnstileToken,omitempty"`
}

// Receipt is the §4.4 CheckoutOrder response.
type Receipt struct {
	TrackingCode   string        `json:"trackingCode"`
	Status         string        `json:"status"`
	Recipient      Recipient     `json:"recipient"`
	ShippingMethod shipBrief     `json:"shippingMethod"`
	Items          []receiptItem `json:"items"`
	SubtotalIrr    int64         `json:"subtotalIrr"`
	ShippingIrr    int64         `json:"shippingIrr"`
	TotalIrr       int64         `json:"totalIrr"`
	Currency       string        `json:"currency"`
	Payment        payment       `json:"payment"`
	CreatedAt      string        `json:"createdAt"`
	PaidAt         string        `json:"paidAt"`
}

type shipBrief struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PriceIrr int64  `json:"priceIrr"`
}

type receiptItem struct {
	ProductName  string `json:"productName"`
	ColorName    string `json:"colorName"`
	StorageGb    int    `json:"storageGb"`
	UnitPriceIrr int64  `json:"unitPriceIrr"`
	Quantity     int    `json:"quantity"`
	LineTotalIrr int64  `json:"lineTotalIrr"`
}

type payment struct {
	Method        string `json:"method"`
	TransactionID string `json:"transactionId"`
}

// Service owns shipping reads and checkout.
type Service struct {
	db              *database.DB
	sessions        *session.Service
	turnstileSecret string
	turnstileURL    string
	client          *http.Client
}

// New builds a checkout Service. An empty cfg.TurnstileSecretKey
// disables human verification (local MVP behavior).
func New(db *database.DB, sessions *session.Service, cfg config.Config) *Service {
	return &Service{
		db:              db,
		sessions:        sessions,
		turnstileSecret: cfg.TurnstileSecretKey,
		turnstileURL:    cfg.TurnstileVerifyURL,
		client:          &http.Client{Timeout: 5 * time.Second},
	}
}

func nameCol(prefix, locale string) string {
	if locale == i18n.LocaleEn {
		return prefix + "_en"
	}
	return prefix + "_fa"
}

// ShippingMethods returns the active, database-owned delivery options
// localized (§4.2).
func (s *Service) ShippingMethods(ctx context.Context, locale string) ([]ShippingMethod, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, %s AS name, %s AS description, price_irr
		FROM shipping_methods WHERE active = 1
		ORDER BY CASE id WHEN 'express' THEN 0 ELSE 1 END`,
		nameCol("name", locale), nameCol("description", locale)))
	if err != nil {
		return nil, fmt.Errorf("checkout: shipping methods: %w", err)
	}
	defer rows.Close()

	methods := []ShippingMethod{}
	for rows.Next() {
		var m ShippingMethod
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.PriceIrr); err != nil {
			return nil, fmt.Errorf("checkout: shipping scan: %w", err)
		}
		methods = append(methods, m)
	}
	return methods, rows.Err()
}

// validateRequest applies the approved §4.4 rules: names 2–60 chars,
// Iranian mobile `^09\d{9}$`, address 10–500 chars, shipping method
// id present. Body validation happens BEFORE the transaction (§4.6
// step 1).
func validateRequest(req CheckoutRequest) []FieldError {
	var fields []FieldError
	req.Recipient.FirstName = strings.TrimSpace(req.Recipient.FirstName)
	req.Recipient.LastName = strings.TrimSpace(req.Recipient.LastName)

	for _, f := range []struct {
		field, value string
	}{
		{"recipient.firstName", req.Recipient.FirstName},
		{"recipient.lastName", req.Recipient.LastName},
	} {
		if f.value == "" {
			fields = append(fields, FieldError{Field: f.field, Code: "REQUIRED"})
			continue
		}
		if n := len([]rune(f.value)); n < 2 || n > 60 {
			fields = append(fields, FieldError{Field: f.field, Code: "NAME_LENGTH"})
		}
	}

	phone := strings.TrimSpace(req.Recipient.Phone)
	if phone == "" {
		fields = append(fields, FieldError{Field: "recipient.phone", Code: "REQUIRED"})
	} else if !validPhone(phone) {
		fields = append(fields, FieldError{Field: "recipient.phone", Code: "INVALID_PHONE"})
	}

	addr := strings.TrimSpace(req.Recipient.Address)
	if addr == "" {
		fields = append(fields, FieldError{Field: "recipient.address", Code: "REQUIRED"})
	} else if n := len([]rune(addr)); n < 10 || n > 500 {
		fields = append(fields, FieldError{Field: "recipient.address", Code: "ADDRESS_LENGTH"})
	}

	if strings.TrimSpace(req.ShippingMethodID) == "" {
		fields = append(fields, FieldError{Field: "shippingMethodId", Code: "REQUIRED"})
	}
	if len(req.TurnstileToken) > 2048 {
		fields = append(fields, FieldError{Field: "turnstileToken", Code: "TURNSTILE_FAILED"})
	}
	return fields
}

func validPhone(p string) bool {
	if len(p) != 11 || !strings.HasPrefix(p, "09") {
		return false
	}
	for _, r := range p[2:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// line is a cart line re-derived from the session with CURRENT
// variant data — the single source of truth for money and stock.
type line struct {
	VariantID        int64
	PriceIrr         int64
	StockQty         int
	Quantity         int
	StorageGb        int
	Slug             string
	NameFa, NameEn   string
	ColorFa, ColorEn string
}

// Checkout runs the §4.6 flow of record. On success the order is
// `paid`, stock is deducted, the cart is emptied, and the receipt is
// returned; on any failure the entire transaction rolls back — no
// order, no payment record, no stock change persists.
func (s *Service) Checkout(ctx context.Context, sessionToken string, req CheckoutRequest, locale string) (*Receipt, error) {
	if fields := validateRequest(req); len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	// Turnstile human verification (optional, fail-closed): enforced
	// before any database work when the secret is configured.
	if err := s.verifyTurnstile(ctx, req.TurnstileToken); err != nil {
		return nil, err
	}

	receipt := &Receipt{}
	err := s.db.WithTx(ctx, func(tx database.Querier) error {
		// §4.6 step 2: cart from the session; missing/expired session
		// or no lines → 409 CART_EMPTY (never 401).
		sessionID, err := s.sessions.ResolveWith(ctx, tx, sessionToken)
		if err != nil {
			return err
		}
		if sessionID == 0 {
			return ErrEmptyCart
		}

		// Active shipping method, database-owned (never client
		// prices); both locale names are fetched for the order
		// snapshot.
		ship := shipBrief{ID: strings.TrimSpace(req.ShippingMethodID)}
		var nameFa, nameEn string
		err = tx.QueryRowContext(ctx, fmt.Sprintf(`
			SELECT name_fa, name_en, %s AS name, price_irr FROM shipping_methods
			WHERE id = ? AND active = 1`,
			nameCol("name", locale)), ship.ID).
			Scan(&nameFa, &nameEn, &ship.Name, &ship.PriceIrr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return &ValidationError{Fields: []FieldError{
					{Field: "shippingMethodId", Code: "SHIPPING_METHOD_NOT_FOUND"}}}
			}
			return fmt.Errorf("checkout: shipping lookup: %w", err)
		}

		// §4.6 step 3: re-read the cart lines with CURRENT variant
		// prices, stock and both locale names (snapshots).
		var lines []line
		rows, err := tx.QueryContext(ctx, `
			SELECT ci.quantity, v.id, v.price_irr, v.stock_qty, v.storage_gb,
			       p.slug, p.name_fa, p.name_en, v.color_name_fa, v.color_name_en
			FROM cart_items ci
			JOIN product_variants v ON v.id = ci.variant_id
			JOIN products p ON p.id = v.product_id
			WHERE ci.cart_id = (SELECT id FROM carts WHERE session_id = ?)
			ORDER BY ci.id`, sessionID)
		if err != nil {
			return fmt.Errorf("checkout: load lines: %w", err)
		}
		for rows.Next() {
			var l line
			if err := rows.Scan(&l.Quantity, &l.VariantID, &l.PriceIrr, &l.StockQty,
				&l.StorageGb, &l.Slug, &l.NameFa, &l.NameEn, &l.ColorFa, &l.ColorEn); err != nil {
				rows.Close()
				return fmt.Errorf("checkout: line scan: %w", err)
			}
			lines = append(lines, l)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("checkout: lines: %w", err)
		}
		rows.Close()
		if len(lines) == 0 {
			return ErrEmptyCart
		}

		// §4.6 step 4: revalidate stock per line.
		for _, l := range lines {
			if l.Quantity > l.StockQty {
				return &StockExceeded{
					VariantID:   l.VariantID,
					ProductName: pick(l.NameFa, l.NameEn, locale),
					Requested:   l.Quantity,
					Available:   l.StockQty,
				}
			}
		}

		// §4.6 step 5: mock payment — generated id, always succeeds in
		// the normal flow; no gateway, no card data.
		txn, err := newTransactionID()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		tracking, err := newTrackingCode()
		if err != nil {
			return err
		}

		subtotal := int64(0)
		items := make([]orders.InsertItem, 0, len(lines))
		receiptItems := make([]receiptItem, 0, len(lines))
		for _, l := range lines {
			lineTotal := l.PriceIrr * int64(l.Quantity)
			subtotal += lineTotal

			// §4.6 step 6a: deduct stock with a guarded update; anything
			// other than exactly one affected row is a hard rollback.
			res, err := tx.ExecContext(ctx,
				"UPDATE product_variants SET stock_qty = stock_qty - ? WHERE id = ? AND stock_qty >= ?",
				l.Quantity, l.VariantID, l.Quantity)
			if err != nil {
				return fmt.Errorf("checkout: deduct stock: %w", err)
			}
			if affected, err := res.RowsAffected(); err != nil || affected != 1 {
				return &StockExceeded{
					VariantID:   l.VariantID,
					ProductName: pick(l.NameFa, l.NameEn, locale),
					Requested:   l.Quantity,
					Available:   l.StockQty,
				}
			}

			items = append(items, orders.InsertItem{
				VariantID: l.VariantID, ProductSlug: l.Slug,
				ProductNameFa: l.NameFa, ProductNameEn: l.NameEn,
				ColorNameFa: l.ColorFa, ColorNameEn: l.ColorEn,
				StorageGb: l.StorageGb, UnitPriceIrr: l.PriceIrr,
				Quantity: l.Quantity, LineTotalIrr: lineTotal,
			})
			receiptItems = append(receiptItems, receiptItem{
				ProductName: pick(l.NameFa, l.NameEn, locale),
				ColorName:   pick(l.ColorFa, l.ColorEn, locale),
				StorageGb:   l.StorageGb, UnitPriceIrr: l.PriceIrr,
				Quantity: l.Quantity, LineTotalIrr: lineTotal,
			})
		}

		// §4.6 step 6b: the order — real checkout is always `paid`.
		order := orders.Insert{
			TrackingCode: tracking, SessionID: sessionID, Status: orders.StatusPaid,
			RecipientFirstName:   strings.TrimSpace(req.Recipient.FirstName),
			RecipientLastName:    strings.TrimSpace(req.Recipient.LastName),
			RecipientPhone:       strings.TrimSpace(req.Recipient.Phone),
			RecipientAddress:     strings.TrimSpace(req.Recipient.Address),
			ShippingMethodID:     ship.ID,
			ShippingNameFa:       nameFa,
			ShippingNameEn:       nameEn,
			ShippingPriceIrr:     ship.PriceIrr,
			SubtotalIrr:          subtotal,
			TotalIrr:             subtotal + ship.PriceIrr,
			PaymentTransactionID: txn,
			CreatedAt:            now,
			PaidAt:               now,
			Items:                items,
		}
		if _, err := orders.InsertOrder(ctx, tx, order); err != nil {
			return err
		}

		// §4.6 step 6c: clear the session's cart lines.
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM cart_items WHERE cart_id = (SELECT id FROM carts WHERE session_id = ?)",
			sessionID); err != nil {
			return fmt.Errorf("checkout: clear cart: %w", err)
		}

		// §4.6 step 7: the receipt.
		receipt = &Receipt{
			TrackingCode: tracking, Status: orders.StatusPaid,
			Recipient:      req.Recipient,
			ShippingMethod: ship,
			Items:          receiptItems,
			SubtotalIrr:    subtotal,
			ShippingIrr:    ship.PriceIrr,
			TotalIrr:       subtotal + ship.PriceIrr,
			Currency:       "IRR",
			Payment:        payment{Method: "mock", TransactionID: txn},
			CreatedAt:      now,
			PaidAt:         now,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// verifyTurnstile checks a Cloudflare Turnstile token when human
// verification is configured (secret non-empty). Missing tokens,
// verification failures, and unreachable verification endpoints ALL
// fail closed — the checkout never proceeds unverified.
func (s *Service) verifyTurnstile(ctx context.Context, token string) error {
	if s.turnstileSecret == "" {
		return nil // disabled — local MVP behavior (documented)
	}
	if strings.TrimSpace(token) == "" {
		return &TurnstileRequired{}
	}
	form := url.Values{}
	form.Set("secret", s.turnstileSecret)
	form.Set("response", strings.TrimSpace(token))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.turnstileURL, strings.NewReader(form.Encode()))
	if err != nil {
		return &TurnstileFailed{}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return &TurnstileFailed{} // unreachable verifier → fail closed
	}
	defer resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || !result.Success {
		return &TurnstileFailed{}
	}
	return nil
}

func pick(fa, en, locale string) string {
	if locale == i18n.LocaleEn {
		return en
	}
	return fa
}

// trackingAlphabet excludes visually ambiguous characters (0/O, 1/I/L).
const trackingAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// newTrackingCode generates an IRC-XXXXXXXX code (docs/API.md §4),
// unique enough for the MVP that the UNIQUE constraint is a hard
// backstop.
func newTrackingCode() (string, error) {
	suffix, err := randChars(8)
	if err != nil {
		return "", fmt.Errorf("checkout: tracking code: %w", err)
	}
	return "IRC-" + suffix, nil
}

// newTransactionID generates a TXN-XXXXXXXXXXXX id for the mock
// payment record.
func newTransactionID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("checkout: transaction id: %w", err)
	}
	return "TXN-" + hex.EncodeToString(buf), nil
}

func randChars(n int) (string, error) {
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(trackingAlphabet))))
		if err != nil {
			return "", err
		}
		out[i] = trackingAlphabet[idx.Int64()]
	}
	return string(out), nil
}
