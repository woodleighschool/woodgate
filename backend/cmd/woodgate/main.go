package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	graphsync "github.com/woodleighschool/go-entrasync"

	appadmin "github.com/woodleighschool/woodgate/internal/app/admin"
	appauth "github.com/woodleighschool/woodgate/internal/app/auth"
	"github.com/woodleighschool/woodgate/internal/app/authz"
	appentrasync "github.com/woodleighschool/woodgate/internal/app/entrasync"
	"github.com/woodleighschool/woodgate/internal/config"
	"github.com/woodleighschool/woodgate/internal/platform/logging"
	"github.com/woodleighschool/woodgate/internal/store/postgres"
	adminpostgres "github.com/woodleighschool/woodgate/internal/store/postgres/admin"
	entrasyncpostgres "github.com/woodleighschool/woodgate/internal/store/postgres/entrasync"
	authhttp "github.com/woodleighschool/woodgate/internal/transport/http/auth"
	httpapi "github.com/woodleighschool/woodgate/internal/transport/http/httpapi"
	httprouter "github.com/woodleighschool/woodgate/internal/transport/http/router"
)

const (
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 5 * time.Second
	idleTimeout       = 2 * time.Minute
	frontendDistDir   = "/frontend"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "woodgate exited with error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := logging.New(cfg.Logging.Level)
	if err != nil {
		return fmt.Errorf("configure logger: %w", err)
	}
	slog.SetDefault(logger)

	store, err := postgres.New(context.Background(), cfg.Database)
	if err != nil {
		return fmt.Errorf("create store: %w", err)
	}
	defer store.Close()

	server, err := buildServer(logger, cfg, store)
	if err != nil {
		return err
	}

	stopContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	syncErr := maybeStartEntraSync(stopContext, logger, cfg, store)
	if syncErr != nil {
		return syncErr
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting server", "port", cfg.HTTP.Port)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErr <- serveErr
			return
		}
		serverErr <- nil
	}()

	select {
	case serveErr := <-serverErr:
		if serveErr != nil {
			return fmt.Errorf("serve: %w", serveErr)
		}
		return nil
	case <-stopContext.Done():
		logger.Info("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownContext)
	if shutdownErr != nil {
		return fmt.Errorf("shutdown: %w", shutdownErr)
	}

	if serveErr := <-serverErr; serveErr != nil {
		return fmt.Errorf("serve after shutdown: %w", serveErr)
	}

	logger.Info("server stopped")
	return nil
}

func buildServer(logger *slog.Logger, cfg config.Config, store *postgres.Store) (*http.Server, error) {
	adminStore := adminpostgres.New(store)
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), adminStore)
	if err != nil {
		return nil, fmt.Errorf("configure authorizer: %w", err)
	}
	adminService, err := appadmin.New(adminStore, authorizer, cfg.Media.RootDir)
	if err != nil {
		return nil, fmt.Errorf("configure admin service: %w", err)
	}

	authService, err := appauth.New(appauth.Config{
		RootURL:            cfg.HTTP.BaseURL,
		EntraTenantID:      cfg.Auth.EntraTenantID,
		EntraClientID:      cfg.Auth.EntraClientID,
		EntraClientSecret:  cfg.Auth.EntraClientSecret,
		JWTSecret:          cfg.Auth.JWTSecret,
		LocalAdminPassword: cfg.Auth.LocalAdminPass,
	})
	if err != nil {
		return nil, fmt.Errorf("configure auth: %w", err)
	}

	apiAuthMiddleware := authhttp.NewAPIMiddleware(
		authService.SessionAuthMiddleware(),
		adminStore,
		adminStore,
		authorizer,
	)
	principalMiddleware := authhttp.NewPrincipalMiddleware(
		authService.SessionAuthMiddleware(),
		adminStore,
		adminStore,
	)
	apiHandler := httpapi.New(adminService, authorizer)
	meHandler := authhttp.NewMeHandler(adminService, authorizer)

	server := &http.Server{
		Handler: httprouter.New(
			logger,
			store.Ping,
			newAuthRouteRegistrar(authService.RegisterRoutes, principalMiddleware, meHandler),
			newAPIRouteRegistrar(apiAuthMiddleware, apiHandler),
			frontendDistDir,
		),
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
		Addr:              cfg.HTTP.Addr(),
	}

	return server, nil
}

func maybeStartEntraSync(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	store *postgres.Store,
) error {
	if !cfg.Entra.Enabled {
		return nil
	}

	graphClient, err := graphsync.NewClient(graphsync.Config{
		TenantID:     cfg.Auth.EntraTenantID,
		ClientID:     cfg.Auth.EntraClientID,
		ClientSecret: cfg.Auth.EntraClientSecret,
	})
	if err != nil {
		return fmt.Errorf("configure entra graph client: %w", err)
	}

	entraSyncService := appentrasync.New(
		logger,
		graphClient,
		entrasyncpostgres.New(store),
		cfg.Entra.Interval,
	)

	go entraSyncService.Run(ctx)
	return nil
}

func newAPIRouteRegistrar(middleware func(http.Handler) http.Handler, handler *httpapi.Server) func(chi.Router) {
	return func(router chi.Router) {
		router.Use(middleware)
		handler.RegisterRoutes(router)
	}
}

func newAuthRouteRegistrar(
	register func(chi.Router),
	principalMiddleware func(http.Handler) http.Handler,
	meHandler http.Handler,
) func(chi.Router) {
	return func(router chi.Router) {
		router.With(principalMiddleware).Handle("/me", meHandler)
		register(router)
	}
}
