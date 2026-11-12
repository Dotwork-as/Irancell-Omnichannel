package checkout_test

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

const origin = "http://localhost:3000"

func newTestApp(t *testing.T) (*fiber.App, *database.DB) {
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
	return server.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db), db
}

func do(t *testing.T, app *fiber.App, method, path, body, cookie, acceptLanguage, originHeader string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if originHeader != "" {
		req.Header.Set("Origin", originHeader)
	}
	if cookie != "" {
		req.Header.Set("Cookie", "guest_sid="+cookie)
	}
	if acceptLanguage != "" {
		req.Header.Set("Accept-Language", acceptLanguage)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func payload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var p map[string]any
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("body %q is not JSON: %v", raw, err)
	}
	return p
}

func guestCookie(t *testing.T, resp *http.Response) string {
	t.Helper()
	for _, c := range resp.Cookies() {
		if c.Name == "guest_sid" {
			return c.Value
		}
	}
	t.Fatal("no guest_sid cookie in response")
	return ""
}

func validBody() string {
	return `{"recipient":{"firstName":"نرگس","lastName":"رضایی","phone":"09123456789","address":"تهران، خیابان نمونه، پلاک ۱۰"},"shippingMethodId":"express"}`
}

func TestShippingMethodsEndpoint(t *testing.T) {
	app, _ := newTestApp(t)

	// fa default.
	resp := do(t, app, http.MethodGet, "/api/v1/shipping-methods", "", "", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	items := payload(t, resp)["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	first := items[0].(map[string]any)
	if first["id"] != "express" || first["name"] != "پیک اکسپرس تهران" || first["priceIrr"].(float64) != 490_000 {
		t.Errorf("fa express wrong: %v", first)
	}

	// en localized.
	resp = do(t, app, http.MethodGet, "/api/v1/shipping-methods", "", "", "en", "")
	first = payload(t, resp)["items"].([]any)[0].(map[string]any)
	if first["name"] != "Tehran express courier" {
		t.Errorf("en express name = %v", first["name"])
	}
}

// Full guest journey over HTTP: cart → checkout → paid order.
func TestCheckoutEndpointEndToEnd(t *testing.T) {
	app, db := newTestApp(t)
	ctx := context.Background()
	variantID := int64(2) // IP15PM-NAT-256 per seeds (id ordering)

	// Fill the cart.
	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variantID)+`,"quantity":2}`, "", "", origin)
	token := guestCookie(t, resp)

	// Checkout.
	resp = do(t, app, http.MethodPost, "/api/v1/checkout", validBody(), token, "", origin)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout status = %d, want 201", resp.StatusCode)
	}
	r := payload(t, resp)
	if r["status"] != "paid" || r["currency"] != "IRR" {
		t.Errorf("receipt header wrong: %v", r)
	}
	if tc, _ := r["trackingCode"].(string); !strings.HasPrefix(tc, "IRC-") {
		t.Errorf("trackingCode = %v, want IRC- prefix", tc)
	}
	payment := r["payment"].(map[string]any)
	if payment["method"] != "mock" || !strings.HasPrefix(payment["transactionId"].(string), "TXN-") {
		t.Errorf("payment wrong: %v", payment)
	}
	if r["totalIrr"].(float64) != 2*489_900_000+490_000 {
		t.Errorf("total = %v (subtotal 2×489900000 + express 490000)", r["totalIrr"])
	}
	recipient := r["recipient"].(map[string]any)
	if recipient["phone"] != "09123456789" {
		t.Errorf("recipient echo wrong: %v", recipient)
	}

	// Cart is emptied after checkout.
	resp = do(t, app, http.MethodGet, "/api/v1/cart", "", token, "", "")
	if got := len(payload(t, resp)["items"].([]any)); got != 0 {
		t.Errorf("cart after checkout has %d items, want 0", got)
	}

	// Stock deducted: 7 − 2 = 5.
	var stock int
	if err := db.QueryRowContext(ctx,
		"SELECT stock_qty FROM product_variants WHERE id = ?", variantID).Scan(&stock); err != nil {
		t.Fatalf("stock lookup: %v", err)
	}
	if stock != 5 {
		t.Errorf("stock = %d, want 5", stock)
	}

	// Exactly one real paid order, not a demo.
	var status string
	var isDemo int
	if err := db.QueryRowContext(ctx,
		"SELECT status, is_demo FROM orders WHERE is_demo = 0").Scan(&status, &isDemo); err != nil {
		t.Fatalf("order lookup: %v", err)
	}
	if status != "paid" || isDemo != 0 {
		t.Errorf("order status=%s demo=%d, want paid/0", status, isDemo)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [21]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestCheckoutErrorsLocalized(t *testing.T) {
	app, _ := newTestApp(t)

	// Invalid phone → 400 with per-field detail (en).
	resp := do(t, app, http.MethodPost, "/api/v1/checkout",
		`{"recipient":{"firstName":"نرگس","lastName":"رضایی","phone":"09123","address":"تهران، خیابان نمونه، پلاک ۱۰"},"shippingMethodId":"express"}`,
		"", "en", origin)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	errObj := payload(t, resp)["error"].(map[string]any)
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Fatalf("code = %v, want VALIDATION_ERROR", errObj["code"])
	}
	detail := errObj["details"].([]any)[0].(map[string]any)
	if detail["field"] != "recipient.phone" || detail["code"] != "INVALID_PHONE" ||
		detail["message"] != "Phone must be a valid Iranian mobile number with 11 digits (09xxxxxxxxx)." {
		t.Errorf("phone detail wrong: %v", detail)
	}

	// Empty cart (no cookie) → 409 CART_EMPTY, localized fa by default.
	resp = do(t, app, http.MethodPost, "/api/v1/checkout", validBody(), "", "", origin)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("empty cart status = %d, want 409", resp.StatusCode)
	}
	errObj = payload(t, resp)["error"].(map[string]any)
	if errObj["code"] != "CART_EMPTY" || errObj["message"] != "سبد خرید خالی است." {
		t.Errorf("CART_EMPTY envelope wrong: %v", errObj)
	}

	// Unknown shipping method → 400 with shipping field detail.
	// (Requires a cart first so validation order doesn't mask it.)
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":2,"quantity":1}`, "", "", origin)
	token := guestCookie(t, resp)
	body := strings.Replace(validBody(), `"express"`, `"drone"`, 1)
	resp = do(t, app, http.MethodPost, "/api/v1/checkout", body, token, "", origin)
	detail = payload(t, resp)["error"].(map[string]any)["details"].([]any)[0].(map[string]any)
	if detail["field"] != "shippingMethodId" || detail["code"] != "SHIPPING_METHOD_NOT_FOUND" {
		t.Errorf("shipping detail wrong: %v", detail)
	}

	// Checkout requires Origin like every state-changing request.
	resp = do(t, app, http.MethodPost, "/api/v1/checkout", validBody(), token, "", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("no-origin status = %d, want 403", resp.StatusCode)
	}
	if code := payload(t, resp)["error"].(map[string]any)["code"]; code != "FORBIDDEN_ORIGIN" {
		t.Errorf("code = %v, want FORBIDDEN_ORIGIN", code)
	}
}

// Error envelopes must never leak internal details.
func TestNoInternalDetailLeakage(t *testing.T) {
	app, _ := newTestApp(t)

	resp := do(t, app, http.MethodGet, "/api/v1/products/does-not-exist", "", "", "", "")
	body := payload(t, resp)
	raw, _ := json.Marshal(body)
	for _, leak := range []string{"sqlite", "SQL", "SELECT", "table", "database"} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(leak)) {
			t.Errorf("error body leaks internal keyword %q: %s", leak, raw)
		}
	}

	// Malformed checkout body → VALIDATION_ERROR, not a stack trace.
	resp = do(t, app, http.MethodPost, "/api/v1/checkout", `{oops`, "", "", origin)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed status = %d, want 400", resp.StatusCode)
	}
	if code := payload(t, resp)["error"].(map[string]any)["code"]; code != "VALIDATION_ERROR" {
		t.Errorf("malformed code = %v", code)
	}

	// SQL-injection attempts in search and fields stay inert.
	resp = do(t, app, http.MethodGet, `/api/v1/products?q=%27%20OR%201%3D1--`, "", "", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("injection probe search status = %d, want 200", resp.StatusCode)
	}
	if items := payload(t, resp)["items"].([]any); len(items) != 0 {
		t.Errorf("injection probe matched %d items, want 0", len(items))
	}
}

// ─── Turnstile human-verification matrix (fail-closed) ──────────

func turnstileApp(t *testing.T, verifyURL string) (*fiber.App, *database.DB) {
	t.Helper()
	cfg, err := config.Load(func(k string) string {
		switch k {
		case "TURNSTILE_SECRET_KEY":
			return "test-secret"
		case "TURNSTILE_VERIFY_URL":
			return verifyURL
		}
		return ""
	})
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
	return server.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db), db
}

// Stubs the Cloudflare siteverify endpoint: token "good" succeeds,
// everything else fails.
func startTurnstileStub(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		success := r.FormValue("response") == "good"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"success": success})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func checkoutWithToken(t *testing.T, app *fiber.App, cookie, token string) *http.Response {
	t.Helper()
	body := `{"recipient":{"firstName":"نرگس","lastName":"رضایی","phone":"09123456789","address":"تهران، خیابان نمونه، پلاک ۱۰"},"shippingMethodId":"express"`
	if token != "" {
		body += `,"turnstileToken":"` + token + `"`
	}
	body += "}"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	req.Header.Set("Cookie", "guest_sid="+cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func TestTurnstileMatrix(t *testing.T) {
	stub := startTurnstileStub(t)
	app, db := turnstileApp(t, stub)
	variant := variantID(t, db, "IP15PM-NAT-256")

	// Baseline: fill a cart and capture the session cookie.
	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":1}`, "", "", origin)
	cookie := guestCookie(t, resp)

	// Missing token (verification configured) → 400 TURNSTILE_REQUIRED.
	resp = checkoutWithToken(t, app, cookie, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing token: status = %d, want 400", resp.StatusCode)
	}
	errObj := payload(t, resp)["error"].(map[string]any)
	if errObj["code"] != "TURNSTILE_REQUIRED" {
		t.Errorf("code = %v, want TURNSTILE_REQUIRED", errObj["code"])
	}

	// Invalid token → 403 TURNSTILE_FAILED, fail-closed.
	resp = checkoutWithToken(t, app, cookie, "invalid-token")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("invalid token: status = %d, want 403", resp.StatusCode)
	}
	errObj = payload(t, resp)["error"].(map[string]any)
	if errObj["code"] != "TURNSTILE_FAILED" {
		t.Errorf("code = %v, want TURNSTILE_FAILED", errObj["code"])
	}

	// Expired/unsuccessful verifier responses → also fail-closed.
	resp = checkoutWithToken(t, app, cookie, "expired-token")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expired token: status = %d, want 403", resp.StatusCode)
	}

	// Cart untouched after every failure.
	cart := payload(t, do(t, app, http.MethodGet, "/api/v1/cart", "", cookie, "", ""))
	if got := len(cart["items"].([]any)); got != 1 {
		t.Errorf("cart after failures has %d items, want 1 (untouched)", got)
	}

	// Valid token → checkout proceeds to 201 paid.
	resp = checkoutWithToken(t, app, cookie, "good")
	if resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("valid token: status = %d, body = %s", resp.StatusCode, raw)
	}
	if payload(t, resp)["status"] != "paid" {
		t.Error("valid token did not produce a paid order")
	}
}

// Without TURNSTILE_SECRET_KEY configured, verification is disabled:
// checkout proceeds without any token (local MVP behavior preserved).
func TestTurnstileDisabledByDefault(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256")

	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":1}`, "", "", origin)
	cookie := guestCookie(t, resp)

	resp = checkoutWithToken(t, app, cookie, "")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (verification disabled)", resp.StatusCode)
	}
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
