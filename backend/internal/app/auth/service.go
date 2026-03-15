package auth

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/auth"
	"github.com/go-pkgz/auth/avatar"
	"github.com/go-pkgz/auth/provider"
	"github.com/go-pkgz/auth/token"
	"golang.org/x/oauth2/microsoft"
)

const (
	sessionCookieName = "woodgate_session"
	xsrfCookieName    = "woodgate_xsrf"

	providerMicrosoft = "microsoft"
	providerLocal     = "local"

	defaultTokenDuration  = 12 * time.Hour
	defaultCookieDuration = 7 * 24 * time.Hour
)

type Config struct {
	RootURL            string
	EntraTenantID      string
	EntraClientID      string
	EntraClientSecret  string
	JWTSecret          string
	LocalAdminPassword string
}

type Service struct {
	service *auth.Service
}

func New(cfg Config) (*Service, error) {
	hasMicrosoft := cfg.EntraTenantID != "" && cfg.EntraClientID != "" && cfg.EntraClientSecret != ""
	secureCookies := strings.HasPrefix(strings.ToLower(cfg.RootURL), "https://")

	options := auth.Opts{
		SecretReader: token.SecretFunc(func(_ string) (string, error) {
			return cfg.JWTSecret, nil
		}),
		TokenDuration:  defaultTokenDuration,
		CookieDuration: defaultCookieDuration,
		Issuer:         "woodgate",
		URL:            cfg.RootURL,
		SecureCookies:  secureCookies,
		SameSiteCookie: http.SameSiteLaxMode,
		JWTCookieName:  sessionCookieName,
		XSRFCookieName: xsrfCookieName,
		XSRFIgnoreMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
		},
		Validator: token.ValidatorFunc(func(_ string, claims token.Claims) bool {
			return claims.User != nil
		}),
		AvatarStore:     avatar.NewNoOp(),
		AvatarRoutePath: "/auth/avatar",
	}

	service := auth.NewService(options)
	if hasMicrosoft {
		service.AddCustomProvider(providerMicrosoft, auth.Client{
			Cid:     cfg.EntraClientID,
			Csecret: cfg.EntraClientSecret,
		}, provider.CustomHandlerOpt{
			Endpoint: microsoft.AzureADEndpoint(cfg.EntraTenantID),
			Scopes:   []string{"User.Read"},
			InfoURL:  "https://graph.microsoft.com/v1.0/me",
			MapUserFn: func(data provider.UserData, _ []byte) token.User {
				return token.User{
					ID:      providerMicrosoft + "_" + token.HashID(sha256.New(), data.Value("id")),
					Name:    data.Value("displayName"),
					Email:   data.Value("userPrincipalName"),
					Picture: "https://graph.microsoft.com/beta/me/photo/$value",
				}
			},
		})
	}

	if cfg.LocalAdminPassword != "" {
		service.AddDirectProvider(providerLocal, provider.CredCheckerFunc(func(user, password string) (bool, error) {
			return user == "admin" && password == cfg.LocalAdminPassword, nil
		}))
	}

	if len(service.Providers()) == 0 {
		return nil, errors.New("no auth providers registered")
	}

	return &Service{service: service}, nil
}

func (service *Service) RegisterRoutes(router chi.Router) {
	authHandler, avatarHandler := service.service.Handlers()
	router.Handle("/avatar/*", avatarHandler)
	router.Handle("/*", authHandler)
}

func (service *Service) SessionAuthMiddleware() func(http.Handler) http.Handler {
	authenticator := service.service.Middleware()
	return (&authenticator).Auth
}
