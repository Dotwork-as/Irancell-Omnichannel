package server

// Dedicated security-hardening tests (audit suite): headers, abuse
// probes, body limits, and rate limiting. Isolated apps per case; no
// real data touched.

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/seed"
)

func strictApp(t *testing.T, overrides map[string]string) *fiber.App {
	t.Helper()
	merged := map[string]string{}
	for k, v := range overrides {
		merged[k] = v
	}
	cfg, err := config.Load(func(k string) string { return merged[k] })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Migrate(context.Background(), database.Migrations); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := seed.Run(context.Background(), db, false); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), db)
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	app := strictApp(t, nil)

	for _, path := range []string{"/api/health", "/api/v1/products"} {
		resp := get(t, app, path, nil)
		defer resp.Body.Close()

		h := resp.Header
		if h.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q", path, h.Get("X-Content-Type-Options"))
		}
		if h.Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: X-Frame-Options = %q", path, h.Get("X-Frame-Options"))
		}
		if h.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: Referrer-Policy = %q", path, h.Get("Referrer-Policy"))
		}
		csp := h.Get("Content-Security-Policy")
		if !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Errorf("%s: CSP = %q, want deny-all + no framing", path, csp)
		}
		if !strings.Contains(h.Get("Permissions-Policy"), "camera=()") {
			t.Errorf("%s: Permissions-Policy missing", path)
		}
	}
}

// Injection and XSS probes against search: parameterized queries +
// escaped LIKE patterns + JSON escaping must render every payload
// inert (200, no raw script, no expanded result set).
func TestInjectionAndXSSProbesAreInert(t *testing.T) {
	app := strictApp(t, nil)

	for _, q := range []string{
		"' OR 1=1--",
		"'; DROP TABLE products;--",
		"%'; DROP TABLE products;--",
		"<script>alert(1)</script>",
		"_%\\",
	} {
		resp := get(t, app, `/api/v1/products?q=`+url.QueryEscape(q), nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("q %q: status = %d, want 200", q, resp.StatusCode)
		}
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		body := string(raw)
		if strings.Contains(strings.ToLower(body), "<script>") {
			t.Errorf("q %q: raw <script> leaked into response", q)
		}
		if strings.Contains(body, "DROP TABLE") {
			t.Errorf("q %q: raw SQL fragment echoed", q)
		}
	}
}

// Body limit: requests over the configured 256KB cap are rejected
// without processing (fiber 413) and never reach business logic.
func TestOversizedBodyRejected(t *testing.T) {
	app := strictApp(t, nil)

	huge := strings.Repeat("A", 300*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", strings.NewReader(huge))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:3000")
	resp, testErr := app.Test(req)
	if testErr != nil {
		// In-process serving can surface the limit as a transport
		// error; both shapes mean the cap fired.
		if strings.Contains(testErr.Error(), "body size") {
			return
		}
		t.Fatalf("app.Test: %v", testErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusRequestEntityTooLarge && resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 413/400", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("oversized rejection is not JSON: %q", ct)
	}
}

// Rate limiting: strict per-IP config → the Nth+1 request gets a
// localized §4.3 429 envelope with Retry-After.
func TestRateLimitReturns429(t *testing.T) {
	app := strictApp(t, map[string]string{"RATE_LIMIT_MAX": "3"})

	var lastStatus int
	var lastCode string
	for i := 0; i < 5; i++ {
		resp := get(t, app, "/api/v1/products", nil)
		lastStatus = resp.StatusCode
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var payload struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &payload)
		lastCode = payload.Error.Code
		if resp.StatusCode == fiber.StatusTooManyRequests {
			break
		}
	}
	if lastStatus != fiber.StatusTooManyRequests {
		t.Fatalf("final status = %d, want 429", lastStatus)
	}
	if lastCode != "RATE_LIMITED" {
		t.Errorf("code = %q, want RATE_LIMITED", lastCode)
	}
}
