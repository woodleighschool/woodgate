//go:build postgres

package checkin

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestSharedAttachmentLifetime(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, objects)
	object, err := objects.Write(ctx, BackgroundObjectPrefix, "background.jpg", "image/jpeg", []byte("synthetic owned object"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.CreateLocation(ctx, LocationMutation{Name: "First", Enabled: true, BackgroundObjectID: &object.ID})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateLocation(ctx, LocationMutation{Name: "Second", Enabled: true, BackgroundObjectID: &object.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetLocationLogo(ctx, first.ID, object.ID); !errors.Is(err, fault.ErrInvalidInput) {
		t.Fatalf("cross-gallery attachment=%v", err)
	}
	var missing *uuid.UUID
	if err := db.QueryRow(ctx, "SELECT logo_app_id FROM locations WHERE id=$1", first.ID).Scan(&missing); err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatal("failed attachment left a companion handle")
	}
	if err := store.DeleteLocation(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.GetByID(ctx, object.ID); err != nil {
		t.Fatalf("shared object deleted early: %v", err)
	}
	if _, err := store.UpdateLocation(ctx, second.ID, LocationMutation{Name: "Second", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.GetByID(ctx, object.ID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("unreferenced object=%v", err)
	}
	if err := db.QueryRow(ctx, "SELECT background_app_id FROM locations WHERE id=$1", second.ID).Scan(&missing); err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatal("cleared owner retains companion handle")
	}
}

func TestFailedCheckinCleansPhoto(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, objects)
	service := NewService(store, objects)
	location, err := store.CreateLocation(ctx, LocationMutation{Name: "Test location", Enabled: true, Photo: true})
	if err != nil {
		t.Fatal(err)
	}
	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	_, err = service.Submit(ctx, CheckinCreate{UserID: 99999, LocationID: location.ID, Direction: DirectionIn}, Actor{Kind: "api_key", AppID: uuid.MustParse("30000000-0000-0000-0000-000000000001")}, imageBytes.Bytes())
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("missing person error=%v", err)
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM storage_objects").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed check-in left %d objects", count)
	}
}

func TestLocationSaveFinalizesUploads(t *testing.T) {
	db, ctx := testdb.Open(t)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, objects)
	service := NewService(store, objects)
	background := uploadPendingImage(t, objects, BackgroundObjectPrefix)
	location, err := service.CreateLocation(ctx, LocationMutation{Name: "Uploaded location", Enabled: true, BackgroundObjectID: &background.ID})
	if err != nil {
		t.Fatal(err)
	}
	if location.BackgroundFile == nil || location.BackgroundFile.ContentType != "image/jpeg" {
		t.Fatalf("unfinalized background=%+v", location.BackgroundFile)
	}
	replacement := uploadPendingImage(t, objects, BackgroundObjectPrefix)
	_, err = service.UpdateLocation(ctx, location.ID, LocationMutation{Name: "Invalid group", Enabled: true, BackgroundObjectID: &replacement.ID, GroupIDs: []int64{999999}})
	if !errors.Is(err, fault.ErrNotFound) {
		t.Fatalf("invalid group=%v", err)
	}
	current, err := store.GetLocation(ctx, location.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Name != location.Name || current.BackgroundObjectID == nil || *current.BackgroundObjectID != background.ID {
		t.Fatalf("failed update changed owner=%+v", current)
	}
	if _, err := objects.GetByID(ctx, replacement.ID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("failed update retained upload=%v", err)
	}
	fresh := uploadPendingImage(t, objects, BackgroundObjectPrefix)
	incomplete, _, err := objects.BeginDirect(ctx, LogoObjectPrefix, "unfinished.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateLocation(ctx, LocationMutation{Name: "Failed location", Enabled: true, BackgroundObjectID: &fresh.ID, LogoObjectID: &incomplete.ID}); err == nil {
		t.Fatal("incomplete upload created location")
	}
	var count int
	if err := db.QueryRow(ctx, "SELECT count(*) FROM locations").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed creation persisted owner: %d locations", count)
	}
	if _, err := objects.GetByID(ctx, fresh.ID); !errors.Is(err, bloby.ErrNotFound) {
		t.Fatalf("failed finalization retained published upload=%v", err)
	}
}

func uploadPendingImage(t *testing.T, objects *bloby.Service, prefix string) *bloby.Object {
	t.Helper()
	object, action, err := objects.BeginDirect(t.Context(), prefix, "uploaded.jpg")
	if err != nil {
		t.Fatal(err)
	}
	var content bytes.Buffer
	if err := jpeg.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	target := action.Target
	request := httptest.NewRequestWithContext(t.Context(), target.Method, target.URL, &content)
	for key, value := range target.Headers {
		request.Header.Set(key, value)
	}
	writer := httptest.NewRecorder()
	objects.TransferHandler().ServeHTTP(writer, request)
	if writer.Code != http.StatusNoContent {
		t.Fatalf("upload status=%d body=%s", writer.Code, writer.Body.String())
	}
	current, err := objects.GetByID(t.Context(), object.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Available() {
		t.Fatal("direct upload bypassed finalization")
	}
	return object
}
