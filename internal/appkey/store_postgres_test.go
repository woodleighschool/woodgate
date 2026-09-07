//go:build postgres

package appkey

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestImportedKeyHashAndLocationScope(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)
	appID := uuid.MustParse("30000000-0000-0000-0000-000000000001")
	secret := "woodgate_" + t.Name()
	hash := sha256.Sum256([]byte(secret))
	var id int64
	if err := db.QueryRow(ctx, "INSERT INTO app_keys(app_id,name,key_prefix,key_hash,all_locations) VALUES($1,'Imported key','woodgate_syn',$2,true) RETURNING id", appID, hex.EncodeToString(hash[:])).Scan(&id); err != nil {
		t.Fatal(err)
	}
	key, err := store.Authenticate(ctx, "  "+secret+" \n")
	if err != nil {
		t.Fatal(err)
	}
	if key.AppID != appID || !key.AllLocations || key.LastUsedAt == nil {
		t.Fatalf("authenticated=%+v", key)
	}
	var locationID int64
	if err := db.QueryRow(ctx, "INSERT INTO locations(name) VALUES('New location') RETURNING id").Scan(&locationID); err != nil {
		t.Fatal(err)
	}
	if !key.Allows(locationID) {
		t.Fatal("all-locations key excludes new location")
	}
	if _, err := store.Update(ctx, id, Mutation{Name: "Scoped", LocationIDs: []int64{locationID, locationID}}); err != nil {
		t.Fatal(err)
	}
	key, err = store.Authenticate(ctx, secret)
	if err != nil {
		t.Fatal(err)
	}
	if key.AllLocations || !key.Allows(locationID) || key.Allows(locationID+1) {
		t.Fatalf("scope=%+v", key)
	}
	if len(key.Locations) != 1 || key.Locations[0].ID != locationID || key.Locations[0].Name != "New location" {
		t.Fatalf("location summaries=%+v", key.Locations)
	}
	listed, _, err := store.List(ctx, listing.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || len(listed[0].Locations) != 1 || listed[0].Locations[0] != key.Locations[0] {
		t.Fatalf("listed location summaries=%+v", listed)
	}
	if !key.ReadUsers || !key.ReadLocations || !key.ReadBranding {
		t.Fatal("coarse credentials lack read access")
	}
	if err := store.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, secret); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("revoked auth=%v", err)
	}
	items, count, err := store.List(ctx, listing.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(items) != 0 {
		t.Fatalf("deleted key still listed: count=%d", count)
	}
}
