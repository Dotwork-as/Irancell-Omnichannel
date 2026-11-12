package server

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
	"backend/internal/seed"
)

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
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db)
}

func get(t *testing.T, app *fiber.App, path string, headers map[string]string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s: %v", path, err)
	}
	return resp
}

func decodeErrorEnvelope(t *testing.T, resp *http.Response) (code string, message string) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body %q is not the §4.3 envelope: %v", body, err)
	}
	return payload.Error.Code, payload.Error.Message
}

func TestUnknownAPIRouteReturnsEnvelope(t *testing.T) {
	app := newTestApp(t)

	// /api/v1/products is a live route; probe an unknown one.
	resp := get(t, app, "/api/v1/definitely-not-a-route", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	code, _ := decodeErrorEnvelope(t, resp)
	if code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", code)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// fiber v2 reports a method mismatch on a registered path as 404
// through the catch-all (no router-level 405 until fiber v3); the
// envelope must still be §4.3 NOT_FOUND.
func TestMethodMismatchReturnsNotFoundEnvelope(t *testing.T) {
	app := newTestApp(t)
	app.Delete("/api/v1/only-delete", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	resp := get(t, app, "/api/v1/only-delete", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (fiber v2 method-mismatch semantics)", resp.StatusCode)
	}
	code, _ := decodeErrorEnvelope(t, resp)
	if code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", code)
	}
}

func TestRequestIDHeaderPresent(t *testing.T) {
	app := newTestApp(t)

	resp := get(t, app, "/api/v1/unknown", nil)
	defer resp.Body.Close()

	if resp.Header.Get(fiber.HeaderXRequestID) == "" {
		t.Error("X-Request-ID response header is empty, want a request id on every response")
	}
}

func TestCORSPreflightWithCredentials(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/anything", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Accept-Language")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the requesting origin",
			resp.Header.Get("Access-Control-Allow-Origin"))
	}
	if resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Access-Control-Allow-Credentials missing or not true")
	}
	if am := resp.Header.Get("Access-Control-Allow-Methods"); !containsAll(am, "GET", "POST", "PATCH", "DELETE") {
		t.Errorf("Access-Control-Allow-Methods = %q, want GET, POST, PATCH, DELETE", am)
	}
	if ah := resp.Header.Get("Access-Control-Allow-Headers"); !containsAll(ah, "Content-Type", "Accept-Language") {
		t.Errorf("Access-Control-Allow-Headers = %q, want Content-Type and Accept-Language", ah)
	}
}

func containsAll(s string, wants ...string) bool {
	for _, w := range wants {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// GET /api/health: liveness only — 200 with the
// minimal {"status":"ok"} body, no readiness or configuration data.
func TestHealthEndpoint(t *testing.T) {
	app := newTestApp(t)

	resp := get(t, app, "/api/health", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want 200", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", raw, err)
	}
	if len(body) != 1 || body["status"] != "ok" {
		t.Errorf("body = %s, want exactly {\"status\":\"ok\"}", raw)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	// Liveness response must not leak configuration or internals.
	for _, leak := range []string{"database", "sqlite", "path", "data/", "dsn", "config"} {
		if strings.Contains(strings.ToLower(string(raw)), leak) {
			t.Errorf("health body leaks internal keyword %q: %s", leak, raw)
		}
	}
}

func TestWriteWithoutOriginRejectedByServerChain(t *testing.T) {
	app := newTestApp(t)
	app.Post("/api/v1/write", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusCreated) })

	req := httptest.NewRequest(http.MethodPost, "/api/v1/write", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (full chain must enforce Origin validation)", resp.StatusCode)
	}
	code, _ := decodeErrorEnvelope(t, resp)
	if code != "FORBIDDEN_ORIGIN" {
		t.Errorf("code = %q, want FORBIDDEN_ORIGIN", code)
	}
}
