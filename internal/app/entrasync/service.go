package entrasync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	graphsync "github.com/woodleighschool/go-entrasync"
)

// GraphClient reads Entra objects from Microsoft Graph.
type GraphClient interface {
	Snapshot(ctx context.Context) (*graphsync.Snapshot, error)
}

// DataStore persists a full Entra snapshot.
type DataStore interface {
	ReconcileSnapshot(ctx context.Context, snapshot *graphsync.Snapshot) (Result, error)
}

// Result reports reconciled object counts.
type Result struct {
	Users       int
	Groups      int
	Memberships int
}

// Service runs Entra snapshot synchronization.
type Service struct {
	logger    *slog.Logger
	client    GraphClient
	dataStore DataStore
	interval  time.Duration
}

// New creates an Entra synchronization service.
func New(logger *slog.Logger, client GraphClient, dataStore DataStore, interval time.Duration) *Service {
	return &Service{
		logger:    logger,
		client:    client,
		dataStore: dataStore,
		interval:  interval,
	}
}

// Run executes an immediate sync, then continues on the configured interval.
func (service *Service) Run(ctx context.Context) {
	service.syncAndLog(ctx)

	ticker := time.NewTicker(service.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			service.logger.InfoContext(ctx, "entra sync stopped")
			return
		case <-ticker.C:
			service.syncAndLog(ctx)
		}
	}
}

// SyncOnce fetches a snapshot from Graph and reconciles it into storage.
func (service *Service) SyncOnce(ctx context.Context) (Result, error) {
	snapshot, err := service.client.Snapshot(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("fetch graph snapshot: %w", err)
	}

	result, err := service.dataStore.ReconcileSnapshot(ctx, snapshot)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile snapshot: %w", err)
	}

	return result, nil
}

func (service *Service) syncAndLog(ctx context.Context) {
	started := time.Now()

	result, err := service.SyncOnce(ctx)
	if err != nil {
		service.logger.ErrorContext(ctx, "entra sync failed", "error", err, "duration", time.Since(started))
		return
	}

	service.logger.InfoContext(
		ctx,
		"entra sync complete",
		"users",
		result.Users,
		"groups",
		result.Groups,
		"memberships",
		result.Memberships,
		"duration",
		time.Since(started),
	)
}
