package cart

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/seed"
	"backend/internal/session"
)

func testCartService(t *testing.T) (*Service, *session.Service, *database.DB) {
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
	return New(db, sessions), sessions, db
}

func variantIDBySKU(t *testing.T, db *database.DB, sku string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRowContext(context.Background(),
		"SELECT id FROM product_variants WHERE sku = ?", sku).Scan(&id); err != nil {
		t.Fatalf("variant %s: %v", sku, err)
	}
	return id
}

func TestAddCreatesSessionAndLine(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()
	variant := variantIDBySKU(t, db, "IP15PM-NAT-256") // stock 7 (§4.4 default variant)

	cart, token, err := svc.AddItem(ctx, "", variant, 2, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if token == "" {
		t.Fatal("first successful write must return a new session token")
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 2 || cart.ItemCount != 2 {
		t.Fatalf("cart = %+v, want one line with quantity 2", cart)
	}
	line := cart.Items[0]
	if line.VariantID != variant || line.ProductSlug != "iphone-15-pro-max" ||
		line.ProductName != "آیفون ۱۵ پرو مکس" || line.UnitPriceIrr != 489_900_000 ||
		line.LineTotalIrr != 979_800_000 {
		t.Errorf("line wrong: %+v", line)
	}
	if line.MaxQuantity != 7 || cart.SubtotalIrr != 979_800_000 || cart.Currency != "IRR" {
		t.Errorf("cart totals wrong: subtotal=%d max=%d currency=%s",
			cart.SubtotalIrr, line.MaxQuantity, cart.Currency)
	}

	// The issued token resolves to a cart with identical content.
	again, err := svc.GetCart(ctx, token, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if len(again.Items) != 1 || again.Items[0].Quantity != 2 {
		t.Errorf("persisted cart wrong: %+v", again)
	}
}

func TestAddIncrementsExistingLine(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()
	variant := variantIDBySKU(t, db, "IP15PM-NAT-256")

	cart, token, err := svc.AddItem(ctx, "", variant, 2, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem 1: %v", err)
	}
	cart, _, err = svc.AddItem(ctx, token, variant, 1, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem 2: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("lines = %d, want 1 (re-add increments, never duplicates)", len(cart.Items))
	}
	if cart.Items[0].Quantity != 3 || cart.SubtotalIrr != 1_469_700_000 {
		t.Errorf("qty=%d subtotal=%d, want 3 / 1469700000",
			cart.Items[0].Quantity, cart.SubtotalIrr)
	}
}

func TestAddStockExceededRollsBack(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()
	variant := variantIDBySKU(t, db, "IP15PM-NAT-256") // stock 7

	cart, token, err := svc.AddItem(ctx, "", variant, 5, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem 5: %v", err)
	}
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 5 {
		t.Fatalf("first write wrong: %+v", cart)
	}

	cart, newToken, err := svc.AddItem(ctx, token, variant, 5, i18n.LocaleFa)
	var stockErr *StockExceeded
	if !asStockExceeded(err, &stockErr) {
		t.Fatalf("err = %v, want StockExceeded", err)
	}
	if stockErr.Requested != 10 || stockErr.Available != 7 || stockErr.VariantID != variant {
		t.Errorf("details wrong: %+v", stockErr)
	}
	if newToken != "" {
		t.Error("failed write must not issue a session token")
	}
	_ = cart

	// The failed write rolled back: the line stays at 5.
	var qty int
	if err := db.QueryRowContext(ctx,
		"SELECT quantity FROM cart_items").Scan(&qty); err != nil {
		t.Fatalf("line count: %v", err)
	}
	if qty != 5 {
		t.Errorf("line quantity after failed write = %d, want 5 (rollback)", qty)
	}
}

func asStockExceeded(err error, target **StockExceeded) bool {
	if e, ok := err.(*StockExceeded); ok {
		*target = e
		return true
	}
	return false
}

func TestAddValidationErrors(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()

	// Unknown variant.
	_, _, err := svc.AddItem(ctx, "", 999_999, 1, i18n.LocaleFa)
	var validation *ValidationError
	if !asValidationError(err, &validation) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if len(validation.Fields) != 1 || validation.Fields[0].Field != "variantId" ||
		validation.Fields[0].Code != "VARIANT_NOT_FOUND" {
		t.Errorf("fields = %+v", validation.Fields)
	}

	// Invalid quantity.
	_, _, err = svc.AddItem(ctx, "", variantIDBySKU(t, db, "IP15-PNK-128"), 100, i18n.LocaleFa)
	if !asValidationError(err, &validation) || validation.Fields[0].Code != "INVALID_QUANTITY" {
		t.Errorf("quantity 100: err = %v, want INVALID_QUANTITY", err)
	}
	_, _, err = svc.AddItem(ctx, "", variantIDBySKU(t, db, "IP15-PNK-128"), 0, i18n.LocaleFa)
	if !asValidationError(err, &validation) || validation.Fields[0].Code != "INVALID_QUANTITY" {
		t.Errorf("quantity 0: err = %v, want INVALID_QUANTITY", err)
	}

	// Zero-stock variant: adding 1 exceeds stock 0.
	se := variantIDBySKU(t, db, "IPSE3-WHT-128")
	_, _, err = svc.AddItem(ctx, "", se, 1, i18n.LocaleFa)
	var stockErr *StockExceeded
	if !asStockExceeded(err, &stockErr) || stockErr.Available != 0 || stockErr.Requested != 1 {
		t.Errorf("zero-stock variant: err = %v, want StockExceeded{1, 0}", err)
	}
}

func asValidationError(err error, target **ValidationError) bool {
	if e, ok := err.(*ValidationError); ok {
		*target = e
		return true
	}
	return false
}

func TestUpdateItemAndRemoveItem(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()
	variant := variantIDBySKU(t, db, "IP15PM-NAT-256") // stock 7

	cart, token, err := svc.AddItem(ctx, "", variant, 2, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	lineID := cart.Items[0].ID

	// Valid update.
	cart, err = svc.UpdateItem(ctx, token, lineID, 5, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if cart.Items[0].Quantity != 5 || cart.Items[0].LineTotalIrr != 2_449_500_000 {
		t.Errorf("updated line wrong: %+v", cart.Items[0])
	}

	// Beyond stock → rollback.
	if _, err = svc.UpdateItem(ctx, token, lineID, 8, i18n.LocaleFa); !errors.As(err, new(*StockExceeded)) {
		t.Fatalf("UpdateItem 8: err = %v, want StockExceeded", err)
	}
	cart, err = svc.GetCart(ctx, token, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if cart.Items[0].Quantity != 5 {
		t.Errorf("quantity after failed update = %d, want 5", cart.Items[0].Quantity)
	}

	// Unknown line and missing session.
	if _, err = svc.UpdateItem(ctx, token, 424242, 1, i18n.LocaleFa); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown line: err = %v, want ErrNotFound", err)
	}
	if _, err = svc.UpdateItem(ctx, "", lineID, 1, i18n.LocaleFa); !errors.Is(err, ErrNotFound) {
		t.Errorf("no session: err = %v, want ErrNotFound", err)
	}

	// Invalid quantity.
	if _, err = svc.UpdateItem(ctx, token, lineID, 0, i18n.LocaleFa); !errors.As(err, new(*ValidationError)) {
		t.Errorf("quantity 0: err = %v, want ValidationError", err)
	}

	// Removal, then removal again.
	cart, err = svc.RemoveItem(ctx, token, lineID, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("RemoveItem: %v", err)
	}
	if len(cart.Items) != 0 || cart.SubtotalIrr != 0 {
		t.Errorf("cart after removal = %+v, want empty", cart)
	}
	if _, err = svc.RemoveItem(ctx, token, lineID, i18n.LocaleFa); !errors.Is(err, ErrNotFound) {
		t.Errorf("remove twice: err = %v, want ErrNotFound", err)
	}
}

func TestGetCartWithoutOrWithInvalidSession(t *testing.T) {
	svc, sessions, db := testCartService(t)
	ctx := context.Background()

	empty, err := svc.GetCart(ctx, "", i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetCart no session: %v", err)
	}
	if len(empty.Items) != 0 || empty.SubtotalIrr != 0 || empty.ItemCount != 0 || empty.Currency != "IRR" {
		t.Errorf("empty cart wrong: %+v", empty)
	}

	got, err := svc.GetCart(ctx, "invalid-token", i18n.LocaleFa)
	if err != nil || len(got.Items) != 0 {
		t.Errorf("GetCart(invalid) = %v, %v; want empty cart, nil error (never 401)", got, err)
	}

	// Expired session → empty cart as well.
	token, id, err := sessions.Create(ctx)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = db.ExecContext(ctx,
		"UPDATE guest_sessions SET expires_at = ? WHERE id = ?",
		time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), id); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	got, err = svc.GetCart(ctx, token, i18n.LocaleFa)
	if err != nil || len(got.Items) != 0 {
		t.Errorf("GetCart(expired) = %v, %v; want empty cart (never 401)", got, err)
	}
}

func TestTotalsRecomputedFromCurrentPrices(t *testing.T) {
	svc, _, db := testCartService(t)
	ctx := context.Background()
	variant := variantIDBySKU(t, db, "IP15PM-NAT-256")

	cart, token, err := svc.AddItem(ctx, "", variant, 2, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if cart.SubtotalIrr != 979_800_000 {
		t.Fatalf("initial subtotal = %d", cart.SubtotalIrr)
	}

	// Change the DB price (simulating a catalog update); compare-at
	// must move too — the CHECK requires compare_at ≥ price.
	if _, err = db.ExecContext(ctx,
		"UPDATE product_variants SET price_irr = 600_000_000, compare_at_price_irr = NULL WHERE id = ?", variant); err != nil {
		t.Fatalf("price update: %v", err)
	}

	cart, err = svc.GetCart(ctx, token, i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if cart.Items[0].UnitPriceIrr != 600_000_000 || cart.SubtotalIrr != 1_200_000_000 ||
		cart.Items[0].LineTotalIrr != 1_200_000_000 {
		t.Errorf("totals not recomputed from DB price: %+v subtotal=%d",
			cart.Items[0], cart.SubtotalIrr)
	}
}
