package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

type contractService struct {
	AdminService

	options domain.GroupListOptions
}

func (service *contractService) ListGroups(_ context.Context, options domain.GroupListOptions) ([]domain.Group, int32, error) {
	service.options = options
	return []domain.Group{}, 0, nil
}
func (service *contractService) CreateLocation(_ context.Context, name, description string, enabled, notes, photo bool, background, logo *uuid.UUID, groups []uuid.UUID) (domain.Location, error) {
	return domain.Location{ID: uuid.MustParse("12345678-1234-1234-1234-123456789abc"), Name: name, Description: description, Enabled: enabled, Notes: notes, Photo: photo, BackgroundAssetID: background, LogoAssetID: logo, GroupIDs: groups}, nil
}

func TestV1ParameterContract(t *testing.T) {
	service := &contractService{}
	router := chi.NewRouter()
	New(service, nil).RegisterRoutes(router)
	for _, test := range []struct {
		query  string
		status int
		detail string
	}{
		{"", 200, ""},
		{"?limit=10&offset=3&search=staff&sort=name&order=desc", 200, ""},
		{"?limit=0", 400, "limit must be >= 1"},
		{"?offset=-1", 400, "offset must be >= 0"},
		{"?limit=invalid", 400, "Request parameters are invalid."},
		{"?limit=2147483648", 400, "Request parameters are invalid."},
		{"?limit=", 400, "Request parameters are invalid."},
		{"?limit=1&limit=2", 400, "Request parameters are invalid."},
	} {
		t.Run(test.query, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), "GET", "/groups"+test.query, nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; %s", response.Code, test.status, response.Body.String())
			}
			if test.status == 200 {
				if strings.TrimSpace(response.Body.String()) != `{"rows":[],"total":0}` {
					t.Fatalf("list body = %s", response.Body.String())
				}
				if strings.Contains(test.query, "search=staff") && (service.options.Limit != 10 || service.options.Offset != 3 || service.options.Search != "staff" || service.options.Order != "desc") {
					t.Fatalf("list options = %+v", service.options)
				}
				return
			}
			var problem Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Code != "invalid_request" || problem.Detail != test.detail || problem.Status != 400 {
				t.Fatalf("problem = %+v", problem)
			}
		})
	}
}

func TestV1JSONContract(t *testing.T) {
	router := chi.NewRouter()
	New(&contractService{}, nil).RegisterRoutes(router)
	for _, test := range []struct {
		name, body string
		status     int
		detail     string
	}{
		{"valid", `{"name":" Reception ","description":" Entry ","enabled":true,"notes":false,"photo":false,"group_ids":[]}`, 201, ""},
		{"empty", "", 400, "request body is required"},
		{"malformed", "{", 400, "request body is invalid"},
		{"unknown field", `{"name":"Reception","unknown":true}`, 400, "request body is invalid"},
		{"multiple objects", `{"name":"Reception"} {}`, 400, "request body must contain a single JSON object"},
		{"blank name", `{"name":" "}`, 422, "Location is invalid."},
		{"invalid UUID", `{"name":"Reception","group_ids":["not-a-uuid"]}`, 400, "request body is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), "POST", "/locations", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "text/plain")
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; %s", response.Code, test.status, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
			}
			if test.status == 201 {
				var location Location
				if err := json.Unmarshal(response.Body.Bytes(), &location); err != nil {
					t.Fatal(err)
				}
				if location.Name != "Reception" || location.Description != "Entry" || !location.Enabled || location.GroupIDs == nil {
					t.Fatalf("location = %+v", location)
				}
				return
			}
			var problem Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem.Detail != test.detail || int(problem.Status) != test.status {
				t.Fatalf("problem = %+v", problem)
			}
			if test.status == http.StatusUnprocessableEntity {
				requireFieldError(t, problem, "name")
			}
		})
	}
}

func (service *contractService) GetGroup(_ context.Context, id uuid.UUID) (domain.Group, error) {
	return domain.Group{ID: id, Name: "Staff"}, nil
}

func TestV1UUIDPaths(t *testing.T) {
	router := chi.NewRouter()
	New(&contractService{}, nil).RegisterRoutes(router)
	id := "12345678-1234-1234-1234-123456789abc"
	for _, value := range []string{id, "not-a-uuid"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), "GET", "/groups/"+value, nil))
		if value == id {
			var group Group
			if err := json.Unmarshal(response.Body.Bytes(), &group); err != nil {
				t.Fatal(err)
			}
			if response.Code != 200 || group.ID.String() != id {
				t.Fatalf("group response = %d %s", response.Code, response.Body.String())
			}
		} else if response.Code != 400 {
			t.Fatalf("invalid path response = %d %s", response.Code, response.Body.String())
		}
	}
}

type contractAuthorizer struct {
	authz.Authorizer

	locations authz.Scope[uuid.UUID]
	assets    authz.Scope[domain.AssetType]
}

func (authorizer contractAuthorizer) CheckinScope(context.Context, authz.Principal, string) (authz.Scope[uuid.UUID], error) {
	return authorizer.locations, nil
}
func (authorizer contractAuthorizer) AssetScope(context.Context, authz.Principal, string) (authz.Scope[domain.AssetType], error) {
	return authorizer.assets, nil
}
func (service *contractService) GetLocation(_ context.Context, id uuid.UUID) (domain.Location, error) {
	return domain.Location{ID: id, Enabled: true, Photo: true, Notes: false}, nil
}

func TestV1MultipartValidationAndScope(t *testing.T) {
	id := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	principal := authz.Principal{Kind: authz.PrincipalKindAPIKey, ID: id.String()}
	for _, test := range []struct {
		name, path string
		allowed    bool
		values     map[string]string
		status     int
		field      string
	}{
		{"asset denied before decoding", "/assets", false, nil, 403, ""},
		{"asset file required", "/assets", true, map[string]string{"name": "Logo"}, 422, "file"},
		{"checkin wrong location", "/checkins", false, map[string]string{"user_id": id.String(), "location_id": id.String(), "direction": "check_in"}, 403, ""},
		{"checkin photo required", "/checkins", true, map[string]string{"user_id": id.String(), "location_id": id.String(), "direction": "check_in"}, 422, "photo"},
		{"checkin malformed UUID", "/checkins", true, map[string]string{"user_id": "invalid", "location_id": id.String(), "direction": "check_in"}, 422, "user_id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := chi.NewRouter()
			authorizer := contractAuthorizer{locations: authz.Scope[uuid.UUID]{All: test.allowed}, assets: authz.Scope[domain.AssetType]{All: test.allowed}}
			New(&contractService{}, authorizer).RegisterRoutes(router)
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			for key, value := range test.values {
				if err := form.WriteField(key, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := form.Close(); err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequestWithContext(authz.WithPrincipal(t.Context(), principal), "POST", test.path, &body)
			request.Header.Set("Content-Type", form.FormDataContentType())
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; %s", response.Code, test.status, response.Body.String())
			}
			var problem Problem
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if test.field != "" {
				requireFieldError(t, problem, test.field)
			}
		})
	}
}

func TestV1SchemaPreservesCompanionContract(t *testing.T) {
	schema := OpenAPI()
	if len(schema.Paths) != 16 {
		t.Fatalf("paths = %d", len(schema.Paths))
	}
	checkins := schema.Paths["/checkins"].Post
	if checkins.OperationID != "createCheckin" || checkins.RequestBody.Content["multipart/form-data"] == nil {
		t.Fatal("check-in multipart operation missing")
	}
	checkin := schema.Components.Schemas.Map()["CheckinCreateRequest"]
	if checkin.Properties["location_id"].Format != "uuid" || checkin.Properties["photo"].Format != "binary" {
		t.Fatalf("check-in field formats = %+v", checkin.Properties)
	}
	location := schema.Components.Schemas.Map()["Location"]
	if !location.Properties["background_asset_id"].Nullable || !location.Properties["logo_asset_id"].Nullable {
		t.Fatal("location attachments must accept null")
	}
	if location.Properties["photo"].Format != "" {
		t.Fatal("location photo setting must remain a boolean")
	}
	if location.Properties["group_ids"].Items.Format != "uuid" {
		t.Fatal("location group ids must be UUIDs")
	}
	if schema.Paths["/locations/{id}"].Patch == nil || schema.Paths["/locations/{id}"].Put != nil {
		t.Fatal("location update must remain PATCH")
	}
	if schema.Components.SecuritySchemes["sessionAuth"].Name != "woodgate_session" || schema.Components.SecuritySchemes["apiKeyAuth"].Name != "X-API-Key" {
		t.Fatal("authentication scheme changed")
	}
}

func requireFieldError(t *testing.T, problem Problem, field string) {
	t.Helper()
	if problem.FieldErrors == nil || len(*problem.FieldErrors) == 0 {
		t.Fatal("missing field errors")
	}
	if (*problem.FieldErrors)[0].Field != field {
		t.Fatalf("field error = %q, want %q", (*problem.FieldErrors)[0].Field, field)
	}
}

func TestV1JSONBodyLimit(t *testing.T) {
	router := chi.NewRouter()
	New(&contractService{}, nil).RegisterRoutes(router)
	const prefix = `{"name":"Reception","description":"`
	const suffix = `"}`
	for _, size := range []int{maxJSONBodyBytes - 1, maxJSONBodyBytes, maxJSONBodyBytes + 1} {
		body := prefix + strings.Repeat(" ", size-len(prefix)-len(suffix)) + suffix
		request := httptest.NewRequestWithContext(t.Context(), "POST", "/locations", strings.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		expected := http.StatusCreated
		if size > maxJSONBodyBytes {
			expected = http.StatusBadRequest
		}
		if response.Code != expected {
			t.Fatalf("%d-byte body: status = %d, want %d; %s", size, response.Code, expected, response.Body.String())
		}
	}
}

func TestV1InvalidPathPrecedesBodyErrors(t *testing.T) {
	router := chi.NewRouter()
	New(&contractService{}, nil).RegisterRoutes(router)
	for _, body := range []string{"", "{"} {
		request := httptest.NewRequestWithContext(t.Context(), "PATCH", "/locations/not-a-uuid", strings.NewReader(body))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var problem Problem
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || problem.Detail != "Request parameters are invalid." {
			t.Fatalf("parameter error = %d %+v", response.Code, problem)
		}
	}
}
