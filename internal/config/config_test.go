package config

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{ //nolint:gosec // Fixed test-only values.
		Listen:               ":8080",
		ServerURL:            "http://localhost:8080",
		SessionCookieSecure:  true,
		DatabaseURL:          "postgres://woodgate:woodgate@localhost:5432/woodgate",
		LogLevel:             "info",
		OIDCScopes:           []string{"openid", "email", "profile"},
		OIDCEmailClaim:       "email",
		OIDCRedirectURL:      "http://localhost:8080/api/auth/sso/callback",
		EntraSyncInterval:    time.Hour,
		StorageKind:          "file",
		StorageFileRoot:      "data/storage",
		StorageCapabilityKey: strings.Repeat("a", 64),
		StorageTransferTTL:   15 * time.Minute,
		ClientIPSource:       ClientIPSourceRemoteAddr,
	}
}

func TestConfigDefaultsAndPrefix(t *testing.T) {
	t.Setenv("WOODGATE_URL", "http://localhost:8080")
	t.Setenv("WOODGATE_DATABASE_URL", "postgres://woodgate:woodgate@localhost:5432/woodgate")
	t.Setenv("WOODGATE_STORAGE_CAPABILITY_KEY", strings.Repeat("a", 64))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Listen != ":8080" || cfg.LogLevel != "info" {
		t.Fatalf("defaults = listen %q log %q", cfg.Listen, cfg.LogLevel)
	}
	if cfg.OIDCRedirectURL != "http://localhost:8080/api/auth/sso/callback" {
		t.Fatalf("redirect URL = %q", cfg.OIDCRedirectURL)
	}
	if !cfg.SessionCookieSecure {
		t.Fatal("session cookie is not Secure by default")
	}
}

func TestConfigNormalizesOrigins(t *testing.T) {
	cfg := validConfig()
	cfg.ServerURL = " http://example.com/ "
	cfg.CORSAllowedOrigins = []string{"https://panel.example/", "https://panel.example"}
	cfg.Normalize()

	if cfg.ServerURL != "http://example.com" {
		t.Fatalf("ServerURL = %q", cfg.ServerURL)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "https://panel.example" {
		t.Fatalf("CORSAllowedOrigins = %#v", cfg.CORSAllowedOrigins)
	}
}

func TestConfigRejectsPartialOIDC(t *testing.T) {
	cfg := validConfig()
	cfg.OIDCIssuerURL = "https://login.example/tenant"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted partial OIDC configuration")
	}
}
