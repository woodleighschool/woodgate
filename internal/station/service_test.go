package station

import (
	"context"
	"testing"

	"github.com/woodleighschool/woodgate/internal/listing"
)

type serviceStoreFake struct {
	createdPrefix string
	createdHash   string
	rotatedPrefix string
	rotatedHash   string
	station       Station
}

func (f *serviceStoreFake) List(context.Context, StationListParams) ([]Station, int, error) {
	return []Station{f.station}, 1, nil
}

func (f *serviceStoreFake) Get(context.Context, int64) (*Station, error) {
	station := f.station
	return &station, nil
}

func (f *serviceStoreFake) create(
	_ context.Context,
	mutation StationMutation,
	prefix string,
	hash string,
) (*Station, error) {
	f.createdPrefix = prefix
	f.createdHash = hash
	f.station = Station{ID: 7, Name: mutation.Name, LocationID: mutation.LocationID, Enabled: mutation.Enabled}
	station := f.station
	return &station, nil
}

func (f *serviceStoreFake) update(
	_ context.Context,
	id int64,
	mutation StationMutation,
) (*Station, error) {
	f.station = Station{ID: id, Name: mutation.Name, LocationID: mutation.LocationID, Enabled: mutation.Enabled}
	station := f.station
	return &station, nil
}

func (*serviceStoreFake) delete(context.Context, int64) error { return nil }

func (f *serviceStoreFake) rotateSecret(
	_ context.Context,
	id int64,
	prefix string,
	hash string,
) (*Station, error) {
	f.rotatedPrefix = prefix
	f.rotatedHash = hash
	f.station.ID = id
	station := f.station
	return &station, nil
}

type locationProviderFake struct{}

func (locationProviderFake) GetStationLocation(_ context.Context, id int64) (*LocationConfiguration, error) {
	return &LocationConfiguration{ID: id, Name: "Reception", Enabled: true}, nil
}

type connectionControlFake struct {
	changed          []int64
	changedLocations []int64
	disconnected     []int64
}

func (f *connectionControlFake) ConfigurationChanged(id int64, locationID int64) {
	f.changed = append(f.changed, id)
	f.changedLocations = append(f.changedLocations, locationID)
}

func (*connectionControlFake) ConfigurationChangedForLocation(int64) {}

func (f *connectionControlFake) Disconnect(id int64, _ string) {
	f.disconnected = append(f.disconnected, id)
}

func TestServiceRevealsSecretsOnlyAcrossCreateAndRotate(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections)

	created, err := service.Create(t.Context(), StationMutation{
		Name:       "  Reception  ",
		LocationID: 11,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Station.Name != "Reception" || created.Secret == "" {
		t.Fatalf("Create returned %#v", created)
	}
	if store.createdPrefix == "" || store.createdHash == "" ||
		store.createdPrefix == created.Secret || store.createdHash == created.Secret {
		t.Fatalf("stored credential = prefix %q hash %q, secret %q", store.createdPrefix, store.createdHash, created.Secret)
	}

	rotated, err := service.RotateSecret(t.Context(), created.Station.ID)
	if err != nil {
		t.Fatalf("RotateSecret: %v", err)
	}
	if rotated.Secret == "" || rotated.Secret == created.Secret {
		t.Fatalf("rotated secret = %q, original %q", rotated.Secret, created.Secret)
	}
	if len(connections.disconnected) != 1 || connections.disconnected[0] != created.Station.ID {
		t.Fatalf("disconnects = %#v", connections.disconnected)
	}

	listed, _, err := service.List(t.Context(), StationListParams{ListParams: listing.Params{PageSize: 1}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 || listed[0].SecretPrefix != "" {
		t.Fatalf("listed stations = %#v", listed)
	}
}

func TestServiceDisconnectsDisabledStation(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections)

	_, err := service.Update(t.Context(), 17, StationMutation{
		Name:       "Reception",
		LocationID: 11,
		Enabled:    false,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(connections.disconnected) != 1 || connections.disconnected[0] != 17 {
		t.Fatalf("disconnects = %#v", connections.disconnected)
	}
	if len(connections.changed) != 0 {
		t.Fatalf("configuration changes = %#v", connections.changed)
	}
}

func TestServiceUpdatesConnectedStationLocationBinding(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections)

	_, err := service.Update(t.Context(), 17, StationMutation{
		Name:       "Reception",
		LocationID: 29,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(connections.changed) != 1 || connections.changed[0] != 17 ||
		len(connections.changedLocations) != 1 || connections.changedLocations[0] != 29 {
		t.Fatalf("configuration changes = %#v at %#v", connections.changed, connections.changedLocations)
	}
}
