// Package config loads backend configuration from environment
// variables. Variable names and defaults are the ones documented in
// .env.example and backend/README.md.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Config holds every environment-driven setting of the server.
type Config struct {
	// Port is the HTTP listen port (PORT, default "8080").
	Port string

	// DatabasePath is the SQLite file path (DATABASE_PATH, default
	// "./data/omnichannel.db").
	DatabasePath string

	// CORSAllowedOrigins lists the origins allowed by CORS and by the
	// Origin validation middleware (CORS_ALLOWED_ORIGINS, default
	// "http://localhost:3000,http://127.0.0.1:3000"). Comma-separated.
	CORSAllowedOrigins []string

	// CookieSecure controls the Secure flag on the guest session
	// cookie (COOKIE_SECURE, default true; false only for local
	// plain-HTTP development).
	CookieSecure bool

	// SeedDemoOrders controls demo-order seeding (APP_SEED_DEMO_ORDERS,
	// default true).
	SeedDemoOrders bool

	// RateLimitMax caps requests per client IP per minute across the
	// versioned API (RATE_LIMIT_MAX, default 600).
	RateLimitMax int

	// RateLimitCheckoutMax caps checkout requests per client IP per
	// minute — the sensitive write path (RATE_LIMIT_CHECKOUT_MAX,
	// default 30).
	RateLimitCheckoutMax int

	// TurnstileSecretKey enables optional Cloudflare Turnstile human
	// verification on checkout (TURNSTILE_SECRET_KEY). Empty = disabled
	// (local MVP behavior); when set, checkout FAILS CLOSED without a
	// valid token. The secret never leaves the backend.
	TurnstileSecretKey string

	// TurnstileVerifyURL overrides the Cloudflare siteverify endpoint
	// (TURNSTILE_VERIFY_URL) — used by tests to stub verification.
	TurnstileVerifyURL string
}

// Load builds a Config from getenv, applying the documented defaults
// and validating every value. getenv is injected so tests do not
// mutate the process environment.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:         "8080",
		DatabasePath: "./data/omnichannel.db",
	}

	if v := getenv("PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 65535 {
			return Config{}, fmt.Errorf("config: PORT must be an integer between 1 and 65535, got %q", v)
		}
		cfg.Port = v
	}

	if v := getenv("DATABASE_PATH"); v != "" {
		cfg.DatabasePath = v
	}

	origins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if v := getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		origins = nil
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		if len(origins) == 0 {
			return Config{}, fmt.Errorf("config: CORS_ALLOWED_ORIGINS must contain at least one origin")
		}
	}
	for _, o := range origins {
		if !strings.HasPrefix(o, "http://") && !strings.HasPrefix(o, "https://") {
			return Config{}, fmt.Errorf("config: CORS origin %q must start with http:// or https://", o)
		}
	}
	cfg.CORSAllowedOrigins = origins

	cookieSecure := true
	if v := getenv("COOKIE_SECURE"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("config: COOKIE_SECURE must be a boolean, got %q", v)
		}
		cookieSecure = b
	}
	cfg.CookieSecure = cookieSecure

	seedDemo := true
	if v := getenv("APP_SEED_DEMO_ORDERS"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("config: APP_SEED_DEMO_ORDERS must be a boolean, got %q", v)
		}
		seedDemo = b
	}
	cfg.SeedDemoOrders = seedDemo

	rateMax := 600
	if v := getenv("RATE_LIMIT_MAX"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("config: RATE_LIMIT_MAX must be a positive integer, got %q", v)
		}
		rateMax = n
	}
	cfg.RateLimitMax = rateMax

	checkoutMax := 30
	if v := getenv("RATE_LIMIT_CHECKOUT_MAX"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("config: RATE_LIMIT_CHECKOUT_MAX must be a positive integer, got %q", v)
		}
		checkoutMax = n
	}
	cfg.RateLimitCheckoutMax = checkoutMax

	cfg.TurnstileSecretKey = getenv("TURNSTILE_SECRET_KEY")
	cfg.TurnstileVerifyURL = getenv("TURNSTILE_VERIFY_URL")
	if cfg.TurnstileVerifyURL == "" {
		cfg.TurnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}

	return cfg, nil
}
