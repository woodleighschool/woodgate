package config

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is the root application configuration.
type Config struct {
	HTTP     HTTPConfig
	Logging  LoggingConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Entra    EntraSyncConfig
	Media    MediaConfig
}

// HTTPConfig contains HTTP server settings.
type HTTPConfig struct {
	Port    int    `env:"WOODGATE_PORT"     envDefault:"8080"`
	BaseURL string `env:"WOODGATE_BASE_URL"`
}

func (cfg HTTPConfig) Addr() string {
	return fmt.Sprintf(":%d", cfg.Port)
}

// LoggingConfig contains structured logger settings.
type LoggingConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

// DatabaseConfig contains Postgres connection settings.
type DatabaseConfig struct {
	Host     string `env:"DATABASE_HOST"`
	Port     int    `env:"DATABASE_PORT"     envDefault:"5432"`
	User     string `env:"DATABASE_USER"`
	Password string `env:"DATABASE_PASSWORD"`
	Name     string `env:"DATABASE_NAME"`
	SSLMode  string `env:"DATABASE_SSLMODE"  envDefault:"disable"`
}

// AuthConfig contains operator auth settings.
type AuthConfig struct {
	EntraTenantID     string `env:"ENTRA_TENANT_ID"`
	EntraClientID     string `env:"ENTRA_CLIENT_ID"`
	EntraClientSecret string `env:"ENTRA_CLIENT_SECRET"`
	JWTSecret         string `env:"JWT_SECRET"`
	LocalAdminPass    string `env:"LOCAL_ADMIN_PASSWORD"`
}

// EntraSyncConfig contains Graph synchronization settings.
type EntraSyncConfig struct {
	Enabled  bool          `env:"ENTRA_SYNC_ENABLED"  envDefault:"false"`
	Interval time.Duration `env:"ENTRA_SYNC_INTERVAL" envDefault:"1h"`
}

// MediaConfig contains media storage settings.
type MediaConfig struct {
	RootDir string `env:"WOODGATE_MEDIA_ROOT" envDefault:"media"`
}

// LoadFromEnv loads and validates all configuration from environment variables.
func LoadFromEnv() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	cfg.Logging.Level = strings.ToLower(cfg.Logging.Level)

	validateErr := validateConfig(cfg)
	if validateErr != nil {
		return Config{}, validateErr
	}

	return cfg, nil
}

func validateConfig(cfg Config) error {
	problems := make([]string, 0)
	problems = append(problems, validateHTTPAndLogging(cfg)...)
	problems = append(problems, validateDatabase(cfg.Database)...)
	problems = append(problems, validateAuth(cfg)...)
	problems = append(problems, validateEntra(cfg)...)
	problems = append(problems, validateMedia(cfg.Media)...)

	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("invalid config: %s", strings.Join(problems, "; "))
}

func validateHTTPAndLogging(cfg Config) []string {
	problems := make([]string, 0)

	if cfg.HTTP.Port < 1 || cfg.HTTP.Port > 65535 {
		problems = append(problems, "WOODGATE_PORT must be between 1 and 65535")
	}
	if cfg.HTTP.BaseURL != "" && !isValidBaseURL(cfg.HTTP.BaseURL) {
		problems = append(problems, "WOODGATE_BASE_URL must be a valid http or https URL")
	}
	if !slices.Contains([]string{"debug", "info", "warn", "error"}, cfg.Logging.Level) {
		problems = append(problems, "LOG_LEVEL must be one of: debug, info, warn, error")
	}

	return problems
}

func validateDatabase(cfg DatabaseConfig) []string {
	problems := make([]string, 0)
	missingDatabaseEnvVars := missingEnvVars(
		envVarValue("DATABASE_HOST", cfg.Host),
		envVarValue("DATABASE_USER", cfg.User),
		envVarValue("DATABASE_PASSWORD", cfg.Password),
		envVarValue("DATABASE_NAME", cfg.Name),
		envVarValue("DATABASE_SSLMODE", cfg.SSLMode),
	)

	if len(missingDatabaseEnvVars) > 0 {
		problems = append(problems, "missing required env vars: "+strings.Join(missingDatabaseEnvVars, ", "))
	}
	if cfg.Port <= 0 {
		problems = append(problems, "DATABASE_PORT must be greater than 0")
	}

	return problems
}

func validateAuth(cfg Config) []string {
	problems := make([]string, 0)
	hasLocalAdmin := cfg.Auth.LocalAdminPass != ""
	hasEntraAuth := hasCompleteEntraCredentials(cfg.Auth)

	if !hasLocalAdmin && !hasEntraAuth {
		problems = append(
			problems,
			"set one auth provider: LOCAL_ADMIN_PASSWORD or ENTRA_TENANT_ID, ENTRA_CLIENT_ID, ENTRA_CLIENT_SECRET",
		)
	}
	if hasAnyEntraCredential(cfg.Auth) {
		missingEntraAuthEnvVars := missingEntraEnvVars(cfg.Auth)
		if len(missingEntraAuthEnvVars) > 0 {
			problems = append(
				problems,
				"missing required env vars for Entra auth: "+strings.Join(missingEntraAuthEnvVars, ", "),
			)
		}
	}
	if hasLocalAdmin || hasEntraAuth {
		if cfg.Auth.JWTSecret == "" {
			problems = append(problems, "missing required env vars: JWT_SECRET")
		}
		if cfg.HTTP.BaseURL == "" {
			problems = append(problems, "missing required env vars: WOODGATE_BASE_URL")
		}
	}

	return problems
}

func validateEntra(cfg Config) []string {
	if !cfg.Entra.Enabled {
		return nil
	}

	problems := make([]string, 0)
	missingEntraSyncEnvVars := missingEntraEnvVars(cfg.Auth)
	if len(missingEntraSyncEnvVars) > 0 {
		problems = append(
			problems,
			"missing required env vars for ENTRA_SYNC_ENABLED=true: "+strings.Join(missingEntraSyncEnvVars, ", "),
		)
	}
	if cfg.Entra.Interval <= 0 {
		problems = append(problems, "ENTRA_SYNC_INTERVAL must be greater than 0")
	}

	return problems
}

func validateMedia(cfg MediaConfig) []string {
	if strings.TrimSpace(cfg.RootDir) == "" {
		return []string{"WOODGATE_MEDIA_ROOT must not be empty"}
	}

	return nil
}

type envVar struct {
	name  string
	value string
}

func envVarValue(name string, value string) envVar {
	return envVar{name: name, value: strings.TrimSpace(value)}
}

func missingEnvVars(vars ...envVar) []string {
	missing := make([]string, 0)
	for _, variable := range vars {
		if variable.value == "" {
			missing = append(missing, variable.name)
		}
	}

	return missing
}

func hasAnyEntraCredential(cfg AuthConfig) bool {
	return strings.TrimSpace(cfg.EntraTenantID) != "" ||
		strings.TrimSpace(cfg.EntraClientID) != "" ||
		strings.TrimSpace(cfg.EntraClientSecret) != ""
}

func hasCompleteEntraCredentials(cfg AuthConfig) bool {
	return strings.TrimSpace(cfg.EntraTenantID) != "" &&
		strings.TrimSpace(cfg.EntraClientID) != "" &&
		strings.TrimSpace(cfg.EntraClientSecret) != ""
}

func missingEntraEnvVars(cfg AuthConfig) []string {
	return missingEnvVars(
		envVarValue("ENTRA_TENANT_ID", cfg.EntraTenantID),
		envVarValue("ENTRA_CLIENT_ID", cfg.EntraClientID),
		envVarValue("ENTRA_CLIENT_SECRET", cfg.EntraClientSecret),
	)
}

func isValidBaseURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}
