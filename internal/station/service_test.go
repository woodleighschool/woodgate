package station

import (
	"context"
	"net/url"
	"testing"

	"github.com/woodleighschool/woodgate/internal/listing"
)

type serviceStoreFake struct {
	createdHash string
	deleted     []int64
	rotatedHash string
	station     Station
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
	hash string,
) (*Station, error) {
	f.createdHash = hash
	f.station = Station{ID: 7, Name: mutation.Name, LocationID: mutation.LocationID, Location: Location{ID: mutation.LocationID, Name: "Reception"}, Enabled: mutation.Enabled, State: StateOffline}
	station := f.station
	return &station, nil
}

func (f *serviceStoreFake) update(
	_ context.Context,
	id int64,
	mutation StationMutation,
) (*Station, error) {
	f.station = Station{ID: id, Name: mutation.Name, LocationID: mutation.LocationID, Enabled: mutation.Enabled, State: StateOffline}
	station := f.station
	return &station, nil
}

func (f *serviceStoreFake) delete(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *serviceStoreFake) rotateKey(
	_ context.Context,
	id int64,
	hash string,
) (*Station, error) {
	f.rotatedHash = hash
	f.station.ID = id
	station := f.station
	return &station, nil
}

type locationProviderFake struct{}

func (locationProviderFake) GetStationLocation(_ context.Context, id int64) (*LocationConfiguration, error) {
	return &LocationConfiguration{ID: id, Name: "Reception", Enabled: true}, nil
}

func (locationProviderFake) ListStationLocations(
	context.Context,
	listing.Params,
) ([]Location, int, error) {
	return []Location{{ID: 11, Name: "Reception"}}, 1, nil
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

func TestServiceRevealsPairingOnlyAcrossCreateAndRotate(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections, "https://gate.example.invalid")
	locations, count, err := service.ListLocationChoices(t.Context(), listing.Params{PageSize: 20})
	if err != nil || count != 1 || len(locations) != 1 || locations[0].Name != "Reception" {
		t.Fatalf("location choices = %+v count %d error %v", locations, count, err)
	}

	created, err := service.Create(t.Context(), StationMutation{
		Name:       "  Reception  ",
		LocationID: 11,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Station.Name != "Reception" || created.Key == "" || created.ServerURL != "https://gate.example.invalid" {
		t.Fatalf("Create returned %#v", created)
	}
	if store.createdHash == "" || store.createdHash == created.Key {
		t.Fatalf("stored credential = hash %q, key %q", store.createdHash, created.Key)
	}
	pairingURL, err := url.Parse(created.URL)
	if err != nil {
		t.Fatalf("parse pairing URL: %v", err)
	}
	if pairingURL.Scheme != "woodgate" || pairingURL.Host != "pair" ||
		pairingURL.Query().Get("server") != created.ServerURL || pairingURL.Query().Get("key") != created.Key {
		t.Fatalf("pairing URL = %q", created.URL)
	}

	rotated, err := service.RotateKey(t.Context(), created.Station.ID)
	if err != nil {
		t.Fatalf("RotateKey: %v", err)
	}
	if rotated.Key == "" || rotated.Key == created.Key {
		t.Fatalf("rotated key = %q, original %q", rotated.Key, created.Key)
	}
	if len(connections.disconnected) != 1 || connections.disconnected[0] != created.Station.ID {
		t.Fatalf("disconnects = %#v", connections.disconnected)
	}

	listed, _, err := service.List(t.Context(), StationListParams{ListParams: listing.Params{PageSize: 1}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 || listed[0].State != StateOffline {
		t.Fatalf("listed stations = %#v", listed)
	}
}

func TestServiceDisconnectsDisabledStation(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections, "https://gate.example.invalid")

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

func TestServiceDeletesAndDisconnectsStation(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections, "https://gate.example.invalid")

	if err := service.Delete(t.Context(), 17); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != 17 {
		t.Fatalf("deleted stations = %#v", store.deleted)
	}
	if len(connections.disconnected) != 1 || connections.disconnected[0] != 17 {
		t.Fatalf("disconnects = %#v", connections.disconnected)
	}
}

func TestServiceUpdatesConnectedStationLocationBinding(t *testing.T) {
	store := &serviceStoreFake{}
	connections := &connectionControlFake{}
	service := newService(store, locationProviderFake{}, connections, "https://gate.example.invalid")

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
