package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load defaults: unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DatabasePath != "./data/omnichannel.db" {
		t.Errorf("DatabasePath = %q, want ./data/omnichannel.db", cfg.DatabasePath)
	}
	wantOrigins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	if strings.Join(cfg.CORSAllowedOrigins, "|") != strings.Join(wantOrigins, "|") {
		t.Errorf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, wantOrigins)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure = false, want true")
	}
	if !cfg.SeedDemoOrders {
		t.Error("SeedDemoOrders = false, want true")
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"PORT":                 "9000",
		"DATABASE_PATH":        "/tmp/test.db",
		"CORS_ALLOWED_ORIGINS": "http://a.test, https://b.test ,",
		"COOKIE_SECURE":        "false",
		"APP_SEED_DEMO_ORDERS": "false",
	}
	cfg, err := Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Load overrides: unexpected error: %v", err)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want 9000", cfg.Port)
	}
	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("DatabasePath = %q, want /tmp/test.db", cfg.DatabasePath)
	}
	want := []string{"http://a.test", "https://b.test"}
	if strings.Join(cfg.CORSAllowedOrigins, "|") != strings.Join(want, "|") {
		t.Errorf("CORSAllowedOrigins = %v, want %v (trimmed, empties dropped)", cfg.CORSAllowedOrigins, want)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure = true, want false")
	}
	if cfg.SeedDemoOrders {
		t.Error("SeedDemoOrders = true, want false")
	}
}

func TestLoadInvalidPort(t *testing.T) {
	for _, port := range []string{"abc", "0", "-1", "70000", "1.5"} {
		if _, err := Load(func(k string) string {
			if k == "PORT" {
				return port
			}
			return ""
		}); err == nil {
			t.Errorf("PORT %q: expected error, got nil", port)
		}
	}
}

func TestLoadInvalidBool(t *testing.T) {
	for _, name := range []string{"COOKIE_SECURE", "APP_SEED_DEMO_ORDERS"} {
		if _, err := Load(func(k string) string {
			if k == name {
				return "yes"
			}
			return ""
		}); err == nil {
			t.Errorf("%s = yes: expected error, got nil", name)
		}
	}
}

func TestLoadInvalidOrigins(t *testing.T) {
	for _, origins := range []string{"localhost:3000", "http://a.test,ftp://b.test", ",,"} {
		if _, err := Load(func(k string) string {
			if k == "CORS_ALLOWED_ORIGINS" {
				return origins
			}
			return ""
		}); err == nil {
			t.Errorf("CORS_ALLOWED_ORIGINS %q: expected error, got nil", origins)
		}
	}
}
