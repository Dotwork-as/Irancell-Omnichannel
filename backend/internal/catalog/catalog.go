// Package catalog serves the read-only product APIs (docs/API.md §2):
// GET /api/v1/products (global search) and GET /api/v1/products/{slug}.
// All responses are single-locale (fa default); prices are integer
// IRR; imagery never leaves the frontend — the API exposes only the
// stable colorKey.
package catalog

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
)

// ErrNotFound is returned by GetBySlug for unknown slugs; the HTTP
// layer maps it to a localized 404 envelope.
var ErrNotFound = errors.New("catalog: product not found")

// ProductSummary is one row of the list/search response (§4.4). The
// color/storage/price fields describe the default variant;
// inStock is therefore the default variant's availability.
type ProductSummary struct {
	Slug              string  `json:"slug"`
	Name              string  `json:"name"`
	DefaultVariantID  int64   `json:"defaultVariantId"`
	ColorKey          string  `json:"colorKey"`
	ColorName         string  `json:"colorName"`
	StorageGb         int     `json:"storageGb"`
	PriceIrr          int64   `json:"priceIrr"`
	CompareAtPriceIrr *int64  `json:"compareAtPriceIrr"`
	InStock           bool    `json:"inStock"`
	Rating            float64 `json:"rating"`
	ReviewCount       int     `json:"reviewCount"`
}

// Variant is a color/storage combination with independent price and
// stock (§4.4).
type Variant struct {
	ID                int64  `json:"id"`
	SKU               string `json:"sku"`
	ColorKey          string `json:"colorKey"`
	ColorName         string `json:"colorName"`
	StorageGb         int    `json:"storageGb"`
	PriceIrr          int64  `json:"priceIrr"`
	CompareAtPriceIrr *int64 `json:"compareAtPriceIrr"`
	StockQty          int    `json:"stockQty"`
	InStock           bool   `json:"inStock"`
}

// Spec is one localized specification row, in seed order.
type Spec struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// Review is one localized review, in seed order.
type Review struct {
	Author string `json:"author"`
	Rating int    `json:"rating"`
	Text   string `json:"text"`
}

// ProductDetail is the GET /products/{slug} response (§4.4): every
// variant actually defined on the product — including unavailable
// ones, with their stock state — plus specs, rating and reviews.
type ProductDetail struct {
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	DefaultVariantID int64     `json:"defaultVariantId"`
	Rating           float64   `json:"rating"`
	ReviewCount      int       `json:"reviewCount"`
	Specs            []Spec    `json:"specs"`
	Reviews          []Review  `json:"reviews"`
	Variants         []Variant `json:"variants"`
}

// Service owns catalog reads against the database.
type Service struct {
	db *database.DB
}

// New builds a catalog Service.
func New(db *database.DB) *Service {
	return &Service{db: db}
}

// nameCol picks the locale column for a prefixed name pair.
func nameCol(prefix, locale string) string {
	if locale == i18n.LocaleEn {
		return prefix + "_en"
	}
	return prefix + "_fa"
}

// likePattern builds an escaped LIKE pattern; SQLite's LIKE is
// case-insensitive for ASCII and Persian has no case.
func likePattern(q string) string {
	var b strings.Builder
	b.WriteByte('%')
	for _, r := range q {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('%')
	return b.String()
}

// List returns every product (q == "") or the products matching the
// global search term (§4.2): product name fa/en, variant color name
// fa/en, SKU or storage value, case-insensitive. Unavailable products
// are always included. Search matches BOTH name columns regardless of
// the response locale, so a Persian term still finds products when
// the response locale is en.
func (s *Service) List(ctx context.Context, q, locale string) ([]ProductSummary, error) {
	// Normalize untrusted input: cap the search term to 100 runes
	// (frontend caps at the same length; abuse-resistant without
	// rejecting the request).
	if len([]rune(q)) > 100 {
		q = string([]rune(q)[:100])
	}
	where := ""
	args := make([]any, 0, 6)
	if q != "" {
		pat := likePattern(q)
		where = `
			WHERE (p.name_fa LIKE ? ESCAPE '\' OR p.name_en LIKE ? ESCAPE '\'
			   OR EXISTS (
			        SELECT 1 FROM product_variants pv WHERE pv.product_id = p.id
			        AND (pv.color_name_fa LIKE ? ESCAPE '\'
			          OR pv.color_name_en LIKE ? ESCAPE '\'
			          OR pv.sku LIKE ? ESCAPE '\'
			          OR CAST(pv.storage_gb AS TEXT) LIKE ? ESCAPE '\')))`
		args = append(args, pat, pat, pat, pat, pat, pat)
	}

	query := fmt.Sprintf(`
		SELECT p.slug, p.%s AS name, p.default_variant_id, p.rating, p.review_count,
		       v.color_key, v.%s AS color_name, v.storage_gb, v.price_irr,
		       v.compare_at_price_irr, v.stock_qty
		FROM products p
		JOIN product_variants v ON v.id = p.default_variant_id
		%s
		ORDER BY p.id`,
		nameCol("name", locale), nameCol("color_name", locale), where)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("catalog: list: %w", err)
	}
	defer rows.Close()

	items := []ProductSummary{}
	for rows.Next() {
		var it ProductSummary
		var compare any
		var stock int64
		if err := rows.Scan(
			&it.Slug, &it.Name, &it.DefaultVariantID, &it.Rating, &it.ReviewCount,
			&it.ColorKey, &it.ColorName, &it.StorageGb, &it.PriceIrr,
			&compare, &stock,
		); err != nil {
			return nil, fmt.Errorf("catalog: list scan: %w", err)
		}
		if compare != nil {
			c := compare.(int64)
			it.CompareAtPriceIrr = &c
		}
		it.InStock = stock > 0
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetBySlug returns the complete localized product detail, or
// ErrNotFound for an unknown slug.
func (s *Service) GetBySlug(ctx context.Context, slug, locale string) (*ProductDetail, error) {
	d := &ProductDetail{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT slug, %s AS name, default_variant_id, rating, review_count
		FROM products WHERE slug = ?`,
		nameCol("name", locale)),
		slug).Scan(&d.Slug, &d.Name, &d.DefaultVariantID, &d.Rating, &d.ReviewCount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("catalog: get %s: %w", slug, err)
	}

	vRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, sku, color_key, %s AS color_name, storage_gb, price_irr,
		       compare_at_price_irr, stock_qty
		FROM product_variants WHERE product_id = (SELECT id FROM products WHERE slug = ?)
		ORDER BY id`, nameCol("color_name", locale)),
		slug)
	if err != nil {
		return nil, fmt.Errorf("catalog: variants %s: %w", slug, err)
	}
	defer vRows.Close()
	d.Variants = []Variant{}
	for vRows.Next() {
		var v Variant
		var compare any
		if err := vRows.Scan(
			&v.ID, &v.SKU, &v.ColorKey, &v.ColorName, &v.StorageGb,
			&v.PriceIrr, &compare, &v.StockQty,
		); err != nil {
			return nil, fmt.Errorf("catalog: variant scan: %w", err)
		}
		if compare != nil {
			c := compare.(int64)
			v.CompareAtPriceIrr = &c
		}
		v.InStock = v.StockQty > 0
		d.Variants = append(d.Variants, v)
	}
	if err := vRows.Err(); err != nil {
		return nil, fmt.Errorf("catalog: variants: %w", err)
	}

	sRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s AS label, %s AS value
		FROM product_specs
		WHERE product_id = (SELECT id FROM products WHERE slug = ?)
		ORDER BY position`, nameCol("label", locale), nameCol("value", locale)),
		slug)
	if err != nil {
		return nil, fmt.Errorf("catalog: specs %s: %w", slug, err)
	}
	defer sRows.Close()
	d.Specs = []Spec{}
	for sRows.Next() {
		var sp Spec
		if err := sRows.Scan(&sp.Label, &sp.Value); err != nil {
			return nil, fmt.Errorf("catalog: spec scan: %w", err)
		}
		d.Specs = append(d.Specs, sp)
	}
	if err := sRows.Err(); err != nil {
		return nil, fmt.Errorf("catalog: specs: %w", err)
	}

	rRows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s AS author, rating, %s AS text
		FROM product_reviews
		WHERE product_id = (SELECT id FROM products WHERE slug = ?)
		ORDER BY position`, nameCol("author", locale), nameCol("text", locale)),
		slug)
	if err != nil {
		return nil, fmt.Errorf("catalog: reviews %s: %w", slug, err)
	}
	defer rRows.Close()
	d.Reviews = []Review{}
	for rRows.Next() {
		var r Review
		if err := rRows.Scan(&r.Author, &r.Rating, &r.Text); err != nil {
			return nil, fmt.Errorf("catalog: review scan: %w", err)
		}
		d.Reviews = append(d.Reviews, r)
	}
	if err := rRows.Err(); err != nil {
		return nil, fmt.Errorf("catalog: reviews: %w", err)
	}

	return d, nil
}
