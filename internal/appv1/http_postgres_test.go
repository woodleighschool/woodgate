//go:build postgres

package appv1

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"
	"github.com/woodleighschool/woodgate/internal/appkey"
	"github.com/woodleighschool/woodgate/internal/checkin"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

type protocolFixture struct {
	db            *pgxpool.Pool
	router        http.Handler
	keys          *appkey.Store
	objects       *bloby.Service
	checkins      *checkin.Store
	key           *appkey.CreatedKey
	location      *checkin.Location
	locationAppID uuid.UUID
	userAppID     uuid.UUID
}

func newProtocolFixture(t *testing.T) *protocolFixture {
	t.Helper()
	db, ctx := testdb.Open(t)
	logger := slog.New(slog.DiscardHandler)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	store := checkin.NewStore(db, objects)
	location, err := store.CreateLocation(ctx, checkin.LocationMutation{Name: "Test location", Enabled: true, Notes: true})
	if err != nil {
		t.Fatal(err)
	}
	fixture := &protocolFixture{db: db, objects: objects, checkins: store, location: location, keys: appkey.NewStore(db)}
	if err := db.QueryRow(ctx, "SELECT app_id FROM locations WHERE id=$1", location.ID).Scan(&fixture.locationAppID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(ctx, "INSERT INTO users(email,name) VALUES('person@example.invalid','Test Person') RETURNING app_id").Scan(&fixture.userAppID); err != nil {
		t.Fatal(err)
	}
	var groupID int64
	if err := db.QueryRow(ctx, "INSERT INTO directory_groups(source,external_id,display_name) VALUES('entra','initial-group','Initial test group') RETURNING id").Scan(&groupID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO directory_group_memberships(user_id,group_id) SELECT id,$1 FROM users WHERE app_id=$2", groupID, fixture.userAppID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "INSERT INTO location_directory_groups(location_id,group_id) VALUES($1,$2)", location.ID, groupID); err != nil {
		t.Fatal(err)
	}

	fixture.key, err = fixture.keys.Create(ctx, appkey.Mutation{Name: "Test app", LocationIDs: []int64{location.ID}})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	RegisterRoutes(router, router, Dependencies{Store: NewStore(db), Keys: fixture.keys, Checkins: checkin.NewService(store, objects), Objects: objects, Logger: logger})
	fixture.router = router
	return fixture
}

func (fixture *protocolFixture) request(t *testing.T, request *http.Request, status int) *httptest.ResponseRecorder {
	t.Helper()
	request.Header.Set("X-Api-Key", fixture.key.APIKey)
	recorder := httptest.NewRecorder()
	fixture.router.ServeHTTP(recorder, request)
	if recorder.Code != status {
		t.Fatalf("%s %s status=%d want=%d body=%s", request.Method, request.URL.Path, recorder.Code, status, recorder.Body.String())
	}
	return recorder
}

func TestCompanionUUIDRoutes(t *testing.T) {
	fixture := newProtocolFixture(t)
	t.Run("principal and scoped pairing", func(t *testing.T) { assertProtocolPrincipal(t, fixture) })
	t.Run("locations and people", func(t *testing.T) { assertProtocolLists(t, fixture) })
	t.Run("scope enforced", func(t *testing.T) { assertProtocolScope(t, fixture) })
	t.Run("empty groups mean empty roster", func(t *testing.T) { assertEmptyProtocolRoster(t, fixture) })

	t.Run("expired and revoked keys", func(t *testing.T) { assertKeyRevocation(t, fixture) })
}

func assertProtocolPrincipal(t *testing.T, fixture *protocolFixture) {
	t.Helper()
	recorder := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil), 200)
	var response meResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Principal.ID != fixture.key.AppID || response.Principal.Type != "api_key" {
		t.Fatalf("principal=%+v", response.Principal)
	}
	var paired bool
	for _, grant := range response.Access {
		if grant.Resource == "checkins" && grant.Action == "create" && grant.LocationID != nil && *grant.LocationID == fixture.locationAppID {
			paired = true
		}
	}
	if !paired {
		t.Fatalf("grants=%+v", response.Access)
	}
}

func assertProtocolLists(t *testing.T, fixture *protocolFixture) {
	t.Helper()
	recorder := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/locations", nil), 200)
	var locations listResponse[location]
	if err := json.Unmarshal(recorder.Body.Bytes(), &locations); err != nil {
		t.Fatal(err)
	}
	if locations.Total != 1 || len(locations.Rows) != 1 || locations.Rows[0].ID != fixture.locationAppID || locations.Rows[0].GroupIDs == nil {
		t.Fatalf("locations=%+v", locations)
	}
	recorder = fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users?location_id="+fixture.locationAppID.String()+"&limit=250&offset=0", nil), 200)
	var people listResponse[person]
	if err := json.Unmarshal(recorder.Body.Bytes(), &people); err != nil {
		t.Fatal(err)
	}
	if people.Total != 1 || len(people.Rows) != 1 || people.Rows[0].ID != fixture.userAppID {
		t.Fatalf("people=%+v", people)
	}
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/locations/"+fixture.locationAppID.String(), nil), 200)
}

func assertProtocolScope(t *testing.T, fixture *protocolFixture) {
	t.Helper()
	other, err := fixture.checkins.CreateLocation(t.Context(), checkin.LocationMutation{Name: "Other location", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	if err := fixture.db.QueryRow(t.Context(), "SELECT app_id FROM locations WHERE id=$1", other.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/locations/"+id.String(), nil), 403)
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users?location_id="+id.String(), nil), 403)
}

func assertEmptyProtocolRoster(t *testing.T, fixture *protocolFixture) {
	t.Helper()
	if _, err := fixture.db.Exec(t.Context(), "DELETE FROM location_directory_groups WHERE location_id=$1", fixture.location.ID); err != nil {
		t.Fatal(err)
	}
	recorder := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users?location_id="+fixture.locationAppID.String(), nil), 200)
	var people listResponse[person]
	if err := json.Unmarshal(recorder.Body.Bytes(), &people); err != nil {
		t.Fatal(err)
	}
	if people.Total != 0 || people.Rows == nil || len(people.Rows) != 0 {
		t.Fatalf("people=%+v", people)
	}
}

func assertKeyRevocation(t *testing.T, fixture *protocolFixture) {
	t.Helper()
	expired := time.Now().Add(-time.Minute)
	if _, err := fixture.keys.Update(t.Context(), fixture.key.ID, appkey.Mutation{Name: "Expired", LocationIDs: []int64{fixture.location.ID}, ExpiresAt: &expired}); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil), 401)
	if response.Body.String() != "{\"error\":\"unauthorized\"}\n" {
		t.Fatalf("auth error=%s", response.Body.String())
	}
	if _, err := fixture.keys.Update(t.Context(), fixture.key.ID, appkey.Mutation{Name: "Revoked", LocationIDs: []int64{fixture.location.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.keys.Delete(t.Context(), fixture.key.ID); err != nil {
		t.Fatal(err)
	}
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil), 401)
}

func TestCompanionPhotoAndAttribution(t *testing.T) {
	fixture := newProtocolFixture(t)
	fields := [][2]string{{"user_id", fixture.userAppID.String()}, {"location_id", fixture.locationAppID.String()}, {"direction", "check_in"}, {"notes", "  test notes  "}}
	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	// Native photos larger than Huma's default JSON body limit still use multipart.
	photo := append(imageBytes.Bytes(), bytes.Repeat([]byte{0}, (1<<20)+1)...)
	response := fixture.request(t, multipartRequest(t, fields, photo), 201)
	var created checkinResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == uuid.Nil || created.UserID != fixture.userAppID || created.LocationID != fixture.locationAppID || created.CreatedByID != fixture.key.AppID || created.CreatedByKind != "api_key" || created.Notes != "test notes" {
		t.Fatalf("created=%+v", created)
	}
	var checkinID, photoObjectID int64
	if err := fixture.db.QueryRow(t.Context(), "SELECT id, photo_object_id FROM checkins WHERE app_id=$1", created.ID).Scan(&checkinID, &photoObjectID); err != nil {
		t.Fatal(err)
	}
	record, err := fixture.checkins.GetCheckin(t.Context(), checkinID)
	if err != nil || record.ActorName != fixture.key.Name {
		t.Fatalf("check-in actor name was not resolved: record=%+v, error=%v", record, err)
	}
	object, err := fixture.objects.GetByID(t.Context(), photoObjectID)
	if err != nil {
		t.Fatal(err)
	}
	if !object.Available() || object.Prefix != checkin.PhotoObjectPrefix {
		t.Fatalf("object=%+v", object)
	}
	// A cached native user remains valid after their group membership changes.
	var groupID int64
	if err := fixture.db.QueryRow(t.Context(), "INSERT INTO directory_groups(source,external_id,display_name) VALUES('entra','synthetic-group','Test group') RETURNING id").Scan(&groupID); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.checkins.UpdateLocation(t.Context(), fixture.location.ID, checkin.LocationMutation{Name: "Test location", Enabled: true, Notes: true, GroupIDs: []int64{groupID}}); err != nil {
		t.Fatal(err)
	}
	fixture.request(t, multipartRequest(t, fields, nil), 201)
	fixture.request(t, multipartRequest(t, fields, []byte("not an image")), 422)
	missingUserFields := append([][2]string(nil), fields...)
	missingUserFields[0][1] = "10000000-0000-0000-0000-000000000099"
	invalidPhoto := fixture.request(t, multipartRequest(t, missingUserFields, []byte("not an image")), 422)
	var invalid problem
	if err := json.Unmarshal(invalidPhoto.Body.Bytes(), &invalid); err != nil {
		t.Fatal(err)
	}
	if invalid.Detail != "Asset is invalid." {
		t.Fatalf("photo validation ordering=%+v", invalid)
	}

	if err := fixture.keys.Delete(t.Context(), fixture.key.ID); err != nil {
		t.Fatal(err)
	}
	var actorID uuid.UUID
	if err := fixture.db.QueryRow(t.Context(), "SELECT actor_app_id FROM checkins WHERE app_id=$1", created.ID).Scan(&actorID); err != nil {
		t.Fatal(err)
	}
	if actorID != fixture.key.AppID {
		t.Fatal("revocation changed actor attribution")
	}
}

func TestCompanionBrandingHandleLifecycle(t *testing.T) {
	fixture := newProtocolFixture(t)
	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	object, err := fixture.objects.Write(t.Context(), checkin.BackgroundObjectPrefix, "background.jpg", "image/jpeg", imageBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.checkins.SetLocationBackground(t.Context(), fixture.location.ID, object.ID); err != nil {
		t.Fatal(err)
	}
	var handle uuid.UUID
	if err := fixture.db.QueryRow(t.Context(), "SELECT background_app_id FROM locations WHERE id=$1", fixture.location.ID).Scan(&handle); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/assets/"+handle.String()+"/content", nil), 200)
	if response.Header().Get("Content-Type") != "image/jpeg" || !bytes.Equal(response.Body.Bytes(), imageBytes.Bytes()) {
		t.Fatal("branding bytes changed")
	}
	replacement, err := fixture.objects.Write(t.Context(), checkin.BackgroundObjectPrefix, "replacement.jpg", "image/jpeg", imageBytes.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.checkins.SetLocationBackground(t.Context(), fixture.location.ID, replacement.ID); err != nil {
		t.Fatal(err)
	}
	var nextHandle uuid.UUID
	if err := fixture.db.QueryRow(t.Context(), "SELECT background_app_id FROM locations WHERE id=$1", fixture.location.ID).Scan(&nextHandle); err != nil {
		t.Fatal(err)
	}
	if handle == nextHandle {
		t.Fatal("replacement retained cached native handle")
	}
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/assets/"+handle.String()+"/content", nil), 404)
	fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/assets/"+nextHandle.String()+"/content", nil), 200)
	if err := fixture.checkins.SetLocationBackground(t.Context(), fixture.location.ID, replacement.ID); err != nil {
		t.Fatal(err)
	}
	var unchanged uuid.UUID
	if err := fixture.db.QueryRow(t.Context(), "SELECT background_app_id FROM locations WHERE id=$1", fixture.location.ID).Scan(&unchanged); err != nil {
		t.Fatal(err)
	}
	if unchanged != nextHandle {
		t.Fatal("same object rotated native handle")
	}
}

func TestIndependentLegacyReadPermissions(t *testing.T) {
	fixture := newProtocolFixture(t)
	if _, err := fixture.db.Exec(t.Context(), "UPDATE app_keys SET read_users=false,read_locations=false,read_branding=false WHERE id=$1", fixture.key.ID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/locations", "/api/v1/locations/" + fixture.locationAppID.String(), "/api/v1/users?location_id=" + fixture.locationAppID.String(), "/api/v1/assets/40000000-0000-0000-0000-000000000001/content"} {
		response := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil), http.StatusForbidden)
		if response.Body.String() != "{\"error\":\"forbidden\"}\n" {
			t.Fatalf("read error=%s", response.Body.String())
		}
	}
	response := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil), http.StatusOK)
	var me meResponse
	if err := json.Unmarshal(response.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	for _, grant := range me.Access {
		if grant.Action == "read" {
			t.Fatalf("undeclared read grant=%+v", grant)
		}
	}
	if me.Capabilities["users"].Read || me.Capabilities["locations"].Read || me.Capabilities["assets"].Read || !me.Capabilities["checkins"].Create {
		t.Fatalf("capabilities=%+v", me.Capabilities)
	}
	fields := [][2]string{{"user_id", fixture.userAppID.String()}, {"location_id", fixture.locationAppID.String()}, {"direction", "check_in"}}
	fixture.request(t, multipartRequest(t, fields, nil), http.StatusCreated)
	if _, err := fixture.keys.Update(t.Context(), fixture.key.ID, appkey.Mutation{Name: "No create scope"}); err != nil {
		t.Fatal(err)
	}
	recorder := fixture.request(t, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkins", strings.NewReader("not multipart")), http.StatusForbidden)
	if recorder.Body.String() != "{\"error\":\"forbidden\"}\n" {
		t.Fatalf("missing create scope error=%s", recorder.Body.String())
	}
}
