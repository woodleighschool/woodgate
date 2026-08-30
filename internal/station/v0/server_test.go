package v0

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/station"
)

type legacyMappingKey struct {
	kind string
	id   uuid.UUID
}

type objectMappingKey struct {
	kind string
	id   int64
}

type repositoryFake struct {
	secret    string
	station   station.Station
	apiKeyID  uuid.UUID
	objectIDs map[legacyMappingKey]int64
	legacyIDs map[objectMappingKey]uuid.UUID
}

func (f *repositoryFake) authenticate(
	_ context.Context,
	secret string,
) (*station.Station, uuid.UUID, error) {
	if secret != f.secret {
		return nil, uuid.Nil, station.ErrUnauthorized
	}
	item := f.station
	return &item, f.apiKeyID, nil
}

func (f *repositoryFake) legacyID(
	_ context.Context,
	kind string,
	objectID int64,
) (uuid.UUID, error) {
	id, ok := f.legacyIDs[objectMappingKey{kind: kind, id: objectID}]
	if !ok {
		return uuid.Nil, pgx.ErrNoRows
	}
	return id, nil
}

func (f *repositoryFake) objectID(
	_ context.Context,
	kind string,
	legacyID uuid.UUID,
) (int64, error) {
	id, ok := f.objectIDs[legacyMappingKey{kind: kind, id: legacyID}]
	if !ok {
		return 0, pgx.ErrNoRows
	}
	return id, nil
}

func (f *repositoryFake) ensureLegacyID(
	ctx context.Context,
	kind string,
	objectID int64,
) (uuid.UUID, error) {
	return f.legacyID(ctx, kind, objectID)
}

type peopleProviderFake struct {
	locations []int64
	people    []station.Person
}

func (f *peopleProviderFake) ListStationPeople(
	_ context.Context,
	locationID int64,
) ([]station.Person, error) {
	f.locations = append(f.locations, locationID)
	return f.people, nil
}

type checkinSubmitterFake struct {
	submission station.CheckinSubmission
	receipt    station.CheckinReceipt
}

func (f *checkinSubmitterFake) SubmitStationCheckin(
	_ context.Context,
	submission station.CheckinSubmission,
) (*station.CheckinReceipt, error) {
	f.submission = submission
	receipt := f.receipt
	return &receipt, nil
}

func TestXAPIKeyAuthentication(t *testing.T) {
	locationID := uuid.MustParse("10000000-0000-4000-8000-000000000011")
	apiKeyID := uuid.MustParse("10000000-0000-4000-8000-000000000007")
	repository := &repositoryFake{
		secret:   "legacy-secret",
		station:  station.Station{ID: 7, Name: "Reception", LocationID: 11, Enabled: true},
		apiKeyID: apiKeyID,
		legacyIDs: map[objectMappingKey]uuid.UUID{
			{kind: "location", id: 11}: locationID,
		},
	}
	router := chi.NewRouter()
	newServer(repository, Dependencies{}, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	for _, test := range []struct {
		name       string
		secret     string
		wantStatus int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "wrong", secret: "wrong-secret", wantStatus: http.StatusUnauthorized},
		{name: "accepted", secret: "legacy-secret", wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil)
			if test.secret != "" {
				request.Header.Set("X-Api-Key", test.secret)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", response.Code, test.wantStatus, response.Body.String())
			}
			if test.wantStatus != http.StatusOK {
				return
			}
			var body struct {
				Principal legacyPrincipal `json:"principal"`
				Access    []legacyGrant   `json:"access"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Principal.ID != apiKeyID || len(body.Access) != 1 || body.Access[0].LocationID != locationID {
				t.Fatalf("response = %#v", body)
			}
		})
	}
}

func TestListPeopleRejectsAnotherLocation(t *testing.T) {
	boundLocationID := uuid.MustParse("20000000-0000-4000-8000-000000000011")
	foreignLocationID := uuid.MustParse("20000000-0000-4000-8000-000000000012")
	legacyPersonID := uuid.MustParse("20000000-0000-4000-8000-000000000023")
	repository := &repositoryFake{
		secret:   "legacy-secret",
		station:  station.Station{ID: 7, LocationID: 11, Enabled: true},
		apiKeyID: uuid.MustParse("20000000-0000-4000-8000-000000000007"),
		objectIDs: map[legacyMappingKey]int64{
			{kind: "location", id: boundLocationID}:   11,
			{kind: "location", id: foreignLocationID}: 12,
		},
		legacyIDs: map[objectMappingKey]uuid.UUID{
			{kind: "user", id: 23}: legacyPersonID,
		},
	}
	people := &peopleProviderFake{people: []station.Person{{ID: 23, Name: "Test Person", Email: "person@example.invalid"}}}
	router := chi.NewRouter()
	newServer(repository, Dependencies{People: people}, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	foreignRequest := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/users?location_id="+foreignLocationID.String(),
		nil,
	)
	foreignRequest.Header.Set("X-Api-Key", repository.secret)
	foreignResponse := httptest.NewRecorder()
	router.ServeHTTP(foreignResponse, foreignRequest)
	if foreignResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign location status = %d, want %d", foreignResponse.Code, http.StatusForbidden)
	}
	if len(people.locations) != 0 {
		t.Fatalf("people provider called for foreign location: %#v", people.locations)
	}

	boundRequest := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/v1/users?location_id="+boundLocationID.String(),
		nil,
	)
	boundRequest.Header.Set("X-Api-Key", repository.secret)
	boundResponse := httptest.NewRecorder()
	router.ServeHTTP(boundResponse, boundRequest)
	if boundResponse.Code != http.StatusOK {
		t.Fatalf("bound location status = %d, want %d; body = %q", boundResponse.Code, http.StatusOK, boundResponse.Body.String())
	}
	if len(people.locations) != 1 || people.locations[0] != 11 {
		t.Fatalf("people provider locations = %#v", people.locations)
	}
	var body legacyList[legacyPerson]
	if err := json.NewDecoder(boundResponse.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Total != 1 || len(body.Rows) != 1 || body.Rows[0].ID != legacyPersonID {
		t.Fatalf("response = %#v", body)
	}
}

func TestCheckinTranslatesLegacyUUIDs(t *testing.T) {
	legacyUserID := uuid.MustParse("30000000-0000-4000-8000-000000000023")
	legacyLocationID := uuid.MustParse("30000000-0000-4000-8000-000000000011")
	legacyCheckinID := uuid.MustParse("30000000-0000-4000-8000-000000000031")
	repository := &repositoryFake{
		secret:   "legacy-secret",
		station:  station.Station{ID: 7, LocationID: 11, Enabled: true},
		apiKeyID: uuid.MustParse("30000000-0000-4000-8000-000000000007"),
		objectIDs: map[legacyMappingKey]int64{
			{kind: "user", id: legacyUserID}:         23,
			{kind: "location", id: legacyLocationID}: 11,
		},
		legacyIDs: map[objectMappingKey]uuid.UUID{
			{kind: "checkin", id: 31}: legacyCheckinID,
		},
	}
	checkins := &checkinSubmitterFake{receipt: station.CheckinReceipt{
		ID: 31, PersonID: 23, LocationID: 11, Direction: station.DirectionIn,
	}}
	router := chi.NewRouter()
	newServer(repository, Dependencies{Checkins: checkins}, slog.New(slog.DiscardHandler)).RegisterRoutes(router)

	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	if err := form.WriteField("user_id", legacyUserID.String()); err != nil {
		t.Fatalf("write user_id: %v", err)
	}
	if err := form.WriteField("location_id", legacyLocationID.String()); err != nil {
		t.Fatalf("write location_id: %v", err)
	}
	if err := form.WriteField("direction", string(station.DirectionIn)); err != nil {
		t.Fatalf("write direction: %v", err)
	}
	if err := form.Close(); err != nil {
		t.Fatalf("close form: %v", err)
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/checkins", body)
	request.Header.Set("X-Api-Key", repository.secret)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %q", response.Code, http.StatusCreated, response.Body.String())
	}
	if checkins.submission.StationID != 7 || checkins.submission.LocationID != 11 ||
		checkins.submission.PersonID != 23 || checkins.submission.Direction != station.DirectionIn {
		t.Fatalf("submission = %#v", checkins.submission)
	}
	var result legacyCheckin
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.ID != legacyCheckinID || result.UserID != legacyUserID || result.LocationID != legacyLocationID {
		t.Fatalf("response = %#v", result)
	}
}
