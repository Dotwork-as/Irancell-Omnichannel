package checkout

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"backend/internal/cart"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/seed"
	"backend/internal/session"
)

func testCheckout(t *testing.T) (*Service, *cart.Service, *session.Service, *database.DB) {
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
	if err := seed.Run(ctx, db, false); err != nil {
		t.Fatalf("seed: %v", err)
	}
	sessions := session.New(db, true)
	return New(db, sessions, config.Config{}), cart.New(db, sessions), sessions, db
}

func variantID(t *testing.T, db *database.DB, sku string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRowContext(context.Background(),
		"SELECT id FROM product_variants WHERE sku = ?", sku).Scan(&id); err != nil {
		t.Fatalf("variant %s: %v", sku, err)
	}
	return id
}

func validReq() CheckoutRequest {
	return CheckoutRequest{
		Recipient: Recipient{
			FirstName: "نرگس", LastName: "رضایی",
			Phone: "09123456789", Address: "تهران، خیابان نمونه، پلاک ۱۰، واحد ۲",
		},
		ShippingMethodID: "express",
	}
}

func TestShippingMethods(t *testing.T) {
	svc, _, _, _ := testCheckout(t)
	ctx := context.Background()

	fa, err := svc.ShippingMethods(ctx, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("ShippingMethods fa: %v", err)
	}
	if len(fa) != 2 {
		t.Fatalf("fa methods = %d, want 2", len(fa))
	}
	if fa[0].ID != "express" || fa[0].Name != "پیک اکسپرس تهران" || fa[0].PriceIrr != 490_000 {
		t.Errorf("fa express wrong: %+v", fa[0])
	}
	if fa[1].ID != "post" || fa[1].PriceIrr != 0 {
		t.Errorf("fa post wrong: %+v", fa[1])
	}

	en, err := svc.ShippingMethods(ctx, i18n.LocaleEn)
	if err != nil {
		t.Fatalf("ShippingMethods en: %v", err)
	}
	if en[0].Name != "Tehran express courier" || en[0].Description != "Same-day delivery — Tehran only" {
		t.Errorf("en express wrong: %+v", en[0])
	}
}

func TestCheckoutValidation(t *testing.T) {
	svc, _, _, _ := testCheckout(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		mutate func(*CheckoutRequest)
		want   []FieldError
	}{
		{"missing first name", func(r *CheckoutRequest) { r.Recipient.FirstName = "" },
			[]FieldError{{Field: "recipient.firstName", Code: "REQUIRED"}}},
		{"short first name", func(r *CheckoutRequest) { r.Recipient.FirstName = "م" },
			[]FieldError{{Field: "recipient.firstName", Code: "NAME_LENGTH"}}},
		{"long last name", func(r *CheckoutRequest) { r.Recipient.LastName = strings.Repeat("ط", 61) },
			[]FieldError{{Field: "recipient.lastName", Code: "NAME_LENGTH"}}},
		{"bad phone prefix", func(r *CheckoutRequest) { r.Recipient.Phone = "08123456789" },
			[]FieldError{{Field: "recipient.phone", Code: "INVALID_PHONE"}}},
		{"short phone", func(r *CheckoutRequest) { r.Recipient.Phone = "09123" },
			[]FieldError{{Field: "recipient.phone", Code: "INVALID_PHONE"}}},
		{"non-digit phone", func(r *CheckoutRequest) { r.Recipient.Phone = "0912345678a" },
			[]FieldError{{Field: "recipient.phone", Code: "INVALID_PHONE"}}},
		{"short address", func(r *CheckoutRequest) { r.Recipient.Address = "تهران" },
			[]FieldError{{Field: "recipient.address", Code: "ADDRESS_LENGTH"}}},
		{"missing shipping method", func(r *CheckoutRequest) { r.ShippingMethodID = "" },
			[]FieldError{{Field: "shippingMethodId", Code: "REQUIRED"}}},
		{"multiple errors", func(r *CheckoutRequest) {
			r.Recipient.FirstName = ""
			r.Recipient.Phone = "123"
		}, []FieldError{
			{Field: "recipient.firstName", Code: "REQUIRED"},
			{Field: "recipient.phone", Code: "INVALID_PHONE"},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validReq()
			tc.mutate(&req)
			_, err := svc.Checkout(ctx, "no-session", req, i18n.LocaleFa)
			var validation *ValidationError
			if err == nil || !asValidation(err, &validation) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			if len(validation.Fields) != len(tc.want) {
				t.Fatalf("fields = %+v, want %+v", validation.Fields, tc.want)
			}
			for i, want := range tc.want {
				if validation.Fields[i] != want {
					t.Errorf("field[%d] = %+v, want %+v", i, validation.Fields[i], want)
				}
			}
		})
	}
}

func asValidation(err error, target **ValidationError) bool {
	if e, ok := err.(*ValidationError); ok {
		*target = e
		return true
	}
	return false
}

func asStock(err error, target **StockExceeded) bool {
	if e, ok := err.(*StockExceeded); ok {
		*target = e
		return true
	}
	return false
}

func TestCheckoutUnknownShippingMethod(t *testing.T) {
	svc, carts, _, db := testCheckout(t)
	ctx := context.Background()

	variant := variantID(t, db, "IP15PM-NAT-256")
	_, token, err := carts.AddItem(ctx, "", variant, 1, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	req := validReq()
	req.ShippingMethodID = "drone"
	if _, err = svc.Checkout(ctx, token, req, i18n.LocaleFa); !asValidation(err, new(*ValidationError)) {
		t.Fatalf("err = %v, want ValidationError for shippingMethodId", err)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 0"); n != 0 {
		t.Errorf("orders created = %d, want 0", n)
	}
}

func TestCheckoutSuccess(t *testing.T) {
	svc, carts, _, db := testCheckout(t)
	ctx := context.Background()
	nat := variantID(t, db, "IP15PM-NAT-256")    // stock 7 @ 489,900,000
	pnk := variantIDBySKU(t, db, "IP15-PNK-128") // stock 9 @ 354,000,000

	_, token, err := carts.AddItem(ctx, "", nat, 2, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem nat: %v", err)
	}
	if _, _, err = carts.AddItem(ctx, token, pnk, 1, i18n.LocaleFa); err != nil {
		t.Fatalf("AddItem pnk: %v", err)
	}

	req := validReq()
	req.ShippingMethodID = "post"
	receipt, err := svc.Checkout(ctx, token, req, i18n.LocaleEn)
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	if !strings.HasPrefix(receipt.TrackingCode, "IRC-") || len(receipt.TrackingCode) != 12 {
		t.Errorf("trackingCode = %q, want IRC-XXXXXXXX", receipt.TrackingCode)
	}
	if receipt.Status != "paid" {
		t.Errorf("status = %q, want paid (only reachable real status)", receipt.Status)
	}
	// Totals recomputed from DB prices: 2×489,900,000 + 354,000,000.
	if receipt.SubtotalIrr != 1_333_800_000 || receipt.ShippingIrr != 0 ||
		receipt.TotalIrr != 1_333_800_000 || receipt.Currency != "IRR" {
		t.Errorf("totals wrong: %+v", receipt)
	}
	if receipt.Payment.Method != "mock" || !strings.HasPrefix(receipt.Payment.TransactionID, "TXN-") {
		t.Errorf("payment wrong: %+v", receipt.Payment)
	}
	if receipt.CreatedAt == "" || receipt.PaidAt == "" {
		t.Error("receipt timestamps missing")
	}
	if len(receipt.Items) != 2 || receipt.Items[0].ProductName != "iPhone 15 Pro Max" ||
		receipt.Items[0].ColorName != "Natural Titanium" {
		t.Errorf("receipt items not localized/snapshotted: %+v", receipt.Items)
	}

	// Stock deducted.
	if got := count(t, db, "SELECT stock_qty FROM product_variants WHERE id = ?", nat); got != 5 {
		t.Errorf("NAT-256 stock = %d, want 5 (7−2)", got)
	}
	if got := count(t, db, "SELECT stock_qty FROM product_variants WHERE id = ?", pnk); got != 8 {
		t.Errorf("PNK-128 stock = %d, want 8 (9−1)", got)
	}

	// Cart emptied.
	c, err := carts.GetCart(ctx, token, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if len(c.Items) != 0 {
		t.Errorf("cart after checkout = %+v, want empty", c)
	}

	// Exactly one real order, paid, snapshots in both locales, payment recorded.
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 0"); n != 1 {
		t.Fatalf("real orders = %d, want 1", n)
	}
	var status, txn, nameFa, nameEn, paidAt string
	var isDemo int
	if err := db.QueryRowContext(ctx, `
		SELECT status, payment_transaction_id, shipping_name_fa, shipping_name_en, paid_at, is_demo
		FROM orders WHERE is_demo = 0`).Scan(&status, &txn, &nameFa, &nameEn, &paidAt, &isDemo); err != nil {
		t.Fatalf("order lookup: %v", err)
	}
	if status != "paid" || isDemo != 0 || paidAt == "" || !strings.HasPrefix(txn, "TXN-") {
		t.Errorf("order row wrong: status=%s demo=%d paidAt=%q txn=%q", status, isDemo, paidAt, txn)
	}
	if nameFa != "پست پیشتاز" || nameEn != "Express post" {
		t.Errorf("shipping snapshot wrong: fa=%q en=%q", nameFa, nameEn)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM order_items oi JOIN orders o ON o.id = oi.order_id WHERE o.is_demo = 0"); n != 2 {
		t.Errorf("order items = %d, want 2", n)
	}
}

func count(t *testing.T, db *database.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", q, err)
	}
	return n
}

func variantIDBySKU(t *testing.T, db *database.DB, sku string) int64 {
	t.Helper()
	return variantID(t, db, sku)
}

func TestCheckoutEmptyCartAndExpiredSession(t *testing.T) {
	svc, carts, sessions, db := testCheckout(t)
	ctx := context.Background()

	// No session at all.
	if _, err := svc.Checkout(ctx, "", validReq(), i18n.LocaleFa); err != ErrEmptyCart {
		t.Errorf("no session: err = %v, want ErrEmptyCart (never 401)", err)
	}

	// Valid session, empty cart.
	_, token, err := carts.AddItem(ctx, "", variantID(t, db, "IP15PM-NAT-256"), 1, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	// Remove the only line to empty the cart.
	c, _ := carts.GetCart(ctx, token, i18n.LocaleFa)
	if _, err = carts.RemoveItem(ctx, token, c.Items[0].ID, i18n.LocaleFa); err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if _, err = svc.Checkout(ctx, token, validReq(), i18n.LocaleFa); err != ErrEmptyCart {
		t.Errorf("empty cart: err = %v, want ErrEmptyCart", err)
	}

	// Expired session → also ErrEmptyCart (real expired-token path).
	token, sid, err := sessions.Create(ctx)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = db.ExecContext(ctx,
		"UPDATE guest_sessions SET expires_at = '2000-01-01T00:00:00Z' WHERE id = ?", sid); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if _, err = svc.Checkout(ctx, token, validReq(), i18n.LocaleFa); err != ErrEmptyCart {
		t.Errorf("expired token: err = %v, want ErrEmptyCart", err)
	}
}

func TestCheckoutStockConflictRollsBackEverything(t *testing.T) {
	svc, carts, _, db := testCheckout(t)
	ctx := context.Background()
	variant := variantID(t, db, "IP15PM-NAT-256") // stock 7

	_, token, err := carts.AddItem(ctx, "", variant, 5, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}

	// Stock changes between the cart operation and checkout.
	if _, err = db.ExecContext(ctx,
		"UPDATE product_variants SET stock_qty = 3 WHERE id = ?", variant); err != nil {
		t.Fatalf("stock change: %v", err)
	}

	_, err = svc.Checkout(ctx, token, validReq(), i18n.LocaleFa)
	var stockErr *StockExceeded
	if !asStock(err, &stockErr) {
		t.Fatalf("err = %v, want StockExceeded", err)
	}
	if stockErr.Requested != 5 || stockErr.Available != 3 {
		t.Errorf("details wrong: %+v", stockErr)
	}

	// Nothing persisted: no order, no payment, no stock deduction,
	// cart intact.
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 0"); n != 0 {
		t.Errorf("orders after failure = %d, want 0", n)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE payment_transaction_id LIKE 'TXN-%' AND is_demo = 0"); n != 0 {
		t.Errorf("payment records after failure = %d, want 0", n)
	}
	if got := count(t, db, "SELECT stock_qty FROM product_variants WHERE id = ?", variant); got != 3 {
		t.Errorf("stock after failure = %d, want 3 (no deduction)", got)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM cart_items"); n != 1 {
		t.Fatalf("cart lines after failure = %d, want 1", n)
	}
	var qty int
	if err = db.QueryRowContext(ctx, "SELECT quantity FROM cart_items").Scan(&qty); err != nil {
		t.Fatalf("qty: %v", err)
	}
	if qty != 5 {
		t.Errorf("cart quantity after failure = %d, want 5", qty)
	}
}

func TestConcurrentCheckoutOnLimitedStock(t *testing.T) {
	svc, carts, _, db := testCheckout(t)
	ctx := context.Background()
	variant := variantID(t, db, "IP15PM-NAT-256") // stock 7

	// Two guests, each wanting 5 of the same 7 available units.
	tokens := make([]string, 2)
	var err error
	for i := 0; i < 2; i++ {
		_, tokens[i], err = carts.AddItem(ctx, "", variant, 5, i18n.LocaleFa)
		if err != nil {
			t.Fatalf("AddItem guest %d: %v", i, err)
		}
	}

	type result struct {
		tracking string
		err      error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, token := range tokens {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			receipt, err := svc.Checkout(context.Background(), token, validReq(), i18n.LocaleFa)
			if err != nil {
				results <- result{err: err}
				return
			}
			results <- result{tracking: receipt.TrackingCode}
		}(token)
	}
	wg.Wait()
	close(results)

	wins, losses := 0, 0
	for r := range results {
		switch {
		case r.err == nil:
			wins++
			if !strings.HasPrefix(r.tracking, "IRC-") {
				t.Errorf("winner tracking = %q", r.tracking)
			}
		default:
			var stockErr *StockExceeded
			if asStock(r.err, &stockErr) {
				losses++
			} else {
				t.Errorf("unexpected concurrent error: %v", r.err)
			}
		}
	}
	if wins != 1 || losses != 1 {
		t.Fatalf("wins=%d losses=%d, want exactly 1 winner (limited stock)", wins, losses)
	}

	// Stock deducted exactly once: 7 − 5 = 2; exactly one paid order.
	if got := count(t, db, "SELECT stock_qty FROM product_variants WHERE id = ?", variant); got != 2 {
		t.Errorf("final stock = %d, want 2", got)
	}
	if n := count(t, db, "SELECT COUNT(*) FROM orders WHERE is_demo = 0"); n != 1 {
		t.Errorf("real orders = %d, want 1", n)
	}
}
