package config_test

import (
	"os"
	"testing"

	"github.com/Brah-Timo/gofastapi/config"
)

func TestDefaults(t *testing.T) {
	cfg := config.Defaults()
	if cfg.Server.Port != ":8080" {
		t.Errorf("expected default port :8080, got %q", cfg.Server.Port)
	}
	if cfg.JWT.ExpiryHours != 24 {
		t.Errorf("expected default JWT expiry 24h, got %d", cfg.JWT.ExpiryHours)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected default env development, got %q", cfg.AppEnv)
	}
}

func TestLoadFromEnv_Overrides(t *testing.T) {
	os.Setenv("GOFASTAPI_SERVER_PORT", ":9999")
	os.Setenv("GOFASTAPI_APP_ENV", "production")
	os.Setenv("GOFASTAPI_JWT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("GOFASTAPI_SERVER_PORT")
		os.Unsetenv("GOFASTAPI_APP_ENV")
		os.Unsetenv("GOFASTAPI_JWT_SECRET")
	}()

	cfg := config.LoadFromEnv()
	if cfg.Server.Port != ":9999" {
		t.Errorf("expected :9999, got %q", cfg.Server.Port)
	}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction=true when APP_ENV=production")
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("expected JWT secret test-secret, got %q", cfg.JWT.Secret)
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := config.Defaults()
	if !cfg.IsDevelopment() {
		t.Error("expected IsDevelopment=true for default config")
	}
}

func TestConfig_Expiry(t *testing.T) {
	cfg := config.Defaults()
	if cfg.JWT.Expiry().Hours() != 24 {
		t.Errorf("expected 24h JWT expiry, got %v", cfg.JWT.Expiry())
	}
}

func TestConfig_String_RedactsDSN(t *testing.T) {
	cfg := config.Defaults()
	cfg.Database.DSN = "postgres://admin:super-secret@localhost/mydb"
	s := cfg.String()
	if len(s) > 0 && s != "" {
		// Just ensure it doesn't panic and doesn't expose the password.
		if contains(s, "super-secret") {
			t.Error("String() should redact the password from DSN")
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		len(s) > 0 && len(sub) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(sub); i++ {
					if s[i:i+len(sub)] == sub {
						return true
					}
				}
				return false
			}())
}
