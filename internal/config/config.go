package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

	"github.com/woodleighschool/woodgate/internal/validation"
)

const oidcCallbackPath = "/api/auth/sso/callback"

// SessionLifetime is the browser session lifetime.
const SessionLifetime = 14 * 24 * time.Hour

var (
	ErrInvalidOIDCRedirectURL = errors.New("invalid OIDC redirect URL")
)

// Config contains runtime settings.
type Config struct {
	Listen              string   `env:"LISTEN"                envDefault:":8080" validate:"required"`
	ServerURL           string   `env:"URL"                                        validate:"required,web_origin"`
	SessionCookieSecure bool     `env:"SESSION_COOKIE_SECURE" envDefault:"true"`
	DatabaseURL         string   `env:"DATABASE_URL"                               validate:"required"`
	LogLevel            string   `env:"LOG_LEVEL"             envDefault:"info"    validate:"required,oneof=debug info warn error"`
	CORSAllowedOrigins  []string `env:"CORS_ALLOWED_ORIGINS"                       validate:"dive,web_origin"`

	OIDCIssuerURL    string   `env:"OIDC_ISSUER_URL"    validate:"required_with=OIDCClientID OIDCClientSecret,omitempty,https_url"`
	OIDCClientID     string   `env:"OIDC_CLIENT_ID"     validate:"required_with=OIDCIssuerURL OIDCClientSecret"`
	OIDCClientSecret string   `env:"OIDC_CLIENT_SECRET" validate:"required_with=OIDCIssuerURL OIDCClientID"`
	OIDCRedirectURL  string   `env:"OIDC_REDIRECT_URL"`
	OIDCScopes       []string `env:"OIDC_SCOPES"        envDefault:"openid,email,profile" validate:"min=1,dive,required"`
	OIDCEmailClaim   string   `env:"OIDC_EMAIL_CLAIM"   envDefault:"email" validate:"required"`

	EntraTenantID         string        `env:"ENTRA_TENANT_ID"         validate:"required_with=EntraClientID EntraClientSecret"`
	EntraClientID         string        `env:"ENTRA_CLIENT_ID"         validate:"required_with=EntraTenantID EntraClientSecret"`
	EntraClientSecret     string        `env:"ENTRA_CLIENT_SECRET"     validate:"required_with=EntraTenantID EntraClientID"`
	EntraTransitiveGroups bool          `env:"ENTRA_TRANSITIVE_GROUPS"`
	EntraSyncInterval     time.Duration `env:"ENTRA_SYNC_INTERVAL" envDefault:"1h" validate:"gt=0"`

	StorageKind          string        `env:"STORAGE_KIND"           envDefault:"file" validate:"required,oneof=file s3"`
	StorageFileRoot      string        `env:"STORAGE_FILE_ROOT"      envDefault:"data/storage" validate:"required_if=StorageKind file"`
	StorageCapabilityKey string        `env:"STORAGE_CAPABILITY_KEY" validate:"required_if=StorageKind file"`
	StorageTransferTTL   time.Duration `env:"STORAGE_TRANSFER_TTL"   envDefault:"15m" validate:"gt=0"`
	StorageS3Bucket      string        `env:"STORAGE_S3_BUCKET"      validate:"required_if=StorageKind s3"`
	StorageS3Region      string        `env:"STORAGE_S3_REGION"      validate:"required_if=StorageKind s3"`
	StorageS3Endpoint    string        `env:"STORAGE_S3_ENDPOINT"    validate:"omitempty,url"`
	StorageS3AccessKey   string        `env:"STORAGE_S3_ACCESS_KEY"  validate:"required_if=StorageKind s3"`
	StorageS3SecretKey   string        `env:"STORAGE_S3_SECRET_KEY"  validate:"required_if=StorageKind s3"`
	StorageS3PathStyle   bool          `env:"STORAGE_S3_PATH_STYLE"`

	ClientIPSource         ClientIPSource `env:"HTTP_CLIENT_IP_SOURCE" envDefault:"remote_addr" validate:"required,oneof=remote_addr xff_trusted_cidrs xff_trusted_proxies header"`
	ClientIPTrustedCIDRs   []string       `env:"HTTP_CLIENT_IP_TRUSTED_CIDRS" validate:"excluded_unless=ClientIPSource xff_trusted_cidrs,required_if=ClientIPSource xff_trusted_cidrs,dive,cidr"`
	ClientIPTrustedProxies int            `env:"HTTP_CLIENT_IP_TRUSTED_PROXY_COUNT" validate:"excluded_unless=ClientIPSource xff_trusted_proxies,required_if=ClientIPSource xff_trusted_proxies,omitempty,gte=1"`
	ClientIPHeader         string         `env:"HTTP_CLIENT_IP_HEADER" validate:"excluded_unless=ClientIPSource header,required_if=ClientIPSource header"`
}

type ClientIPSource string

const (
	ClientIPSourceRemoteAddr        ClientIPSource = "remote_addr"
	ClientIPSourceXFFTrustedCIDRs   ClientIPSource = "xff_trusted_cidrs"
	ClientIPSourceXFFTrustedProxies ClientIPSource = "xff_trusted_proxies"
	ClientIPSourceHeader            ClientIPSource = "header"
)

func (cfg *Config) OIDCEnabled() bool {
	return cfg.OIDCIssuerURL != "" && cfg.OIDCClientID != "" && cfg.OIDCClientSecret != ""
}

func (cfg *Config) EntraEnabled() bool {
	return cfg.EntraTenantID != "" && cfg.EntraClientID != "" && cfg.EntraClientSecret != ""
}

func Load() (Config, error) {
	var cfg Config
	if err := parseEnvironment(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse environment: %w", err)
	}
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func parseEnvironment(cfg *Config) error {
	return env.ParseWithOptions(cfg, env.Options{
		Prefix:                       "WOODGATE_",
		SetDefaultsForZeroValuesOnly: true,
	})
}

func (cfg *Config) Normalize() {
	cfg.Listen = strings.TrimSpace(cfg.Listen)
	cfg.ServerURL = normalizeOrigin(cfg.ServerURL)
	cfg.OIDCRedirectURL = strings.TrimSpace(cfg.OIDCRedirectURL)
	if cfg.OIDCRedirectURL == "" && cfg.ServerURL != "" {
		cfg.OIDCRedirectURL = cfg.ServerURL + oidcCallbackPath
	}
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
	cfg.OIDCIssuerURL = strings.TrimSpace(cfg.OIDCIssuerURL)
	cfg.OIDCClientID = strings.TrimSpace(cfg.OIDCClientID)
	cfg.OIDCScopes = normalizeStrings(cfg.OIDCScopes)
	cfg.OIDCEmailClaim = strings.TrimSpace(cfg.OIDCEmailClaim)
	cfg.EntraTenantID = strings.TrimSpace(cfg.EntraTenantID)
	cfg.EntraClientID = strings.TrimSpace(cfg.EntraClientID)
	cfg.normalizeStorage()
	cfg.normalizeCORSAllowedOrigins()
	cfg.normalizeClientIP()
}

func (cfg *Config) Validate() error {
	if !validOIDCRedirectURL(cfg.OIDCRedirectURL) {
		return fmt.Errorf(
			"%w: must be an HTTP or HTTPS URL ending in %s",
			ErrInvalidOIDCRedirectURL,
			oidcCallbackPath,
		)
	}
	if err := validation.Struct(cfg); err != nil {
		return err
	}
	if cfg.StorageKind == "s3" && cfg.StorageS3Endpoint != "" &&
		!validation.IsHTTPSOrigin(cfg.StorageS3Endpoint) {
		return errors.New("StorageS3Endpoint must resolve to an HTTPS origin")
	}
	return nil
}

func validOIDCRedirectURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.ForceQuery || parsed.Fragment != "" || parsed.Path != oidcCallbackPath {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func (cfg *Config) normalizeCORSAllowedOrigins() {
	normalized := make([]string, 0, len(cfg.CORSAllowedOrigins))
	seen := make(map[string]struct{}, len(cfg.CORSAllowedOrigins))
	for _, raw := range cfg.CORSAllowedOrigins {
		origin := normalizeOrigin(raw)
		if origin == "" {
			continue
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		normalized = append(normalized, origin)
	}
	cfg.CORSAllowedOrigins = normalized
}

func (cfg *Config) normalizeClientIP() {
	cfg.ClientIPSource = ClientIPSource(strings.TrimSpace(string(cfg.ClientIPSource)))
	cfg.ClientIPHeader = strings.TrimSpace(cfg.ClientIPHeader)
	for i := range cfg.ClientIPTrustedCIDRs {
		cfg.ClientIPTrustedCIDRs[i] = strings.TrimSpace(cfg.ClientIPTrustedCIDRs[i])
	}
}

func (cfg *Config) normalizeStorage() {
	cfg.StorageKind = strings.ToLower(strings.TrimSpace(cfg.StorageKind))
	cfg.StorageFileRoot = strings.TrimSpace(cfg.StorageFileRoot)
	cfg.StorageS3Bucket = strings.TrimSpace(cfg.StorageS3Bucket)
	cfg.StorageS3Region = strings.TrimSpace(cfg.StorageS3Region)
	cfg.StorageS3Endpoint = normalizeOrigin(cfg.StorageS3Endpoint)
	cfg.StorageS3AccessKey = strings.TrimSpace(cfg.StorageS3AccessKey)
}

func normalizeOrigin(value string) string {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return value
	}
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	return parsed.String()
}

func normalizeStrings(values []string) []string {
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	return values
}
