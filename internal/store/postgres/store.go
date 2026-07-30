package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/woodleighschool/woodgate/internal/config"
	"github.com/woodleighschool/woodgate/internal/store/db"
	"github.com/woodleighschool/woodgate/internal/store/migrations"
)

const databasePingTimeout = 5 * time.Second

// Store owns the Postgres connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a Postgres store, runs migrations, and verifies connectivity.
func New(ctx context.Context, cfg config.DatabaseConfig) (*Store, error) {
	connectionString := ConnectionString(cfg)

	if err := runMigrations(connectionString); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, databasePingTimeout)
	defer cancel()
	if pingErr := pool.Ping(pingContext); pingErr != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", pingErr)
	}

	return &Store{pool: pool}, nil
}

// Ping checks database connectivity for readiness probes.
func (store *Store) Ping(ctx context.Context) error {
	return store.pool.Ping(ctx)
}

// Close closes the Postgres connection pool.
func (store *Store) Close() {
	store.pool.Close()
}

// Queries returns a sqlc query runner bound to the store pool.
func (store *Store) Queries() *db.Queries {
	return db.New(store.pool)
}

// Pool exposes the underlying pgx pool for read queries that are awkward to express via sqlc.
func (store *Store) Pool() *pgxpool.Pool {
	return store.pool
}

// RunInTx executes the given function in a database transaction.
func (store *Store) RunInTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	queries := db.New(tx)
	if runErr := fn(queries); runErr != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("run transaction: %w (rollback: %w)", runErr, rollbackErr)
		}
		return runErr
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}

	return nil
}

// ConnectionString builds a canonical Postgres URI from environment configuration.
func ConnectionString(cfg config.DatabaseConfig) string {
	query := url.Values{}
	query.Set("sslmode", cfg.SSLMode)

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Path:     cfg.Name,
		RawQuery: query.Encode(),
	}).String()
}

func runMigrations(connectionString string) error {
	sqlDB, err := openSQLDB(connectionString)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if upErr := migrations.Up(sqlDB); upErr != nil {
		return fmt.Errorf("apply migrations: %w", upErr)
	}

	return nil
}

func openSQLDB(connectionString string) (*sql.DB, error) {
	pgxConfig, err := pgx.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parse connection string: %w", err)
	}

	sqlDB := stdlib.OpenDB(*pgxConfig)
	pingContext, cancel := context.WithTimeout(context.Background(), databasePingTimeout)
	defer cancel()

	if pingErr := sqlDB.PingContext(pingContext); pingErr != nil {
		if closeErr := sqlDB.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping database before migrations: %w (close: %w)", pingErr, closeErr)
		}
		return nil, fmt.Errorf("ping database before migrations: %w", pingErr)
	}

	return sqlDB, nil
}
