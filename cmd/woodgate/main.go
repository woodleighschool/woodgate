package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/spf13/cobra"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"

	"github.com/woodleighschool/woodgate/internal/account"
	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/backgroundjobs"
	"github.com/woodleighschool/woodgate/internal/buildinfo"
	"github.com/woodleighschool/woodgate/internal/checkin"
	checkinapi "github.com/woodleighschool/woodgate/internal/checkin/httpapi"
	"github.com/woodleighschool/woodgate/internal/config"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/directory/entra"
	directoryapi "github.com/woodleighschool/woodgate/internal/directory/httpapi"
	"github.com/woodleighschool/woodgate/internal/logging"
	"github.com/woodleighschool/woodgate/internal/postgres"
	"github.com/woodleighschool/woodgate/internal/rbac"
	authzapi "github.com/woodleighschool/woodgate/internal/rbac/httpapi"
	"github.com/woodleighschool/woodgate/internal/station"
	stationapi "github.com/woodleighschool/woodgate/internal/station/httpapi"
	"github.com/woodleighschool/woodgate/internal/webui"
	webdist "github.com/woodleighschool/woodgate/web"
)

const gracefulShutdownTimeout = 15 * time.Second

func main() {
	if err := rootCommand().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: "woodgate", Short: "check-in service", Version: buildinfo.Version,
		SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return run(cmd.Context()) },
	}
	root.AddCommand(userCommand(), openAPICommand())
	return root
}

func run(parent context.Context) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	level, err := logging.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}
	logger := logging.New(os.Stdout, level)
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	sessions, sessionStore := newSessions(pool, cfg, logger)
	defer sessionStore.StopCleanup()
	objects, err := bloby.New(ctx, blobydb.New(pool), storageConfig(cfg), logger.With("component", "storage"))
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	app, err := buildApplication(ctx, cfg, pool, sessions, logger, objects)
	if err != nil {
		return fmt.Errorf("build services: %w", err)
	}
	defer app.close()
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", app.server.Addr())
	if err != nil {
		return fmt.Errorf("listen %s: %w", app.server.Addr(), err)
	}
	stopJobs, err := start(ctx, app.starters...)
	if err != nil {
		return fmt.Errorf("start background services: %w", err)
	}
	defer stopJobs()
	return runServer(ctx, app, listener)
}

type application struct {
	server   *api.Server
	station  *station.Server
	starters []starter
}

func (app *application) close() { app.station.Close() }
func (app *application) shutdown(ctx context.Context) error {
	app.close()
	return app.server.Shutdown(ctx)
}

func runServer(ctx context.Context, app *application, listener net.Listener) error {
	errCh := make(chan error, 1)
	go func() { errCh <- app.server.Serve(listener) }()
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), gracefulShutdownTimeout)
		defer cancel()
		if err := app.shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		if err := <-errCh; err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

func buildApplication(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, sessions *scs.SessionManager, logger *slog.Logger, objects *bloby.Service) (*application, error) {
	directoryStore := directory.NewStore(pool)
	users := directory.NewUserService(directoryStore)
	roleStore := rbac.NewStore(pool)
	authzService, err := authz.NewService(roleStore, rbac.Resources())
	if err != nil {
		return nil, fmt.Errorf("create authorization service: %w", err)
	}
	authnService, err := newAuth(ctx, cfg, directoryStore, sessions, authzService, logger.With("component", "auth"))
	if err != nil {
		return nil, err
	}
	checkinService := checkin.NewService(checkin.NewStore(pool, objects), objects)
	stationStore := station.NewStore(pool)
	stationServer, err := station.NewServer(stationStore, station.Dependencies{Locations: checkinService, People: checkinService, Checkins: checkinService, Branding: checkinService}, buildinfo.Version, logger.With("component", "station"))
	if err != nil {
		return nil, fmt.Errorf("configure Station protocol: %w", err)
	}
	stationService := station.NewService(stationStore, checkinService, stationServer, cfg.ServerURL)
	checkinService.SetLocationNotifier(stationService)
	jobs, directorySync, err := newBackgroundJobs(cfg, pool, directoryStore, logger)
	if err != nil {
		stationServer.Close()
		return nil, err
	}
	apiLogger := logger.With("component", "api")
	server := api.NewServer(api.ServerOptions{
		Config: cfg, Ready: pool.Ping, Version: buildinfo.Version, Logger: logger,
		SessionManager: sessions, Authn: authnService, TransferOrigin: objects.TransferOrigin(),
		WebHandler: webui.NewHandler(webui.HandlerOptions{FS: webdist.DistDirFS, Version: buildinfo.Version, ServerURL: cfg.ServerURL, Logger: logger.With("component", "web")}),
		RegisterRoutes: func(routes api.Routes) {
			routes.StorageTransfers.Handle("/storage/*", objects.TransferHandler())
			account.RegisterAPI(routes.App, account.Dependencies{Users: users, Authn: authnService, Authz: authzService, Logger: apiLogger})
			directoryapi.RegisterAPI(routes.App, users, directoryStore, directorySync, authzService, apiLogger)
			authzapi.RegisterAPI(routes.App, roleStore, authzService, apiLogger)
			checkinapi.RegisterAPI(routes.App, checkinapi.Dependencies{Service: checkinService, Authorizer: authzService, Authenticator: authnService, Logger: apiLogger})
			stationapi.RegisterAPI(routes.App, stationService, authzService, apiLogger)
			stationServer.RegisterRoutes(routes.Protocols.Ordinary, routes.Protocols.WebSockets)
		},
	})
	starters := []starter{storageCleanupStarter(objects)}
	if jobs != nil {
		starters = append(starters, backgroundJobsStarter(jobs, logger.With("component", "background_jobs")))
	}
	return &application{server: server, station: stationServer, starters: starters}, nil
}

func newBackgroundJobs(cfg config.Config, pool *pgxpool.Pool, store *directory.Store, logger *slog.Logger) (*backgroundjobs.Runtime, *entra.SyncJobs, error) {
	workers := river.NewWorkers()
	var periodic []*river.PeriodicJob
	service, err := newEntraSyncService(cfg, store, logger)
	if err != nil {
		return nil, nil, err
	}
	if service == nil {
		return nil, entra.NewSyncJobs(false, nil), nil
	}
	if err := river.AddWorkerSafely(workers, entra.NewSyncWorker(service, postgres.NewSessionLocker(pool, entra.SyncAdvisoryLockID))); err != nil {
		return nil, nil, fmt.Errorf("register Entra sync worker: %w", err)
	}
	periodic = append(periodic, periodicJob(entra.SyncJobKind, cfg.EntraSyncInterval, func() river.JobArgs { return entra.SyncJobArgs{Trigger: backgroundjobs.TriggerScheduled} }))
	jobs, err := backgroundjobs.New(pool, workers, periodic, logger.With("component", "background_jobs"))
	if err != nil {
		return nil, nil, err
	}
	return jobs, entra.NewSyncJobs(true, jobs), nil
}

func newEntraSyncService(cfg config.Config, store *directory.Store, logger *slog.Logger) (*entra.Service, error) {
	if !cfg.EntraEnabled() {
		return nil, nil
	}
	client, err := entra.NewClient(entra.Config{TenantID: cfg.EntraTenantID, ClientID: cfg.EntraClientID, ClientSecret: cfg.EntraClientSecret, TransitiveGroups: cfg.EntraTransitiveGroups})
	if err != nil {
		return nil, fmt.Errorf("configure Entra sync: %w", err)
	}
	return entra.NewService(store, client, logger.With("component", "entra")), nil
}

func newAuth(ctx context.Context, cfg config.Config, store *directory.Store, sessions *scs.SessionManager, authorization *authz.Service, logger *slog.Logger) (*authn.Service, error) {
	authConfig := authn.Config{Admit: authorization.HasAccess, SuccessRedirect: "/checkins", FailureRedirect: "/login", Logger: logger}
	if cfg.OIDCEnabled() {
		authConfig.OIDC = &authn.OIDCConfig{IssuerURL: cfg.OIDCIssuerURL, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret, RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes, EmailClaim: cfg.OIDCEmailClaim}
	}
	service, err := authn.New(ctx, directory.NewAuthnStore(store), sessions, authConfig)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	return service, nil
}

func newSessions(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(pool)
	sessions := scs.New()
	sessions.ErrorFunc = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.ErrorContext(r.Context(), "session persistence failed", "operation", "session", "err", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	sessions.Store = store
	sessions.HashTokenInStore = true
	sessions.Lifetime = config.SessionLifetime
	sessions.Cookie.Name = "woodgate_session"
	sessions.Cookie.Path = "/"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = cfg.SessionCookieSecure
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Persist = true
	return sessions, store
}

func storageConfig(cfg config.Config) bloby.Config {
	return bloby.Config{Kind: bloby.Kind(cfg.StorageKind), TransferTTL: cfg.StorageTransferTTL,
		File: bloby.FileConfig{Root: cfg.StorageFileRoot, BaseURL: cfg.ServerURL, CapabilityKeyHex: cfg.StorageCapabilityKey},
		S3:   bloby.S3Config{Bucket: cfg.StorageS3Bucket, Region: cfg.StorageS3Region, Endpoint: cfg.StorageS3Endpoint, AccessKey: cfg.StorageS3AccessKey, SecretKey: cfg.StorageS3SecretKey, PathStyle: cfg.StorageS3PathStyle}}
}

type starter func(context.Context) (func(), error)

func start(ctx context.Context, starters ...starter) (func(), error) {
	var stops []func()
	for _, start := range starters {
		if start == nil {
			continue
		}
		stop, err := start(ctx)
		if err != nil {
			for _, stop := range slices.Backward(stops) {
				stop()
			}
			return nil, err
		}
		if stop != nil {
			stops = append(stops, stop)
		}
	}
	return func() {
		for _, stop := range slices.Backward(stops) {
			stop()
		}
	}, nil
}

func storageCleanupStarter(objects *bloby.Service) starter {
	return func(ctx context.Context) (func(), error) {
		ctx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() { defer close(done); objects.RunCleanup(ctx) }()
		return func() { cancel(); <-done }, nil
	}
}

func periodicJob(id string, interval time.Duration, args func() river.JobArgs) *river.PeriodicJob {
	return river.NewPeriodicJob(river.PeriodicInterval(interval), func() (river.JobArgs, *river.InsertOpts) { return args(), nil }, &river.PeriodicJobOpts{ID: id, RunOnStart: true})
}

func backgroundJobsStarter(jobs *backgroundjobs.Runtime, logger *slog.Logger) starter {
	return func(ctx context.Context) (func(), error) {
		if err := jobs.Start(ctx); err != nil {
			return nil, err
		}
		return func() {
			stopCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
			defer cancel()
			if err := jobs.Stop(stopCtx); err != nil {
				logger.WarnContext(stopCtx, "stop background jobs", "err", err)
			}
		}, nil
	}
}
