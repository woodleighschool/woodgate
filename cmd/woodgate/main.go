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

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/auth"
	authapi "github.com/woodleighschool/woodgate/internal/auth/httpapi"
	"github.com/woodleighschool/woodgate/internal/authorization"
	authorizationapi "github.com/woodleighschool/woodgate/internal/authorization/httpapi"
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
	"github.com/woodleighschool/woodgate/internal/station"
	stationapi "github.com/woodleighschool/woodgate/internal/station/httpapi"
	"github.com/woodleighschool/woodgate/internal/storage"
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
	slog.SetDefault(logger)
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	sessions, sessionStore := newSessions(pool, cfg)
	defer sessionStore.StopCleanup()
	storageBackend, err := storage.New(ctx, storageConfig(cfg))
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	app, err := buildApplication(ctx, cfg, pool, sessions, logger, storageBackend)
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

func buildApplication(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, sessions *scs.SessionManager, logger *slog.Logger, storageBackend storage.Backend) (*application, error) {
	storageLogger := logger.With("component", "storage")
	objects := storage.NewObjectStore(pool, storageBackend, storageLogger)
	ingestor := storage.NewIngestor(objects, storageBackend)
	delivery := storage.NewDelivery(storageBackend)
	directoryStore := directory.NewStore(pool)
	users := directory.NewUserService(directoryStore)
	authService, err := newAuth(ctx, cfg, users, sessions)
	if err != nil {
		return nil, err
	}
	authorizationService := authorization.NewService(authorization.NewStore(pool))
	checkinService := checkin.NewService(checkin.NewStore(pool, objects), objects, ingestor, delivery)
	stationStore := station.NewStore(pool)
	stationServer, err := station.NewServer(stationStore, station.Dependencies{Locations: checkinService, People: checkinService, Checkins: checkinService, Branding: checkinService}, buildinfo.Version, logger.With("component", "station"))
	if err != nil {
		return nil, fmt.Errorf("configure Station protocol: %w", err)
	}
	stationService := station.NewService(stationStore, checkinService, stationServer)
	checkinService.SetLocationNotifier(stationService)
	jobs, directorySync, err := newBackgroundJobs(cfg, pool, directoryStore, logger)
	if err != nil {
		stationServer.Close()
		return nil, err
	}
	apiLogger := logger.With("component", "api")
	server := api.NewServer(api.ServerOptions{
		Config: cfg, Ready: pool.Ping, Version: buildinfo.Version, Logger: logger,
		SessionManager: sessions, AuthService: authService, TransferOrigin: storageBackend.TransferOrigin(),
		WebHandler: webui.NewHandler(webui.HandlerOptions{FS: webdist.DistDirFS, Version: buildinfo.Version, ServerURL: cfg.ServerURL, Logger: logger.With("component", "web")}),
		RegisterRoutes: func(routes api.Routes) {
			storage.RegisterTransferRoutes(routes.StorageTransfers, storageBackend, storageLogger)
			authapi.RegisterAPI(routes.App, authapi.Dependencies{AuthService: authService, Users: users, Permissions: authorizationService, Logger: apiLogger})
			directoryapi.RegisterAPI(routes.App, users, directoryStore, directorySync, authorizationService, apiLogger)
			authorizationapi.RegisterAPI(routes.App, authorizationService, apiLogger)
			checkinapi.RegisterAPI(routes.App, checkinapi.Dependencies{Service: checkinService, Authorizer: authorizationService, Authenticator: authService, Logger: apiLogger})
			stationapi.RegisterAPI(routes.App, stationService, authorizationService, apiLogger)
			stationServer.RegisterRoutes(routes.Protocols.Ordinary, routes.Protocols.WebSockets)
		},
	})
	starters := []starter{storageUploadCleanupStarter(ingestor, cfg.StorageTransferTTL, storageLogger)}
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

func newAuth(ctx context.Context, cfg config.Config, users *directory.UserService, sessions *scs.SessionManager) (*auth.Service, error) {
	service, err := auth.NewService(users, sessions)
	if err != nil {
		return nil, fmt.Errorf("configure authentication: %w", err)
	}
	if !cfg.OIDCEnabled() {
		return service, nil
	}
	if err := service.ConfigureOIDC(ctx, auth.OIDCConfig{IssuerURL: cfg.OIDCIssuerURL, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret, RedirectURL: cfg.OIDCRedirectURL, Scopes: cfg.OIDCScopes, EmailClaim: cfg.OIDCEmailClaim}); err != nil {
		return nil, fmt.Errorf("configure OIDC: %w", err)
	}
	return service, nil
}

func newSessions(pool *pgxpool.Pool, cfg config.Config) (*scs.SessionManager, *pgxstore.PostgresStore) {
	store := pgxstore.New(pool)
	sessions := scs.New()
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

func storageConfig(cfg config.Config) storage.Config {
	return storage.Config{Kind: storage.Kind(cfg.StorageKind), TransferTTL: cfg.StorageTransferTTL,
		File: storage.FileConfig{Root: cfg.StorageFileRoot, BaseURL: cfg.ServerURL, CapabilityKeyHex: cfg.StorageCapabilityKey},
		S3:   storage.S3Config{Bucket: cfg.StorageS3Bucket, Region: cfg.StorageS3Region, Endpoint: cfg.StorageS3Endpoint, AccessKey: cfg.StorageS3AccessKey, SecretKey: cfg.StorageS3SecretKey, PathStyle: cfg.StorageS3PathStyle}}
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

func storageUploadCleanupStarter(ingestor *storage.Ingestor, ttl time.Duration, logger *slog.Logger) starter {
	return func(ctx context.Context) (func(), error) {
		cleanup := storage.StartUploadCleanup(ctx, ingestor, ttl, logger)
		return cleanup.Stop, nil
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
