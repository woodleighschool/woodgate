//go:build postgres

package checkin

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestLocationOwnsAndLifecyclesAttachments(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{
		Kind: bloby.KindFile, TransferTTL: time.Minute,
		File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)},
	}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, objects)

	background := createAvailableImage(t, ctx, objects, BackgroundObjectPrefix, "first.png")
	location, err := store.CreateLocation(ctx, LocationMutation{
		Name: "Reception", Enabled: true, BackgroundObjectID: &background.ID,
	})
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	if location.BackgroundFile == nil || location.BackgroundFile.Filename != "first.png" || location.BackgroundURL == "" {
		t.Fatalf("background projection = %+v", location)
	}

	logo := createAvailableImage(t, ctx, objects, LogoObjectPrefix, "logo.png")
	if _, err := store.UpdateLocation(ctx, location.ID, LocationMutation{
		Name: "Reception", Enabled: true, BackgroundObjectID: &logo.ID,
	}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("cross-gallery update error = %v, want invalid input", err)
	}

	replacement := createAvailableImage(t, ctx, objects, BackgroundObjectPrefix, "second.webp")
	location, err = store.UpdateLocation(ctx, location.ID, LocationMutation{
		Name: "Reception", Enabled: true, BackgroundObjectID: &replacement.ID,
	})
	if err != nil {
		t.Fatalf("UpdateLocation: %v", err)
	}
	if location.BackgroundObjectID == nil || *location.BackgroundObjectID != replacement.ID {
		t.Fatalf("background object = %v, want %d", location.BackgroundObjectID, replacement.ID)
	}
	if _, err := objects.GetByID(ctx, background.ID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("replaced object error = %v, want not found", err)
	}

	if err := store.DeleteLocation(ctx, location.ID); err != nil {
		t.Fatalf("DeleteLocation: %v", err)
	}
	if _, err := objects.GetByID(ctx, replacement.ID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("deleted location object error = %v, want not found", err)
	}
}

func createAvailableImage(
	t *testing.T,
	ctx context.Context,
	objects *bloby.Service,
	prefix, filename string,
) *bloby.Object {
	t.Helper()
	contentType := "image/png"
	if strings.HasSuffix(filename, ".webp") {
		contentType = "image/webp"
	}
	object, err := objects.Write(ctx, prefix, filename, contentType, []byte("synthetic image fixture"))
	if err != nil {
		t.Fatalf("write image %s: %v", filename, err)
	}
	return object
}

func TestLocationGroupFiltersStationPeopleAndCheckins(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db, nil)
	service := NewService(store, nil)

	var eligibleID, outsiderID, groupID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,source,external_id,department)
VALUES ('eligible@example.invalid','Eligible Person','entra','user-eligible','Students') RETURNING id`).Scan(&eligibleID); err != nil {
		t.Fatalf("insert eligible user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,source,external_id,department)
VALUES ('outsider@example.invalid','Outside Person','entra','user-outsider','Staff') RETURNING id`).Scan(&outsiderID); err != nil {
		t.Fatalf("insert outsider user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO directory_groups (source,external_id,display_name)
VALUES ('entra','group-checkin','Check-in Group') RETURNING id`).Scan(&groupID); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO directory_group_memberships (user_id,group_id) VALUES ($1,$2)`, eligibleID, groupID); err != nil {
		t.Fatalf("insert group membership: %v", err)
	}

	location, err := service.CreateLocation(ctx, LocationMutation{
		Name: "Reception", Enabled: true, Notes: true, GroupIDs: []int64{groupID},
	})
	if err != nil {
		t.Fatalf("CreateLocation: %v", err)
	}
	assertLocationRelationProjections(t, ctx, service, location, groupID)
	people, err := service.ListStationPeople(ctx, location.ID)
	if err != nil {
		t.Fatalf("ListStationPeople: %v", err)
	}
	if len(people) != 1 || people[0].ID != eligibleID {
		t.Fatalf("people = %+v, want only eligible user %d", people, eligibleID)
	}

	created, err := service.CreateCheckin(ctx, CheckinCreate{
		UserID: eligibleID, LocationID: location.ID, Direction: DirectionIn, Notes: "Front desk",
	}, eligibleID)
	if err != nil {
		t.Fatalf("CreateCheckin eligible: %v", err)
	}
	if created.Person.Name != "Eligible Person" || created.Person.Email != "eligible@example.invalid" ||
		created.Person.Department != "Students" || created.Location.Name != "Reception" {
		t.Fatalf("created check-in = %+v", created)
	}
	if _, err := service.CreateCheckin(ctx, CheckinCreate{
		UserID: outsiderID, LocationID: location.ID, Direction: DirectionIn,
	}, eligibleID); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("CreateCheckin outsider error = %v, want invalid input", err)
	}

	items, count, err := service.ListCheckins(ctx, CheckinListParams{LocationID: location.ID})
	if err != nil {
		t.Fatalf("ListCheckins: %v", err)
	}
	if count != 1 || len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("check-ins = %+v count=%d, want created check-in", items, count)
	}
}

func assertLocationRelationProjections(
	t *testing.T,
	ctx context.Context,
	service *Service,
	location *Location,
	groupID int64,
) {
	t.Helper()
	if len(location.Groups) != 1 || location.Groups[0].ID != groupID ||
		location.Groups[0].DisplayName != "Check-in Group" ||
		location.Groups[0].Source != directory.SourceEntra {
		t.Fatalf("location groups = %+v", location.Groups)
	}
	groupChoices, count, err := service.ListLocationGroupChoices(ctx, listing.Params{
		Q: "check-in", PageSize: 20,
	})
	if err != nil || count != 1 || len(groupChoices) != 1 || groupChoices[0].ID != groupID {
		t.Fatalf("location group choices = %+v count %d error %v", groupChoices, count, err)
	}
	stationLocations, count, err := service.ListStationLocations(ctx, listing.Params{
		Q: "reception", PageSize: 20,
	})
	if err != nil || count != 1 || len(stationLocations) != 1 || stationLocations[0].ID != location.ID {
		t.Fatalf("station location choices = %+v count %d error %v", stationLocations, count, err)
	}
}

func TestListCheckinsUsesHalfOpenCreatedRange(t *testing.T) {
	db, ctx := testdb.Open(t)
	service := NewService(NewStore(db, nil), nil)

	var userID, locationID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,source,external_id)
VALUES ('range@example.invalid','Range Person','entra','range-user') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO locations (name,enabled) VALUES ('Range Location',true) RETURNING id`).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}

	from := time.Date(2026, time.August, 30, 0, 0, 0, 0, time.UTC)
	before := from.Add(24 * time.Hour)
	var fromID, middleID int64
	for _, event := range []struct {
		createdAt time.Time
		id        *int64
	}{
		{createdAt: from, id: &fromID},
		{createdAt: from.Add(12 * time.Hour), id: &middleID},
		{createdAt: before},
	} {
		var id int64
		if err := db.QueryRow(ctx, `INSERT INTO checkins (user_id,location_id,direction,created_at)
VALUES ($1,$2,'check_in',$3) RETURNING id`, userID, locationID, event.createdAt).Scan(&id); err != nil {
			t.Fatalf("insert check-in at %s: %v", event.createdAt, err)
		}
		if event.id != nil {
			*event.id = id
		}
	}

	items, count, err := service.ListCheckins(ctx, CheckinListParams{CreatedFrom: &from, CreatedBefore: &before})
	if err != nil {
		t.Fatalf("ListCheckins: %v", err)
	}
	if count != 2 || len(items) != 2 || items[0].ID != middleID || items[1].ID != fromID {
		t.Fatalf("check-ins = %+v, count = %d; want IDs %d and %d", items, count, middleID, fromID)
	}

	if _, _, err := service.ListCheckins(ctx, CheckinListParams{CreatedFrom: &before, CreatedBefore: &before}); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("equal created bounds error = %v, want invalid input", err)
	}
}

func TestCheckinProjectsRawRelationSummaries(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db, nil)

	var userID, creatorID, locationID, stationID, checkinID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,source)
VALUES ('unnamed@example.invalid','','local') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,source)
VALUES ('creator@example.invalid','Creator','local') RETURNING id`).Scan(&creatorID); err != nil {
		t.Fatalf("insert creator: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO locations (name,enabled)
VALUES ('Reception',true) RETURNING id`).Scan(&locationID); err != nil {
		t.Fatalf("insert location: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO stations (name,location_id,secret_hash)
VALUES ('Reception iPad',$1,$2) RETURNING id`, locationID, strings.Repeat("a", 64)).Scan(&stationID); err != nil {
		t.Fatalf("insert station: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO checkins (user_id,location_id,direction,station_id,created_by_user_id)
VALUES ($1,$2,'check_in',$3,$4) RETURNING id`, userID, locationID, stationID, creatorID).Scan(&checkinID); err != nil {
		t.Fatalf("insert check-in: %v", err)
	}

	checkin, err := store.GetCheckin(ctx, checkinID)
	if err != nil {
		t.Fatalf("GetCheckin: %v", err)
	}
	if checkin.Person.ID != userID || checkin.Person.Name != "" || checkin.Person.Email != "unnamed@example.invalid" {
		t.Fatalf("person projection = %+v", checkin.Person)
	}
	if checkin.Location.ID != locationID || checkin.Location.Name != "Reception" {
		t.Fatalf("location projection = %+v", checkin.Location)
	}
	if checkin.Station == nil || checkin.Station.ID != stationID || checkin.Station.Name != "Reception iPad" {
		t.Fatalf("station projection = %+v", checkin.Station)
	}
	if checkin.CreatedBy == nil || checkin.CreatedBy.ID != creatorID ||
		checkin.CreatedBy.Name != "Creator" || checkin.CreatedBy.Email != "creator@example.invalid" {
		t.Fatalf("creator projection = %+v", checkin.CreatedBy)
	}
}
