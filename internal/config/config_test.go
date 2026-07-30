package config_test

import (
	"strings"
	"testing"

	"github.com/woodleighschool/woodgate/internal/config"
)

func TestLoadFromEnv_RequiresEntraCredentialsWhenSyncEnabled(t *testing.T) {
	t.Setenv("WOODGATE_PORT", "18080")
	t.Setenv("WOODGATE_BASE_URL", "https://woodgate.example.com")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "admin")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("DATABASE_HOST", "db")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "postgres")
	t.Setenv("DATABASE_NAME", "woodgate")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("ENTRA_SYNC_ENABLED", "true")
	t.Setenv("ENTRA_TENANT_ID", "")
	t.Setenv("ENTRA_CLIENT_ID", "")
	t.Setenv("ENTRA_CLIENT_SECRET", "")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatalf("LoadFromEnv() expected error, got nil")
	}

	if !strings.Contains(
		err.Error(),
		"missing required env vars for ENTRA_SYNC_ENABLED=true: ENTRA_TENANT_ID, ENTRA_CLIENT_ID, ENTRA_CLIENT_SECRET",
	) {
		t.Fatalf("expected missing Entra sync env vars error, got: %v", err)
	}
}

func TestLoadFromEnv_RequiresAuthProvider(t *testing.T) {
	t.Setenv("WOODGATE_PORT", "18080")
	t.Setenv("WOODGATE_BASE_URL", "https://woodgate.example.com")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("DATABASE_HOST", "db")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "postgres")
	t.Setenv("DATABASE_NAME", "woodgate")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("ENTRA_SYNC_ENABLED", "false")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatalf("LoadFromEnv() expected error, got nil")
	}

	if !strings.Contains(
		err.Error(),
		"set one auth provider: LOCAL_ADMIN_PASSWORD or ENTRA_TENANT_ID, ENTRA_CLIENT_ID, ENTRA_CLIENT_SECRET",
	) {
		t.Fatalf("expected auth provider error, got: %v", err)
	}
}

func TestLoadFromEnv_RequiresJWTSecretWhenAuthEnabled(t *testing.T) {
	t.Setenv("WOODGATE_PORT", "18080")
	t.Setenv("WOODGATE_BASE_URL", "https://woodgate.example.com")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("DATABASE_HOST", "db")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "postgres")
	t.Setenv("DATABASE_NAME", "woodgate")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("ENTRA_SYNC_ENABLED", "false")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "admin")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatalf("LoadFromEnv() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing required env vars: JWT_SECRET") {
		t.Fatalf("expected missing JWT_SECRET error, got: %v", err)
	}
}

func TestLoadFromEnv_RequiresBaseURLWhenAuthEnabled(t *testing.T) {
	t.Setenv("WOODGATE_PORT", "18080")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("DATABASE_HOST", "db")
	t.Setenv("DATABASE_PORT", "5432")
	t.Setenv("DATABASE_USER", "postgres")
	t.Setenv("DATABASE_PASSWORD", "postgres")
	t.Setenv("DATABASE_NAME", "woodgate")
	t.Setenv("DATABASE_SSLMODE", "disable")
	t.Setenv("ENTRA_SYNC_ENABLED", "false")
	t.Setenv("LOCAL_ADMIN_PASSWORD", "admin")
	t.Setenv("JWT_SECRET", "jwt-secret")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatalf("LoadFromEnv() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "missing required env vars: WOODGATE_BASE_URL") {
		t.Fatalf("expected missing WOODGATE_BASE_URL error, got: %v", err)
	}
}
