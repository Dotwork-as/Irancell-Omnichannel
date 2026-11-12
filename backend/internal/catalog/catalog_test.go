package catalog

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"backend/internal/platform/database"
	"backend/internal/platform/i18n"
	"backend/internal/seed"
)

func testService(t *testing.T) *Service {
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
	return New(db)
}

func find(items []ProductSummary, slug string) *ProductSummary {
	for i := range items {
		if items[i].Slug == slug {
			return &items[i]
		}
	}
	return nil
}

func TestListIncludesUnavailableProducts(t *testing.T) {
	svc := testService(t)

	items, err := svc.List(context.Background(), "", i18n.LocaleFa)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 6 {
		t.Fatalf("items = %d, want 6 (unavailable products included)", len(items))
	}

	mvp := find(items, "iphone-15-pro-max")
	if mvp == nil {
		t.Fatal("iphone-15-pro-max missing from list")
	}
	if mvp.Name != "آیفون ۱۵ پرو مکس" || mvp.ColorKey != "natural" ||
		mvp.ColorName != "تیتانیوم طبیعی" || mvp.StorageGb != 256 {
		t.Errorf("default variant fields wrong: %+v", mvp)
	}
	if mvp.PriceIrr != 489_900_000 || mvp.CompareAtPriceIrr == nil || *mvp.CompareAtPriceIrr != 525_000_000 {
		t.Errorf("prices wrong: price=%d compare=%v", mvp.PriceIrr, mvp.CompareAtPriceIrr)
	}
	if !mvp.InStock || mvp.Rating != 4.8 || mvp.ReviewCount != 246 || mvp.DefaultVariantID == 0 {
		t.Errorf("summary flags wrong: inStock=%v rating=%v reviews=%d defaultId=%d",
			mvp.InStock, mvp.Rating, mvp.ReviewCount, mvp.DefaultVariantID)
	}

	se := find(items, "iphone-se-3")
	if se == nil {
		t.Fatal("iphone-se-3 (fully unavailable) must still be listed")
	}
	if se.InStock {
		t.Error("iphone-se-3 inStock = true, want false (all variants zero stock)")
	}
}

func TestListSearch(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	cases := []struct {
		q    string
		want []string
	}{
		{"پرو مکس", []string{"iphone-15-pro-max"}}, // fa product name
		{"Pro Max", []string{"iphone-15-pro-max"}}, // en product name
		{"iPhone", []string{"iphone-15-pro-max", "iphone-15-pro", "iphone-15", "iphone-14-plus", "iphone-14", "iphone-se-3"}},
		{"titanium", []string{"iphone-15-pro-max", "iphone-15-pro"}}, // en color name
		{"صورتی", []string{"iphone-15"}},                             // fa color name
		{"Blue", []string{"iphone-15-pro-max", "iphone-15-pro", "iphone-14-plus"}},
		{"IP14", []string{"iphone-14-plus", "iphone-14"}},       // sku
		{"256", []string{"iphone-15-pro-max", "iphone-15-pro"}}, // storage value
		{"SE", []string{"iphone-se-3"}},                         // matches unavailable product
		{"چیزی‌که‌وجود‌ندارد", nil},                             // no match → empty list
	}

	for _, tc := range cases {
		items, err := svc.List(ctx, tc.q, i18n.LocaleFa)
		if err != nil {
			t.Fatalf("List(%q): %v", tc.q, err)
		}
		if len(items) != len(tc.want) {
			t.Errorf("List(%q) = %d items, want %d", tc.q, len(items), len(tc.want))
			continue
		}
		for i, slug := range tc.want {
			if items[i].Slug != slug {
				t.Errorf("List(%q)[%d] = %s, want %s", tc.q, i, items[i].Slug, slug)
			}
		}
	}
}

func TestListLocaleEnglish(t *testing.T) {
	svc := testService(t)

	items, err := svc.List(context.Background(), "", i18n.LocaleEn)
	if err != nil {
		t.Fatalf("List en: %v", err)
	}
	mvp := find(items, "iphone-15-pro-max")
	if mvp.Name != "iPhone 15 Pro Max" || mvp.ColorName != "Natural Titanium" {
		t.Errorf("en localization wrong: name=%q color=%q", mvp.Name, mvp.ColorName)
	}
}

func TestGetBySlugDetail(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()

	d, err := svc.GetBySlug(ctx, "iphone-15-pro-max", i18n.LocaleFa)
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if d.Name != "آیفون ۱۵ پرو مکس" || d.Rating != 4.8 || d.ReviewCount != 246 {
		t.Errorf("detail header wrong: %+v", d)
	}
	if len(d.Variants) != 16 {
		t.Fatalf("variants = %d, want 16 (only DB-defined variants)", len(d.Variants))
	}

	var white1tb, nat256 *Variant
	for i := range d.Variants {
		switch d.Variants[i].SKU {
		case "IP15PM-WHT-1TB":
			white1tb = &d.Variants[i]
		case "IP15PM-NAT-256":
			nat256 = &d.Variants[i]
		}
	}
	if white1tb == nil || nat256 == nil {
		t.Fatal("expected variants missing")
	}
	if white1tb.InStock || white1tb.StockQty != 0 {
		t.Errorf("white 1TB: inStock=%v stock=%d, want false/0 (unavailable variants preserved)", white1tb.InStock, white1tb.StockQty)
	}
	if !nat256.InStock || nat256.PriceIrr != 489_900_000 || nat256.StockQty != 7 {
		t.Errorf("natural 256GB variant wrong: %+v", nat256)
	}
	if nat256.ID != d.DefaultVariantID {
		t.Errorf("defaultVariantID = %d, want variant %s (%d)", d.DefaultVariantID, nat256.SKU, nat256.ID)
	}

	if len(d.Specs) != 6 {
		t.Fatalf("specs = %d, want 6", len(d.Specs))
	}
	if d.Specs[0].Label != "تراشه" || d.Specs[0].Value != "Apple A17 Pro" {
		t.Errorf("first spec = %+v, want the seeded chip row in order", d.Specs[0])
	}
	if len(d.Reviews) != 3 {
		t.Fatalf("reviews = %d, want 3", len(d.Reviews))
	}
	if d.Reviews[0].Author != "سارا محمدی" || d.Reviews[0].Rating != 5 {
		t.Errorf("first review wrong: %+v", d.Reviews[0])
	}
}

func TestGetBySlugDetailEnglish(t *testing.T) {
	svc := testService(t)

	d, err := svc.GetBySlug(context.Background(), "iphone-15-pro-max", i18n.LocaleEn)
	if err != nil {
		t.Fatalf("GetBySlug en: %v", err)
	}
	if d.Name != "iPhone 15 Pro Max" {
		t.Errorf("name = %q, want en", d.Name)
	}
	if d.Specs[0].Label != "Chip" {
		t.Errorf("spec label = %q, want en", d.Specs[0].Label)
	}
	if d.Reviews[0].Author != "Sara Mohammadi" {
		t.Errorf("review author = %q, want en", d.Reviews[0].Author)
	}
	found := false
	for _, v := range d.Variants {
		if v.SKU == "IP15PM-NAT-256" {
			found = v.ColorName == "Natural Titanium"
		}
	}
	if !found {
		t.Error("variant colorName not localized to en")
	}
}

func TestGetBySlugUnknown(t *testing.T) {
	svc := testService(t)

	if _, err := svc.GetBySlug(context.Background(), "no-such-phone", i18n.LocaleFa); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
