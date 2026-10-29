//go:build postgres

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodgate/internal/checkin"
	"github.com/woodleighschool/woodgate/internal/rbac"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestCheckinHistoryFiltersAndScope(t *testing.T) {
	db, ctx := testdb.Open(t)
	_, err := db.Exec(ctx, `
 INSERT INTO users (id,email,name,department) VALUES
 (101,'first@example.invalid','First person','First, Second'),
 (102,'second@example.invalid','Second person','Third'),
 (103,'blank@example.invalid','Blank person',''),
 (104,'unused@example.invalid','Unused person','Unused');
 INSERT INTO locations (id,name,enabled) VALUES (201,'First location',true),(202,'Second location',false),(203,'Unused location',true);
 INSERT INTO checkins (id,user_id,location_id,direction,actor_kind,actor_app_id,occurred_at) VALUES
 (301,101,201,'check_in','user',gen_random_uuid(),'2026-01-01T10:00:00Z'),
 (302,102,201,'check_in','user',gen_random_uuid(),'2026-01-02T10:00:00Z'),
 (303,101,202,'check_in','user',gen_random_uuid(),'2026-01-03T10:00:00Z'),
 (304,101,201,'check_out','user',gen_random_uuid(),'2026-01-04T10:00:00Z'),
 (305,103,201,'check_in','user',gen_random_uuid(),'2026-01-05T10:00:00Z');`)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	routes := humachi.New(router, huma.DefaultConfig("test", "test"))
	grants := historyAccess(true)
	registerCheckins(routes, Dependencies{
		Service:    checkin.NewService(checkin.NewStore(db, nil), nil),
		Authorizer: &grants,
		Logger:     slog.New(slog.DiscardHandler),
	})
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequestWithContext(authn.WithPrincipal(ctx, &authn.Principal{ID: 101}), http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	const departments = "departments=First%2C%20Second&departments=Third"
	for _, tc := range []struct {
		name  string
		query string
		ids   []int64
		count int
	}{
		{name: "any selected department", query: departments, ids: []int64{304, 303, 302, 301}, count: 4},
		{name: "department name containing comma", query: "departments=First%2C%20Second", ids: []int64{304, 303, 301}, count: 3},
		{name: "scope and location", query: departments + "&user_id=101&location_id=201", ids: []int64{304, 301}, count: 2},
		{name: "direction and inclusive start", query: departments + "&direction=check_in&occurred_from=2026-01-02T10:00:00Z&occurred_before=2026-01-03T10:00:00Z", ids: []int64{302}, count: 1},
		{name: "pagination keeps filtered total", query: departments + "&sort=occurred_at&per_page=1&page=2", ids: []int64{302}, count: 4},
		{name: "department sorting", query: "sort=department", ids: []int64{305, 304, 303, 301, 302}, count: 5},
		{name: "empty user history", query: "user_id=104", ids: []int64{}, count: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := get("/api/checkins?" + tc.query)
			if response.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var result checkinListOutput
			if err := json.Unmarshal(response.Body.Bytes(), &result.Body); err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, len(result.Body.Items))
			for i, item := range result.Body.Items {
				ids[i] = item.ID
			}
			if !slices.Equal(ids, tc.ids) || result.Body.Count != tc.count {
				t.Fatalf("ids=%v count=%d, want %v count=%d", ids, result.Body.Count, tc.ids, tc.count)
			}
		})
	}
	response := get("/api/checkins/departments")
	var choices checkinDepartmentListOutput
	if response.Code != http.StatusOK {
		t.Fatalf("department choices: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &choices.Body); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(choices.Body.Items, []string{"First, Second", "Third"}) {
		t.Fatalf("departments=%v", choices.Body.Items)
	}
	response = get("/api/checkins/locations")
	var locations checkinLocationListOutput
	if response.Code != http.StatusOK {
		t.Fatalf("location choices: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &locations.Body); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(locations.Body.Items, []checkin.LocationSummary{{ID: 201, Name: "First location"}, {ID: 202, Name: "Second location"}}) {
		t.Fatalf("locations=%v", locations.Body.Items)
	}
	response = get("/api/checkins/users/104")
	var person checkin.PersonSummary
	if response.Code != http.StatusOK {
		t.Fatalf("user summary: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &person); err != nil {
		t.Fatal(err)
	}
	if person.ID != 104 || person.Name != "Unused person" {
		t.Fatalf("empty history lost user context: %+v", person)
	}
	grants = false
	for _, path := range []string{"/api/checkins", "/api/checkins/departments", "/api/checkins/locations", "/api/checkins/users/104"} {
		if response := get(path); response.Code != http.StatusForbidden {
			t.Fatalf("%s without check-in access: %d", path, response.Code)
		}
	}
}

type historyAccess bool

func (access *historyAccess) CanAll(_ context.Context, _ int64, requirements ...authz.Requirement) (bool, error) {
	for _, requirement := range requirements {
		if !bool(*access) || requirement.Resource != rbac.ResourceCheckins || requirement.Access != authz.View {
			return false, nil
		}
	}
	return true, nil
}

func TestListCheckinChoices(t *testing.T) {
	request, _, _ := newCheckinTestAPI(t)
	response := request(http.MethodGet, "/api/checkins/available-locations?q=recep&per_page=1", "")
	var locations availableLocationListOutput
	if response.Code != http.StatusOK {
		t.Fatalf("locations: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &locations.Body); err != nil {
		t.Fatal(err)
	}
	if locations.Body.Count != 1 || !slices.Equal(locations.Body.Items, []checkin.LocationSummary{{ID: 301, Name: "Reception"}}) {
		t.Fatalf("locations without history: %+v", locations.Body)
	}
	response = request(http.MethodGet, "/api/checkins/available-locations", "")
	if response.Code != http.StatusOK {
		t.Fatalf("locations: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &locations.Body); err != nil {
		t.Fatal(err)
	}
	if locations.Body.Count != 2 || !slices.Equal(locations.Body.Items, []checkin.LocationSummary{{ID: 301, Name: "Reception"}, {ID: 303, Name: "Unrestricted"}}) {
		t.Fatalf("enabled locations: %+v", locations.Body)
	}
	for _, tc := range []struct {
		query string
		ids   []int64
		count int
	}{
		{query: "location_id=301", ids: []int64{102, 103}, count: 2},
		{query: "location_id=301&q=blake", ids: []int64{103}, count: 1},
		{query: "location_id=301&per_page=1&page=2", ids: []int64{103}, count: 2},
		{query: "location_id=303", ids: []int64{102, 103, 104, 101}, count: 4},
	} {
		response := request(http.MethodGet, "/api/checkins/people?"+tc.query, "")
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.query, response.Code, response.Body.String())
		}
		var people checkinPeopleOutput
		if err := json.Unmarshal(response.Body.Bytes(), &people.Body); err != nil {
			t.Fatal(err)
		}
		ids := make([]int64, len(people.Body.Items))
		for i, person := range people.Body.Items {
			ids[i] = person.ID
		}
		if !slices.Equal(ids, tc.ids) || people.Body.Count != tc.count {
			t.Fatalf("%s: ids=%v count=%d", tc.query, ids, people.Body.Count)
		}
	}
}

func TestCreateCheckin(t *testing.T) {
	request, access, _ := newCheckinTestAPI(t)

	const body = `{"location_id":301,"user_id":102,"direction":"check_in","notes":"  Appointment  "}`
	var previousID int64
	for range 2 {
		response := request(http.MethodPost, "/api/checkins", body)
		if response.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", response.Code, response.Body.String())
		}
		var item checkin.Checkin
		if err := json.Unmarshal(response.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		if item.Person.ID != 102 || item.CreatedBy == nil || item.CreatedBy.ID != 101 || item.ActorKind != "user" || item.ActorName != "Recorder" || item.Station != nil || item.PhotoObjectID != nil || item.Notes != "Appointment" || item.ID == previousID {
			t.Fatalf("user attribution or repeated event: %+v", item)
		}
		previousID = item.ID
	}
	for _, tc := range []struct{ name, body string }{
		{"ineligible person", `{"location_id":301,"user_id":104,"direction":"check_in","notes":""}`},
		{"deleted person", `{"location_id":301,"user_id":105,"direction":"check_out","notes":""}`},
		{"disabled location", `{"location_id":302,"user_id":102,"direction":"check_out","notes":""}`},
	} {
		response := request(http.MethodPost, "/api/checkins", tc.body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", tc.name, response.Code, response.Body.String())
		}
	}
	if response := request(http.MethodGet, "/api/checkins/people?location_id=302", ""); response.Code != http.StatusOK {
		t.Fatalf("disabled location choices: %d", response.Code)
	}
	access.edit = false
	for _, path := range []string{"/api/checkins/available-locations", "/api/checkins/people?location_id=301"} {
		if response := request(http.MethodGet, path, ""); response.Code != http.StatusForbidden {
			t.Fatalf("read-only %s: %d", path, response.Code)
		}
	}
	if response := request(http.MethodPost, "/api/checkins", body); response.Code != http.StatusForbidden {
		t.Fatalf("read-only create: %d", response.Code)
	}
}

func TestUpdateCheckin(t *testing.T) {
	request, access, _ := newCheckinTestAPI(t)
	created := decodeCheckin(t, request(http.MethodPost, "/api/checkins", `{"location_id":301,"user_id":102,"direction":"check_in","notes":"Appointment"}`), http.StatusCreated)
	if !created.OccurredAt.Equal(created.CreatedAt) || !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Fatalf("creation timestamps differ: %+v", created)
	}
	path := fmt.Sprintf("/api/checkins/%d", created.ID)
	const body = `{"location_id":303,"user_id":103,"direction":"check_out","notes":"  Corrected  ","occurred_at":"2026-01-02T10:00:00+11:00"}`
	updated := decodeCheckin(t, request(http.MethodPut, path, body), http.StatusOK)
	expectedTime := time.Date(2026, time.January, 1, 23, 0, 0, 0, time.UTC)
	if updated.ID != created.ID || updated.Person.ID != 103 || updated.Location.ID != 303 || updated.Direction != checkin.DirectionOut || updated.Notes != "Corrected" || !updated.OccurredAt.Equal(expectedTime) {
		t.Fatalf("updated record: %+v", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.After(created.UpdatedAt) || updated.ActorAppID != created.ActorAppID || updated.CreatedBy == nil || updated.CreatedBy.ID != created.CreatedBy.ID || updated.ActorKind != created.ActorKind {
		t.Fatalf("creation metadata changed: before=%+v after=%+v", created, updated)
	}
	saved := decodeCheckin(t, request(http.MethodGet, path, ""), http.StatusOK)
	if saved.Person.ID != updated.Person.ID || !saved.OccurredAt.Equal(updated.OccurredAt) || saved.Notes != updated.Notes {
		t.Fatalf("update not persisted: %+v", saved)
	}
	response := request(http.MethodGet, "/api/checkins?occurred_from=2026-01-01T23:00:00Z&occurred_before=2026-01-02T00:00:00Z&sort=occurred_at", "")
	var page checkinListOutput
	if response.Code != http.StatusOK {
		t.Fatalf("list: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &page.Body); err != nil {
		t.Fatal(err)
	}
	if page.Body.Count != 1 || len(page.Body.Items) != 1 || page.Body.Items[0].ID != created.ID {
		t.Fatalf("corrected time filter: %+v", page.Body)
	}
	if response := request(http.MethodPut, "/api/checkins/999999", body); response.Code != http.StatusNotFound {
		t.Fatalf("missing update: %d", response.Code)
	}
	access.edit = false
	if response := request(http.MethodPut, path, body); response.Code != http.StatusForbidden {
		t.Fatalf("read-only update: %d", response.Code)
	}
}

func TestUpdateCheckinValidatesReplacement(t *testing.T) {
	request, _, _ := newCheckinTestAPI(t)
	created := decodeCheckin(t, request(http.MethodPost, "/api/checkins", `{"location_id":301,"user_id":102,"direction":"check_in","notes":"Appointment"}`), http.StatusCreated)
	path := fmt.Sprintf("/api/checkins/%d", created.ID)
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"ineligible person", `{"location_id":301,"user_id":104,"direction":"check_out","notes":"","occurred_at":"2026-01-02T10:00:00Z"}`, http.StatusBadRequest},
		{"deleted person", `{"location_id":301,"user_id":105,"direction":"check_out","notes":"","occurred_at":"2026-01-02T10:00:00Z"}`, http.StatusBadRequest},
		{"disabled location", `{"location_id":302,"user_id":102,"direction":"check_out","notes":"","occurred_at":"2026-01-02T10:00:00Z"}`, http.StatusBadRequest},
		{"missing time", `{"location_id":301,"user_id":102,"direction":"check_out","notes":""}`, http.StatusUnprocessableEntity},
		{"invalid time", `{"location_id":301,"user_id":102,"direction":"check_out","notes":"","occurred_at":"invalid"}`, http.StatusUnprocessableEntity},
		{"invalid direction", `{"location_id":301,"user_id":102,"direction":"invalid","notes":"","occurred_at":"2026-01-02T10:00:00Z"}`, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := request(http.MethodPut, path, tc.body)
			if response.Code != tc.status {
				t.Fatalf("update: %d %s", response.Code, response.Body.String())
			}
		})
	}
	saved := decodeCheckin(t, request(http.MethodGet, path, ""), http.StatusOK)
	if saved.Person.ID != created.Person.ID || saved.Location.ID != created.Location.ID || saved.Direction != created.Direction || saved.Notes != created.Notes || !saved.OccurredAt.Equal(created.OccurredAt) || !saved.UpdatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("invalid update changed record: %+v", saved)
	}
}

func TestUpdateCheckinRetainsHistoricalSelections(t *testing.T) {
	request, _, db := newCheckinTestAPI(t)
	created := decodeCheckin(t, request(http.MethodPost, "/api/checkins", `{"location_id":301,"user_id":102,"direction":"check_in","notes":"Appointment"}`), http.StatusCreated)
	if _, err := db.Exec(t.Context(), `UPDATE locations SET enabled=false WHERE id=301; DELETE FROM directory_group_memberships WHERE user_id=102; UPDATE users SET deleted_at=now() WHERE id=102;`); err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/checkins/%d", created.ID)
	saved := decodeCheckin(t, request(http.MethodPut, path, `{"location_id":301,"user_id":102,"direction":"check_out","notes":"","occurred_at":"2026-01-02T10:00:00Z"}`), http.StatusOK)
	if saved.Person.ID != 102 || saved.Location.ID != 301 || saved.Notes != "" || saved.Direction != checkin.DirectionOut {
		t.Fatalf("historical selection: %+v", saved)
	}
}

func decodeCheckin(t *testing.T, response *httptest.ResponseRecorder, status int) checkin.Checkin {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d, want %d: %s", response.Code, status, response.Body.String())
	}
	var item checkin.Checkin
	if err := json.Unmarshal(response.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	return item
}

func newCheckinTestAPI(t *testing.T) (func(string, string, string) *httptest.ResponseRecorder, *checkinAccess, *pgxpool.Pool) {
	t.Helper()
	db, ctx := testdb.Open(t)
	_, err := db.Exec(ctx, `
INSERT INTO users (id,email,name,department,deleted_at) VALUES
 (101,'recorder@example.invalid','Recorder','Office',NULL),
 (102,'alex@example.invalid','Alex Example','Students',NULL),
 (103,'blake@example.invalid','Blake Example','Students',NULL),
 (104,'outsider@example.invalid','Outside Person','Other',NULL),
 (105,'deleted@example.invalid','Deleted Person','Students',now());
INSERT INTO directory_groups (id,source,external_id,display_name) VALUES (201,'entra','students','Students');
INSERT INTO directory_group_memberships (group_id,user_id) VALUES (201,102),(201,103),(201,105);
INSERT INTO locations (id,name,enabled,notes,photo) VALUES
 (301,'Reception',true,false,true),(302,'Disabled',false,true,false),(303,'Unrestricted',true,false,false);
INSERT INTO location_directory_groups (location_id,group_id) VALUES (301,201);`)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	routes := humachi.New(router, huma.DefaultConfig("test", "test"))
	access := checkinAccess{edit: true}
	registerCheckins(routes, Dependencies{
		Service:    checkin.NewService(checkin.NewStore(db, nil), nil),
		Authorizer: &access, Logger: slog.New(slog.DiscardHandler),
	})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequestWithContext(authn.WithPrincipal(ctx, &authn.Principal{ID: 101}), method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, r)
		return response
	}
	return request, &access, db
}

type checkinAccess struct{ edit bool }

func (access *checkinAccess) CanAll(_ context.Context, _ int64, requirements ...authz.Requirement) (bool, error) {
	for _, requirement := range requirements {
		if requirement.Resource != rbac.ResourceCheckins || requirement.Access == authz.Edit && !access.edit {
			return false, nil
		}
	}
	return true, nil
}
