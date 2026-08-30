//go:build postgres

package checkin

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/storage"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

type attachmentBackendFake struct{}

func (attachmentBackendFake) Delete(context.Context, string) error { return nil }

func TestLocationOwnsAndLifecyclesAttachments(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects := storage.NewObjectStore(db, attachmentBackendFake{}, slog.New(slog.DiscardHandler))
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
	if _, err := objects.GetByID(ctx, background.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("replaced object error = %v, want not found", err)
	}

	if err := store.DeleteLocation(ctx, location.ID); err != nil {
		t.Fatalf("DeleteLocation: %v", err)
	}
	if _, err := objects.GetByID(ctx, replacement.ID); !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("deleted location object error = %v, want not found", err)
	}
}

func createAvailableImage(
	t *testing.T,
	ctx context.Context,
	objects *storage.ObjectStore,
	prefix, filename string,
) *storage.Object {
	t.Helper()
	object, err := objects.CreatePending(ctx, prefix, filename)
	if err != nil {
		t.Fatalf("CreatePending(%s): %v", filename, err)
	}
	contentType := "image/png"
	if strings.HasSuffix(filename, ".webp") {
		contentType = "image/webp"
	}
	object, err = objects.MarkAvailable(ctx, object.ID, 42, contentType, strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("MarkAvailable(%s): %v", filename, err)
	}
	return object
}

func TestLocationGroupFiltersStationPeopleAndCheckins(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db, nil)
	service := NewService(store, nil, nil, nil)

	var eligibleID, outsiderID, groupID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,access_enabled,source,external_id,department)
VALUES ('eligible@example.invalid','Eligible Person',true,'entra','user-eligible','Students') RETURNING id`).Scan(&eligibleID); err != nil {
		t.Fatalf("insert eligible user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,access_enabled,source,external_id,department)
VALUES ('outsider@example.invalid','Outside Person',true,'entra','user-outsider','Staff') RETURNING id`).Scan(&outsiderID); err != nil {
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
	if created.UserName != "Eligible Person" || created.LocationName != "Reception" || created.Department != "Students" {
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

func TestListCheckinsUsesHalfOpenCreatedRange(t *testing.T) {
	db, ctx := testdb.Open(t)
	service := NewService(NewStore(db, nil), nil, nil, nil)

	var userID, locationID int64
	if err := db.QueryRow(ctx, `INSERT INTO users (email,name,access_enabled,source,external_id)
VALUES ('range@example.invalid','Range Person',true,'entra','range-user') RETURNING id`).Scan(&userID); err != nil {
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
