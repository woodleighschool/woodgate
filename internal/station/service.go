package station

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/randtoken"
)

const stationKeyBytes = 32

// ErrUnauthorized reports an invalid or disabled Station credential.
var ErrUnauthorized = errors.New("station unauthorized")

// LocationProvider resolves the location configuration owned by the application.
type LocationProvider interface {
	GetStationLocation(context.Context, int64) (*LocationConfiguration, error)
}

// LocationCatalog adds the location choices needed by Station administration.
type LocationCatalog interface {
	LocationProvider
	ListStationLocations(context.Context, listing.Params) ([]Location, int, error)
}

type stationRepository interface {
	List(context.Context, StationListParams) ([]Station, int, error)
	Get(context.Context, int64) (*Station, error)
	create(context.Context, StationMutation, string) (*Station, error)
	update(context.Context, int64, StationMutation) (*Station, error)
	delete(context.Context, int64) error
	rotateKey(context.Context, int64, string) (*Station, error)
}

type connectionControl interface {
	ConfigurationChanged(int64, int64)
	ConfigurationChangedForLocation(int64)
	Disconnect(int64, string)
}

// Service owns station validation, one-time keys, and connection invalidation.
type Service struct {
	store       stationRepository
	locations   LocationCatalog
	connections connectionControl
	serverURL   string
}

// NewService returns the administrative station service.
func NewService(store *Store, locations LocationCatalog, connections *Server, serverURL string) *Service {
	return newService(store, locations, connections, serverURL)
}

func newService(
	store stationRepository,
	locations LocationCatalog,
	connections connectionControl,
	serverURL string,
) *Service {
	return &Service{store: store, locations: locations, connections: connections, serverURL: serverURL}
}

// ListLocationChoices returns location identities assignable to a Station.
func (s *Service) ListLocationChoices(
	ctx context.Context,
	params listing.Params,
) ([]Location, int, error) {
	params = listing.Normalize(params)
	if err := listing.Validate(params); err != nil {
		return nil, 0, err
	}
	return s.locations.ListStationLocations(ctx, params)
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

// Create persists a station and reveals its pairing configuration once.
func (s *Service) Create(ctx context.Context, mutation StationMutation) (*Pairing, error) {
	mutation.normalize()
	if err := mutation.validate(); err != nil {
		return nil, err
	}
	if err := s.requireLocation(ctx, mutation.LocationID); err != nil {
		return nil, err
	}
	key, hash, err := newKey()
	if err != nil {
		return nil, err
	}
	created, err := s.store.create(ctx, mutation, hash)
	if err != nil {
		return nil, err
	}
	return s.pairing(*created, key), nil
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

// RotateKey invalidates the old key and reveals the replacement pairing configuration once.
func (s *Service) RotateKey(ctx context.Context, id int64) (*Pairing, error) {
	key, hash, err := newKey()
	if err != nil {
		return nil, err
	}
	updated, err := s.store.rotateKey(ctx, id, hash)
	if err != nil {
		return nil, err
	}
	if s.connections != nil {
		s.connections.Disconnect(id, "station key rotated")
	}
	return s.pairing(*updated, key), nil
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

func newKey() (string, string, error) {
	token, err := randtoken.Generate(stationKeyBytes)
	if err != nil {
		return "", "", fmt.Errorf("generate station key: %w", err)
	}
	key := "wgst_" + token
	return key, hashSecret(key), nil
}

func (s *Service) pairing(item Station, key string) *Pairing {
	query := url.Values{}
	query.Set("server", s.serverURL)
	query.Set("key", key)
	pairingURL := url.URL{Scheme: "woodgate", Host: "pair", RawQuery: query.Encode()}
	return &Pairing{Station: item, URL: pairingURL.String(), ServerURL: s.serverURL, Key: key}
}
