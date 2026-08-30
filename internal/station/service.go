package station

import (
	"context"
	"errors"
	"fmt"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/randtoken"
)

const stationSecretBytes = 32

// ErrUnauthorized reports an invalid or disabled Station credential.
var ErrUnauthorized = errors.New("station unauthorized")

// LocationProvider resolves the location configuration owned by the application.
type LocationProvider interface {
	GetStationLocation(context.Context, int64) (*LocationConfiguration, error)
}

type stationRepository interface {
	List(context.Context, StationListParams) ([]Station, int, error)
	Get(context.Context, int64) (*Station, error)
	create(context.Context, StationMutation, string, string) (*Station, error)
	update(context.Context, int64, StationMutation) (*Station, error)
	delete(context.Context, int64) error
	rotateSecret(context.Context, int64, string, string) (*Station, error)
}

type connectionControl interface {
	ConfigurationChanged(int64, int64)
	ConfigurationChangedForLocation(int64)
	Disconnect(int64, string)
}

// Service owns station validation, one-time secrets, and connection invalidation.
type Service struct {
	store       stationRepository
	locations   LocationProvider
	connections connectionControl
}

// NewService returns the administrative station service.
func NewService(store *Store, locations LocationProvider, connections *Server) *Service {
	return newService(store, locations, connections)
}

func newService(
	store stationRepository,
	locations LocationProvider,
	connections connectionControl,
) *Service {
	return &Service{store: store, locations: locations, connections: connections}
}

// List returns administratively visible stations.
func (s *Service) List(ctx context.Context, params StationListParams) ([]Station, int, error) {
	params.normalize()
	if err := params.validate(); err != nil {
		return nil, 0, err
	}
	return s.store.List(ctx, params)
}

// Get returns one station.
func (s *Service) Get(ctx context.Context, id int64) (*Station, error) {
	return s.store.Get(ctx, id)
}

// Create persists a station and reveals its secret once.
func (s *Service) Create(ctx context.Context, mutation StationMutation) (*StationSecret, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	if err := s.requireLocation(ctx, mutation.LocationID); err != nil {
		return nil, err
	}
	secret, prefix, hash, err := newSecret()
	if err != nil {
		return nil, err
	}
	created, err := s.store.create(ctx, mutation, prefix, hash)
	if err != nil {
		return nil, err
	}
	return &StationSecret{Station: *created, Secret: secret}, nil
}

// Update replaces the administrative station configuration.
func (s *Service) Update(
	ctx context.Context,
	id int64,
	mutation StationMutation,
) (*Station, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	if err := s.requireLocation(ctx, mutation.LocationID); err != nil {
		return nil, err
	}
	updated, err := s.store.update(ctx, id, mutation)
	if err != nil {
		return nil, err
	}
	if s.connections != nil {
		if updated.Enabled {
			s.connections.ConfigurationChanged(id, updated.LocationID)
		} else {
			s.connections.Disconnect(id, "station disabled")
		}
	}
	return updated, nil
}

// Delete removes a station and disconnects its companion app.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.store.delete(ctx, id); err != nil {
		return err
	}
	if s.connections != nil {
		s.connections.Disconnect(id, "station deleted")
	}
	return nil
}

// RotateSecret invalidates the old credential and reveals the replacement once.
func (s *Service) RotateSecret(ctx context.Context, id int64) (*StationSecret, error) {
	secret, prefix, hash, err := newSecret()
	if err != nil {
		return nil, err
	}
	updated, err := s.store.rotateSecret(ctx, id, prefix, hash)
	if err != nil {
		return nil, err
	}
	if s.connections != nil {
		s.connections.Disconnect(id, "station secret rotated")
	}
	return &StationSecret{Station: *updated, Secret: secret}, nil
}

// LocationChanged notifies every connected station bound to locationID.
func (s *Service) LocationChanged(locationID int64) {
	if s.connections != nil {
		s.connections.ConfigurationChangedForLocation(locationID)
	}
}

func (s *Service) requireLocation(ctx context.Context, locationID int64) error {
	if s.locations == nil {
		return errors.New("station location provider is required")
	}
	if _, err := s.locations.GetStationLocation(ctx, locationID); err != nil {
		if errors.Is(err, fault.ErrNotFound) {
			return fault.ErrNotFound
		}
		return fmt.Errorf("get station location: %w", err)
	}
	return nil
}

func newSecret() (string, string, string, error) {
	token, err := randtoken.Generate(stationSecretBytes)
	if err != nil {
		return "", "", "", fmt.Errorf("generate station secret: %w", err)
	}
	secret := "wgst_" + token
	prefix, err := secretPrefix(secret)
	if err != nil {
		return "", "", "", err
	}
	return secret, prefix, hashSecret(secret), nil
}
