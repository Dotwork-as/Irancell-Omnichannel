package catalog_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/server"
	"backend/internal/seed"
)

// newTestApp builds the full server (middleware chain + catalog
// routes) against a freshly seeded temporary database.
func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	cfg, err := config.Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
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
	return server.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db)
}

func getJSON(t *testing.T, app *fiber.App, path, acceptLanguage string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body %q is not JSON: %v", body, err)
	}
	return resp, payload
}

func TestProductsEndpoint(t *testing.T) {
	app := newTestApp(t)

	resp, payload := getJSON(t, app, "/api/v1/products", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	items, _ := payload["items"].([]any)
	if len(items) != 6 {
		t.Fatalf("items = %d, want 6", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["slug"] != "iphone-15-pro-max" {
		t.Errorf("first slug = %v, want iphone-15-pro-max", first["slug"])
	}
	for _, key := range []string{"name", "defaultVariantId", "colorKey", "colorName",
		"storageGb", "priceIrr", "compareAtPriceIrr", "inStock", "rating", "reviewCount"} {
		if _, ok := first[key]; !ok {
			t.Errorf("ProductSummary missing key %q in %v", key, first)
		}
	}
	if _, hasImage := first["image"]; hasImage {
		t.Error("response must not carry image data (frontend-managed)")
	}
}

func TestProductsSearchEndpoint(t *testing.T) {
	app := newTestApp(t)

	resp, payload := getJSON(t, app, "/api/v1/products?q=IP14", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	items, _ := payload["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("search IP14 items = %d, want 2", len(items))
	}
}

func TestProductsEndpointLocalizedEnglish(t *testing.T) {
	app := newTestApp(t)

	_, payload := getJSON(t, app, "/api/v1/products", "en-US,en;q=0.9")
	items, _ := payload["items"].([]any)
	first, _ := items[0].(map[string]any)
	if first["name"] != "iPhone 15 Pro Max" || first["colorName"] != "Natural Titanium" {
		t.Errorf("en response not consistent: %v", first)
	}
}

func TestProductDetailEndpoint(t *testing.T) {
	app := newTestApp(t)

	resp, payload := getJSON(t, app, "/api/v1/products/iphone-15-pro-max", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	variants, _ := payload["variants"].([]any)
	if len(variants) != 16 {
		t.Fatalf("variants = %d, want 16", len(variants))
	}
	var zeroStockSeen bool
	for _, v := range variants {
		m, _ := v.(map[string]any)
		if m["sku"] == "IP15PM-WHT-1TB" {
			zeroStockSeen = m["stockQty"].(float64) == 0 && m["inStock"] == false
		}
	}
	if !zeroStockSeen {
		t.Error("white 1TB variant must stay visible with stockQty 0 / inStock false")
	}
	specs, _ := payload["specs"].([]any)
	reviews, _ := payload["reviews"].([]any)
	if len(specs) != 6 || len(reviews) != 3 {
		t.Errorf("specs = %d, reviews = %d, want 6 and 3", len(specs), len(reviews))
	}
}

func TestProductDetailLanguageConsistency(t *testing.T) {
	app := newTestApp(t)

	_, payload := getJSON(t, app, "/api/v1/products/iphone-15-pro-max", "en")
	if payload["name"] != "iPhone 15 Pro Max" {
		t.Errorf("name = %v, want en", payload["name"])
	}
	specs, _ := payload["specs"].([]any)
	first, _ := specs[0].(map[string]any)
	if first["label"] != "Chip" {
		t.Errorf("spec label = %v, want en", first["label"])
	}
	variants, _ := payload["variants"].([]any)
	v0, _ := variants[0].(map[string]any)
	if v0["colorName"] == "تیتانیوم طبیعی" {
		t.Error("variant colorName still fa in an en response — language must be consistent")
	}
}

func TestProductDetailUnknownSlug(t *testing.T) {
	app := newTestApp(t)

	// Default locale (fa): Persian message.
	resp, payload := getJSON(t, app, "/api/v1/products/no-such-phone", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", errObj["code"])
	}
	if msg, _ := errObj["message"].(string); msg != "منبع درخواست‌شده پیدا نشد." {
		t.Errorf("fa message = %q", msg)
	}

	// English locale: English message for the same error.
	resp, payload = getJSON(t, app, "/api/v1/products/no-such-phone", "en")
	errObj, _ = payload["error"].(map[string]any)
	if msg, _ := errObj["message"].(string); msg != "The requested resource was not found." {
		t.Errorf("en message = %q", msg)
	}
}

func TestCatalogReadsNeedNoOrigin(t *testing.T) {
	app := newTestApp(t)

	// GETs carry no Origin header in the wild; they must pass the
	// Origin validation (writes-only rule).
	resp, _ := getJSON(t, app, "/api/v1/products", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET without Origin status = %d, want 200", resp.StatusCode)
	}
}
