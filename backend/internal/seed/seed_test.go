package seed

import (
	"context"
	"path/filepath"
	"testing"

	"backend/internal/platform/database"
)

func seededDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx, database.Migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Run(ctx, db, true); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return db
}

func count(t *testing.T, db *database.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

func TestSeedCatalogCounts(t *testing.T) {
	db := seededDB(t)

	if n := count(t, db, "SELECT COUNT(*) FROM products"); n != 6 {
		t.Errorf("products = %d, want 6", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM product_variants"); n != 21 { // 4×4 MVP + 5 single-variant
		t.Errorf("variants = %d, want 21", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM product_specs"); n != 36 { // 6 spec rows × 6 products
		t.Errorf("specs = %d, want 36", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM product_reviews"); n != 12 { // 3+2+2+2+2+1
		t.Errorf("reviews = %d, want 12", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM shipping_methods"); n != 2 {
		t.Errorf("shipping methods = %d, want 2", n)
	}
}

func TestSeedLocalizedContent(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	var nameFa, nameEn string
	err := db.QueryRowContext(ctx,
		"SELECT name_fa, name_en FROM products WHERE slug = 'iphone-15-pro-max'").Scan(&nameFa, &nameEn)
	if err != nil {
		t.Fatalf("lookup product: %v", err)
	}
	if nameFa != "آیفون ۱۵ پرو مکس" || nameEn != "iPhone 15 Pro Max" {
		t.Errorf("names = %q / %q, want frontend values", nameFa, nameEn)
	}

	var colorFa, colorEn string
	if err := db.QueryRowContext(ctx, `
		SELECT color_name_fa, color_name_en FROM product_variants WHERE sku = 'IP15PM-NAT-256'`).
		Scan(&colorFa, &colorEn); err != nil {
		t.Fatalf("lookup variant: %v", err)
	}
	if colorFa != "تیتانیوم طبیعی" || colorEn != "Natural Titanium" {
		t.Errorf("color names = %q / %q, want frontend values", colorFa, colorEn)
	}

	var specCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM product_specs
		WHERE label_fa <> '' AND label_en <> '' AND value_fa <> '' AND value_en <> ''`).
		Scan(&specCount); err != nil {
		t.Fatalf("specs: %v", err)
	}
	if specCount != 36 {
		t.Errorf("fully populated specs = %d, want 36 (fa + en always present)", specCount)
	}

	// Backend never stores image data (frontend-managed per contract).
	var imageColumns int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND sql LIKE '%image%'`).Scan(&imageColumns); err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	if imageColumns != 0 {
		t.Errorf("found %d tables mentioning images; imagery stays frontend-side", imageColumns)
	}
}

func TestSeedPricesStocksAndDefaultVariant(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	type row struct {
		price, compare, stock int64
	}
	cases := []struct {
		sku            string
		price, compare int64
		stock          int64
	}{
		{"IP15PM-NAT-256", 489_900_000, 525_000_000, 7}, // catalog price, old price, §4.4 example stock
		{"IP15PM-NAT-128", 464_900_000, 500_000_000, 9}, // one tier below default
		{"IP15PM-NAT-512", 514_900_000, 550_000_000, 9}, // one tier above default
		{"IP15PM-WHT-1TB", 539_900_000, 575_000_000, 0}, // forced zero-stock variant
		{"IP15P-BLU-256", 435_000_000, 0, 6},
		{"IP14PL-BLU-128", 299_000_000, 315_000_000, 5},
		{"IPSE3-WHT-128", 198_000_000, 0, 0}, // fully unavailable product
	}
	for _, tc := range cases {
		var r row
		var compare any
		err := db.QueryRowContext(ctx, `
			SELECT price_irr, compare_at_price_irr, stock_qty
			FROM product_variants WHERE sku = ?`, tc.sku).Scan(&r.price, &compare, &r.stock)
		if err != nil {
			t.Fatalf("%s: %v", tc.sku, err)
		}
		if r.price != tc.price {
			t.Errorf("%s price = %d, want %d", tc.sku, r.price, tc.price)
		}
		if tc.compare == 0 {
			if compare != nil {
				t.Errorf("%s compare_at = %v, want NULL", tc.sku, compare)
			}
		} else if got, _ := compare.(int64); got != tc.compare {
			t.Errorf("%s compare_at = %v, want %d", tc.sku, compare, tc.compare)
		}
		if r.stock != tc.stock {
			t.Errorf("%s stock = %d, want %d", tc.sku, r.stock, tc.stock)
		}
	}

	// Every product's default_variant_id must resolve to its own
	// default SKU.
	var broken int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM products p
		JOIN product_variants v ON v.id = p.default_variant_id
		JOIN product_variants dv ON dv.sku =
			CASE p.slug
				WHEN 'iphone-15-pro-max' THEN 'IP15PM-NAT-256'
				WHEN 'iphone-15-pro'     THEN 'IP15P-BLU-256'
				WHEN 'iphone-15'         THEN 'IP15-PNK-128'
				WHEN 'iphone-14-plus'    THEN 'IP14PL-BLU-128'
				WHEN 'iphone-14'         THEN 'IP14-BLK-128'
				WHEN 'iphone-se-3'       THEN 'IPSE3-WHT-128'
			END
		WHERE v.id <> dv.id`).Scan(&broken); err != nil {
		t.Fatalf("default variant check: %v", err)
	}
	if broken != 0 {
		t.Errorf("%d products have a default_variant_id that is not their default SKU", broken)
	}
}

func TestSeedShippingMethods(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	var price int64
	var nameFa string
	if err := db.QueryRowContext(ctx,
		"SELECT price_irr, name_fa FROM shipping_methods WHERE id = 'express'").Scan(&price, &nameFa); err != nil {
		t.Fatalf("express: %v", err)
	}
	if price != 490_000 { // 49,000 toman × 10
		t.Errorf("express price = %d, want 490000", price)
	}
	if nameFa != "پیک اکسپرس تهران" {
		t.Errorf("express name_fa = %q, want frontend value", nameFa)
	}
	if err := db.QueryRowContext(ctx,
		"SELECT price_irr FROM shipping_methods WHERE id = 'post'").Scan(&price); err != nil {
		t.Fatalf("post: %v", err)
	}
	if price != 0 {
		t.Errorf("post price = %d, want 0 (free)", price)
	}
}

func TestSeedDemoOrders(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 1"); n != 7 {
		t.Fatalf("demo orders = %d, want 7", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 0"); n != 0 {
		t.Errorf("real orders = %d, want 0 (seeds never create real orders)", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE session_id IS NOT NULL"); n != 0 {
		t.Errorf("demo orders with a session = %d, want 0", n)
	}

	var statuses int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT status) FROM orders WHERE is_demo = 1`).Scan(&statuses); err != nil {
		t.Fatalf("distinct statuses: %v", err)
	}
	if statuses != 7 {
		t.Errorf("distinct demo statuses = %d, want 7", statuses)
	}
	for _, s := range []string{"pending", "paid", "processing", "shipped", "delivered", "cancelled", "failed"} {
		if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE status = ? AND is_demo = 1", s); n != 1 {
			t.Errorf("status %s: %d demo orders, want 1", s, n)
		}
	}

	// Arithmetic and payment-field rules.
	var badTotals, badPayment int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders o
		WHERE o.total_irr <> o.subtotal_irr + o.shipping_price_irr
		   OR o.subtotal_irr <> (SELECT COALESCE(SUM(line_total_irr), 0) FROM order_items i WHERE i.order_id = o.id)`).
		Scan(&badTotals); err != nil {
		t.Fatalf("totals: %v", err)
	}
	if badTotals != 0 {
		t.Errorf("%d demo orders have inconsistent totals", badTotals)
	}
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE (status = 'pending' AND (payment_transaction_id IS NOT NULL OR paid_at IS NOT NULL))
		   OR (status = 'failed'  AND (payment_transaction_id IS NULL OR paid_at IS NOT NULL))
		   OR (status IN ('paid','processing','shipped','delivered','cancelled')
		       AND (payment_transaction_id IS NULL OR paid_at IS NULL))`).
		Scan(&badPayment); err != nil {
		t.Fatalf("payment fields: %v", err)
	}
	if badPayment != 0 {
		t.Errorf("%d demo orders violate the payment-field rules per status", badPayment)
	}

	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE tracking_code NOT LIKE 'DEMO-%' AND is_demo = 1"); n != 0 {
		t.Errorf("%d demo orders without the DEMO- tracking prefix", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM order_items"); n == 0 {
		t.Error("demo order items missing")
	}
}

func TestSeedIdempotentAndNeverResets(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	// Simulate runtime changes: deducted stock and a changed price.
	if _, err := db.ExecContext(ctx,
		"UPDATE product_variants SET stock_qty = 3, price_irr = 123 WHERE sku = 'IP15PM-NAT-256'"); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	if err := Run(ctx, db, true); err != nil {
		t.Fatalf("re-seed: %v", err)
	}

	if n := count(t, db, "SELECT COUNT(*) FROM products"); n != 6 {
		t.Errorf("products after re-seed = %d, want 6", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders"); n != 7 {
		t.Errorf("orders after re-seed = %d, want 7 (no duplicates)", n)
	}

	var stock int64
	var price int64
	if err := db.QueryRowContext(ctx,
		"SELECT stock_qty, price_irr FROM product_variants WHERE sku = 'IP15PM-NAT-256'").Scan(&stock, &price); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if stock != 3 || price != 123 {
		t.Errorf("runtime data reset by re-seed: stock=%d price=%d, want 3 and 123", stock, price)
	}
}

func TestSeedWithoutDemoOrders(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if err := db.Migrate(ctx, database.Migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := Run(ctx, db, false); err != nil {
		t.Fatalf("seed without demo: %v", err)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders"); n != 0 {
		t.Errorf("orders = %d, want 0 when demo seeding is disabled", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM products"); n != 6 {
		t.Errorf("products = %d, want 6 (catalog seeds are not gated)", n)
	}
}

func TestSchemaIntegrity(t *testing.T) {
	db := seededDB(t)
	ctx := context.Background()

	// No dangling foreign keys.
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("foreign_key_check: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Error("foreign_key_check reported violations")
	}

	// Expected indexes exist.
	for _, idx := range []string{
		"idx_product_variants_product", "idx_cart_items_cart",
		"idx_orders_session", "idx_orders_status", "idx_orders_demo",
		"idx_order_items_order",
	} {
		if n := count(t, db,
			"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", idx); n != 1 {
			t.Errorf("index %s missing", idx)
		}
	}

	// Uniqueness: duplicate slug / sku / cart line are rejected.
	if _, err := db.ExecContext(ctx,
		"INSERT INTO products (slug, name_fa, name_en, default_variant_id, rating) VALUES ('iphone-15-pro-max', 'x', 'x', 0, 1)"); err == nil {
		t.Error("duplicate product slug accepted")
	}
	if _, err := db.ExecContext(ctx,
		"INSERT INTO product_variants (product_id, sku, color_key, color_name_fa, color_name_en, storage_gb, price_irr) SELECT product_id, 'IP15PM-NAT-256', color_key, color_name_fa, color_name_en, storage_gb, price_irr FROM product_variants WHERE sku = 'IP15PM-NAT-256'"); err == nil {
		t.Error("duplicate variant sku accepted")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO guest_sessions (token_hash, expires_at) VALUES ('h1', '2026-12-08T00:00:00Z');
		INSERT INTO carts (session_id) SELECT id FROM guest_sessions WHERE token_hash = 'h1';
		INSERT INTO cart_items (cart_id, variant_id, quantity)
			SELECT c.id, v.id, 1 FROM carts c, product_variants v WHERE c.session_id = (SELECT id FROM guest_sessions WHERE token_hash = 'h1') AND v.sku = 'IP15PM-NAT-256';
		INSERT INTO cart_items (cart_id, variant_id, quantity)
			SELECT c.id, v.id, 1 FROM carts c, product_variants v WHERE c.session_id = (SELECT id FROM guest_sessions WHERE token_hash = 'h1') AND v.sku = 'IP15PM-NAT-256';
	`); err == nil {
		t.Error("duplicate cart line accepted")
	}

	// CHECK constraints: bad status and inconsistent totals rejected.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orders (tracking_code, status, recipient_first_name, recipient_last_name,
			recipient_phone, recipient_address, shipping_method_id, shipping_name_fa,
			shipping_name_en, shipping_price_irr, subtotal_irr, total_irr)
		VALUES ('X-1', 'weird', 'a', 'b', '09120000000', 'addr', 'post', 'پست', 'Post', 0, 100, 100)`); err == nil {
		t.Error("invalid status accepted")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO orders (tracking_code, status, recipient_first_name, recipient_last_name,
			recipient_phone, recipient_address, shipping_method_id, shipping_name_fa,
			shipping_name_en, shipping_price_irr, subtotal_irr, total_irr)
		VALUES ('X-2', 'paid', 'a', 'b', '09120000000', 'addr', 'post', 'پست', 'Post', 0, 100, 200)`); err == nil {
		t.Error("inconsistent total (≠ subtotal + shipping) accepted")
	}
	// compare_at must stay ≥ price.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO product_variants (product_id, sku, color_key, color_name_fa, color_name_en, storage_gb, price_irr, compare_at_price_irr)
		SELECT product_id, 'TEST-1', color_key, color_name_fa, color_name_en, storage_gb, 100, 50 FROM product_variants WHERE sku = 'IP15PM-NAT-256'`); err == nil {
		t.Error("compare_at below price accepted")
	}
}
