package cart_test

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
	"time"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/server"
	"backend/internal/seed"
	"backend/internal/session"
)

const origin = "http://localhost:3000"

// newTestApp builds the full server (middleware chain + catalog +
// cart routes) against a freshly seeded temporary database and hands
// the database back for seed lookups.
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

func variantID(t *testing.T, db *database.DB, sku string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRowContext(context.Background(),
		"SELECT id FROM product_variants WHERE sku = ?", sku).Scan(&id); err != nil {
		t.Fatalf("variant %s: %v", sku, err)
	}
	return id
}

// do performs a request; origin "" means the header is omitted (the
// natural browser-less case for reads).
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
		req.Header.Set("Cookie", session.CookieName+"="+cookie)
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
		if c.Name == session.CookieName {
			return c.Value
		}
	}
	t.Fatal("no guest_sid cookie in response")
	return ""
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestGetCartWithoutSession(t *testing.T) {
	app, _ := newTestApp(t)

	resp := do(t, app, http.MethodGet, "/api/v1/cart", "", "", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	p := payload(t, resp)
	if len(p["items"].([]any)) != 0 || p["subtotalIrr"].(float64) != 0 ||
		p["itemCount"].(float64) != 0 || p["currency"] != "IRR" {
		t.Errorf("empty cart wrong: %v", p)
	}
	for _, c := range resp.Cookies() {
		if c.Name == session.CookieName {
			t.Error("GET cart must never set a session cookie")
		}
	}
}

func TestAddItemIssuesApprovedCookie(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256")

	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":2}`, "", "", origin)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var guest *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == session.CookieName {
			guest = c
		}
	}
	if guest == nil {
		t.Fatal("first successful write must set the guest_sid cookie")
	}
	if !guest.HttpOnly || !guest.Secure || guest.Path != "/" || guest.MaxAge != session.CookieMaxAge {
		t.Errorf("cookie flags wrong: %+v", guest)
	}
	// SameSite and the full attribute set are asserted on the raw
	// Set-Cookie header (fiber serializes attributes lowercase and
	// net/http does not reliably round-trip SameSite on parsed
	// cookies).
	setCookie := strings.ToLower(resp.Header.Get("Set-Cookie"))
	for _, attr := range []string{"httponly", "secure", "samesite=lax", "path=/", "max-age=2592000"} {
		if !strings.Contains(setCookie, attr) {
			t.Errorf("Set-Cookie %q missing attribute %q", setCookie, attr)
		}
	}
	if len(guest.Value) < 40 {
		t.Errorf("token too short (%d chars), want 256-bit entropy", len(guest.Value))
	}

	p := payload(t, resp)
	items := p["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	line := items[0].(map[string]any)
	if line["productName"] != "آیفون ۱۵ پرو مکس" || line["unitPriceIrr"].(float64) != 489_900_000 ||
		line["quantity"].(float64) != 2 || line["lineTotalIrr"].(float64) != 979_800_000 ||
		line["maxQuantity"].(float64) != 7 {
		t.Errorf("line wrong: %v", line)
	}
	if p["subtotalIrr"].(float64) != 979_800_000 || p["itemCount"].(float64) != 2 || p["currency"] != "IRR" {
		t.Errorf("cart totals wrong: %v", p)
	}
}

func TestAddThenIncrementAndIsolation(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256")

	// First write → cookie.
	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":2}`, "", "", origin)
	token := guestCookie(t, resp)

	// Second write with the same cookie increments the same line.
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":1}`, token, "", origin)
	p := payload(t, resp)
	items := p["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1 (incremented, not duplicated)", len(items))
	}
	line := items[0].(map[string]any)
	if line["quantity"].(float64) != 3 || line["lineTotalIrr"].(float64) != 1_469_700_000 {
		t.Errorf("incremented line wrong: %v", line)
	}

	// Another guest session sees an empty cart (session isolation).
	resp = do(t, app, http.MethodGet, "/api/v1/cart", "", "", "", "")
	if got := len(payload(t, resp)["items"].([]any)); got != 0 {
		t.Errorf("stranger's cart has %d items, want 0", got)
	}
	// The owner still sees theirs.
	resp = do(t, app, http.MethodGet, "/api/v1/cart", "", token, "", "")
	if got := len(payload(t, resp)["items"].([]any)); got != 1 {
		t.Errorf("owner's cart has %d items, want 1", got)
	}
}

func TestAddStockExceededAndRollback(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256") // stock 7

	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":5}`, "", "", origin)
	token := guestCookie(t, resp)

	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":5}`, token, "", origin)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
	p := payload(t, resp)
	errObj := p["error"].(map[string]any)
	if errObj["code"] != "STOCK_UNAVAILABLE" {
		t.Errorf("code = %v, want STOCK_UNAVAILABLE", errObj["code"])
	}
	details := errObj["details"].([]any)[0].(map[string]any)
	if details["requested"].(float64) != 10 || details["available"].(float64) != 7 ||
		details["variantId"].(float64) != float64(variant) {
		t.Errorf("stock details wrong: %v", details)
	}

	// The failed write rolled back — the line stays at 5.
	resp = do(t, app, http.MethodGet, "/api/v1/cart", "", token, "", "")
	line := payload(t, resp)["items"].([]any)[0].(map[string]any)
	if line["quantity"].(float64) != 5 {
		t.Errorf("quantity after rollback = %v, want 5", line["quantity"])
	}
}

func TestCartValidationErrorsLocalized(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256")

	// Invalid quantity (fa default).
	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":0}`, "", "", origin)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	errObj := payload(t, resp)["error"].(map[string]any)
	if errObj["code"] != "VALIDATION_ERROR" {
		t.Errorf("code = %v, want VALIDATION_ERROR", errObj["code"])
	}
	detail := errObj["details"].([]any)[0].(map[string]any)
	if detail["field"] != "quantity" || detail["code"] != "INVALID_QUANTITY" {
		t.Errorf("detail wrong: %v", detail)
	}
	if msg, _ := detail["message"].(string); msg != "تعداد باید عددی بین ۱ تا ۹۹ باشد." {
		t.Errorf("fa message = %q", msg)
	}

	// English locale → English message, same envelope shape.
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":0}`, "", "en", origin)
	detail = payload(t, resp)["error"].(map[string]any)["details"].([]any)[0].(map[string]any)
	if msg, _ := detail["message"].(string); msg != "Quantity must be a number between 1 and 99." {
		t.Errorf("en message = %q", msg)
	}

	// Unknown variant.
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":999999,"quantity":1}`, "", "", origin)
	detail = payload(t, resp)["error"].(map[string]any)["details"].([]any)[0].(map[string]any)
	if detail["code"] != "VARIANT_NOT_FOUND" {
		t.Errorf("unknown variant detail = %v", detail)
	}

	// Malformed JSON body → VALIDATION_ERROR via the error handler.
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items", `{not json`, "", "", origin)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed body status = %d, want 400", resp.StatusCode)
	}
	if code := payload(t, resp)["error"].(map[string]any)["code"]; code != "VALIDATION_ERROR" {
		t.Errorf("malformed body code = %v, want VALIDATION_ERROR", code)
	}
}

func TestCartWritesRequireOrigin(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256")

	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":1}`, "", "", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
	if code := payload(t, resp)["error"].(map[string]any)["code"]; code != "FORBIDDEN_ORIGIN" {
		t.Errorf("code = %v, want FORBIDDEN_ORIGIN", code)
	}
	for _, c := range resp.Cookies() {
		if c.Name == session.CookieName {
			t.Error("rejected write must not issue a session cookie")
		}
	}
}

func TestUpdateAndRemoveEndpoints(t *testing.T) {
	app, db := newTestApp(t)
	variant := variantID(t, db, "IP15PM-NAT-256") // stock 9

	resp := do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":2}`, "", "", origin)
	token := guestCookie(t, resp)
	lineID := itoa(int64(payload(t, resp)["items"].([]any)[0].(map[string]any)["id"].(float64)))

	// Valid quantity update.
	resp = do(t, app, http.MethodPatch, "/api/v1/cart/items/"+lineID,
		`{"quantity":5}`, token, "", origin)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200", resp.StatusCode)
	}
	if q := payload(t, resp)["items"].([]any)[0].(map[string]any)["quantity"].(float64); q != 5 {
		t.Errorf("quantity = %v, want 5", q)
	}

	// Quantity beyond stock → 409, nothing changed.
	resp = do(t, app, http.MethodPatch, "/api/v1/cart/items/"+lineID,
		`{"quantity":10}`, token, "", origin)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("PATCH 10 status = %d, want 409", resp.StatusCode)
	}

	// Quantity 0 → 400.
	resp = do(t, app, http.MethodPatch, "/api/v1/cart/items/"+lineID,
		`{"quantity":0}`, token, "", origin)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("PATCH 0 status = %d, want 400", resp.StatusCode)
	}

	// Unknown line → 404 envelope.
	resp = do(t, app, http.MethodPatch, "/api/v1/cart/items/424242",
		`{"quantity":1}`, token, "", origin)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH unknown status = %d, want 404", resp.StatusCode)
	}
	if code := payload(t, resp)["error"].(map[string]any)["code"]; code != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", code)
	}

	// No session → 404 as well (never 401).
	resp = do(t, app, http.MethodPatch, "/api/v1/cart/items/"+lineID,
		`{"quantity":1}`, "", "", origin)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("PATCH no session status = %d, want 404", resp.StatusCode)
	}

	// Removal → empty cart; second removal → 404.
	resp = do(t, app, http.MethodDelete, "/api/v1/cart/items/"+lineID, "", token, "", origin)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200", resp.StatusCode)
	}
	if got := len(payload(t, resp)["items"].([]any)); got != 0 {
		t.Errorf("items after delete = %d, want 0", got)
	}
	resp = do(t, app, http.MethodDelete, "/api/v1/cart/items/"+lineID, "", token, "", origin)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE twice status = %d, want 404", resp.StatusCode)
	}
}

func TestExpiredSessionCookieBehavesAsEmpty(t *testing.T) {
	app, db := newTestApp(t)

	// A real session, then expired.
	sessions := session.New(db, true)
	token, id, err := sessions.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = db.ExecContext(context.Background(),
		"UPDATE guest_sessions SET expires_at = ? WHERE id = ?",
		time.Now().UTC().Add(-time.Minute).Format(time.RFC3339), id); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	resp := do(t, app, http.MethodGet, "/api/v1/cart", "", token, "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (expired session is not an auth error)", resp.StatusCode)
	}
	if got := len(payload(t, resp)["items"].([]any)); got != 0 {
		t.Errorf("expired-session cart has %d items, want 0", got)
	}

	// Writes with the expired cookie create a fresh session.
	variant := variantID(t, db, "IP15PM-NAT-256")
	resp = do(t, app, http.MethodPost, "/api/v1/cart/items",
		`{"variantId":`+itoa(variant)+`,"quantity":1}`, token, "", origin)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST with expired cookie status = %d, want 200", resp.StatusCode)
	}
	newToken := guestCookie(t, resp)
	if newToken == token {
		t.Error("expired cookie must be replaced by a fresh session token")
	}
}
