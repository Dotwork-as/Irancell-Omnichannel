package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

const allowedOrigin = "http://localhost:3000"

func originTestApp() *fiber.App {
	app := fiber.New()
	app.Use(Locale())
	app.Use(OriginValidation([]string{allowedOrigin}))
	app.All("/target", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func do(t *testing.T, app *fiber.App, method, origin, acceptLanguage string) (*http.Response, error) {
	t.Helper()
	req := httptest.NewRequest(method, "/target", nil)
	if origin != "" {
		req.Header.Set(fiber.HeaderOrigin, origin)
	}
	if acceptLanguage != "" {
		req.Header.Set(fiber.HeaderAcceptLanguage, acceptLanguage)
	}
	return app.Test(req)
}

func errorPayload(t *testing.T, resp *http.Response) map[string]any {
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
		t.Fatalf("body is not the §4.3 envelope (%q): %v", body, err)
	}
	return map[string]any{"code": payload.Error.Code, "message": payload.Error.Message}
}

func TestOriginValidationBlocksStateChangingRequests(t *testing.T) {
	app := originTestApp()

	cases := []struct {
		name   string
		method string
		origin string
	}{
		{"POST without origin", http.MethodPost, ""},
		{"POST with foreign origin", http.MethodPost, "https://evil.test"},
		{"PATCH without origin", http.MethodPatch, ""},
		{"DELETE with foreign origin", http.MethodDelete, "https://evil.test"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := do(t, app, tc.method, tc.origin, "")
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", resp.StatusCode)
			}
			payload := errorPayload(t, resp)
			if payload["code"] != "FORBIDDEN_ORIGIN" {
				t.Errorf("code = %v, want FORBIDDEN_ORIGIN", payload["code"])
			}
			// Default locale is fa, so the message must be Persian.
			if msg, _ := payload["message"].(string); msg == "" || msg == "Request origin is not allowed." {
				t.Errorf("message = %q, want the localized (fa) message", msg)
			}
		})
	}
}

func TestOriginValidationAllowsStateChangingFromAllowedOrigin(t *testing.T) {
	app := originTestApp()

	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		resp, err := do(t, app, method, allowedOrigin, "")
		if err != nil {
			t.Fatalf("%s app.Test: %v", method, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s status = %d, want 200", method, resp.StatusCode)
		}
	}
}

func TestOriginValidationExemptsReads(t *testing.T) {
	app := originTestApp()

	resp, err := do(t, app, http.MethodGet, "", "")
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET without origin status = %d, want 200 (reads are exempt)", resp.StatusCode)
	}
}

func TestOriginValidationLocalizesRejection(t *testing.T) {
	app := originTestApp()

	resp, err := do(t, app, http.MethodPost, "https://evil.test", "en-US,en;q=0.9")
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	payload := errorPayload(t, resp)
	if payload["message"] != "Request origin is not allowed." {
		t.Errorf("message = %v, want the en message", payload["message"])
	}
}
