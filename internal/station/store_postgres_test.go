//go:build postgres

package station

import (
	"context"
	"errors"
	"testing"

	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestStoreProjectsSharedClientStateAndInvalidatesItOnKeyRotation(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	otherStore := NewStore(db)

	var locationID int64
	if err := db.QueryRow(ctx, `INSERT INTO locations (name, enabled)
VALUES ('Reception', true)
RETURNING id`).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}

	oldKey := "wgst_old"
	created, err := store.create(ctx, StationMutation{
		Name: "Reception iPad", LocationID: locationID, Enabled: true,
	}, hashSecret(oldKey))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Location.ID != locationID || created.Location.Name != "Reception" {
		t.Fatalf("location projection = %+v", created.Location)
	}
	assertStationClientState(t, created, StateOffline, "", nil)

	rejectedProtocol := 2
	if err := store.ObserveRejectedClient(
		ctx, created.ID, oldKey, &rejectedProtocol, "2.0.0+4",
	); err != nil {
		t.Fatalf("ObserveRejectedClient: %v", err)
	}
	rejected := getStation(t, ctx, otherStore, created.ID)
	assertStationClientState(t, rejected, StateIncompatible, "2.0.0+4", &rejectedProtocol)

	if err := store.claimClientSession(ctx, created.ID, oldKey, "current", 1, "1.3.2+7"); err != nil {
		t.Fatalf("claimClientSession: %v", err)
	}
	connected := getStation(t, ctx, otherStore, created.ID)
	onlineProtocol := 1
	assertStationClientState(t, connected, StateOnline, "1.3.2+7", &onlineProtocol)
	if err := store.ObserveRejectedClient(
		ctx, created.ID, oldKey, &rejectedProtocol, "2.0.0+5",
	); err != nil {
		t.Fatalf("observe rejected client during session: %v", err)
	}
	connected = getStation(t, ctx, otherStore, created.ID)
	assertStationClientState(t, connected, StateOnline, "1.3.2+7", &onlineProtocol)

	if err := store.renewClientSession(ctx, created.ID, "current"); err != nil {
		t.Fatalf("renewClientSession: %v", err)
	}
	if err := store.releaseClientSession(ctx, created.ID, "superseded"); err != nil {
		t.Fatalf("release superseded session: %v", err)
	}
	assertStationClientState(
		t, getStation(t, ctx, otherStore, created.ID), StateOnline, "1.3.2+7", &onlineProtocol,
	)
	if err := store.releaseClientSession(ctx, created.ID, "current"); err != nil {
		t.Fatalf("release current session: %v", err)
	}
	rejected = getStation(t, ctx, otherStore, created.ID)
	assertStationClientState(t, rejected, StateIncompatible, "2.0.0+5", &rejectedProtocol)
	if err := store.claimClientSession(ctx, created.ID, oldKey, "replacement", 1, "1.3.2+8"); err != nil {
		t.Fatalf("claim replacement session: %v", err)
	}
	assertKeyRotationInvalidatesSession(t, ctx, store, created, oldKey)
}

func assertKeyRotationInvalidatesSession(
	t *testing.T,
	ctx context.Context,
	store *Store,
	station *Station,
	oldKey string,
) {
	t.Helper()
	newKey := "wgst_new"
	rotated, err := store.rotateKey(ctx, station.ID, hashSecret(newKey))
	if err != nil {
		t.Fatalf("rotateKey: %v", err)
	}
	assertStationClientState(t, rotated, StateOffline, "", nil)
	if err := store.renewClientSession(ctx, station.ID, "replacement"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("renew rotated session error = %v, want session invalid", err)
	}
	if _, err := store.authenticate(ctx, oldKey); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old key authentication error = %v, want unauthorized", err)
	}
	authenticated, err := store.authenticate(ctx, newKey)
	if err != nil {
		t.Fatalf("new key authentication: %v", err)
	}
	if authenticated.ID != station.ID || authenticated.Location.Name != "Reception" {
		t.Fatalf("authenticated station = %+v", authenticated)
	}
}

func TestStoreExpiresClientState(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var locationID int64
	if err := db.QueryRow(ctx, `INSERT INTO locations (name, enabled)
VALUES ('Reception', true)
RETURNING id`).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	key := "wgst_test"
	created, err := store.create(ctx, StationMutation{
		Name: "Reception iPad", LocationID: locationID, Enabled: true,
	}, hashSecret(key))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.claimClientSession(ctx, created.ID, key, "expired", 1, "1.3.2+7"); err != nil {
		t.Fatalf("claimClientSession: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE station_sessions
SET expires_at = now() - interval '1 second'
WHERE station_id = $1`, created.ID); err != nil {
		t.Fatalf("expire session: %v", err)
	}
	if state := getStation(t, ctx, store, created.ID).State; state != StateOffline {
		t.Fatalf("expired state = %q, want offline", state)
	}
	if err := store.renewClientSession(ctx, created.ID, "expired"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("renew expired session error = %v, want session invalid", err)
	}
}

func getStation(t *testing.T, ctx context.Context, store *Store, id int64) *Station {
	t.Helper()
	item, err := store.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get(%d): %v", id, err)
	}
	return item
}

func assertStationClientState(
	t *testing.T,
	station *Station,
	wantState State,
	wantVersion string,
	wantProtocol *int,
) {
	t.Helper()
	if station.State != wantState || station.Version != wantVersion {
		t.Fatalf("client state = %q/%q/%v, want %q/%q/%v",
			station.State, station.Version, station.ProtocolVersion,
			wantState, wantVersion, wantProtocol)
	}
	if wantProtocol == nil && station.ProtocolVersion != nil ||
		wantProtocol != nil && (station.ProtocolVersion == nil || *station.ProtocolVersion != *wantProtocol) {
		t.Fatalf("client protocol = %v, want %v", station.ProtocolVersion, wantProtocol)
	}
}
