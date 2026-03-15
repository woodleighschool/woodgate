package authhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
	authhttp "github.com/woodleighschool/woodgate/internal/transport/http/auth"
)

type stubAPIKeys struct {
	id  uuid.UUID
	err error
}

func (stub stubAPIKeys) AuthenticateAPIKey(context.Context, string) (uuid.UUID, error) {
	if stub.err != nil {
		return uuid.Nil, stub.err
	}
	return stub.id, nil
}

type recordingAuthorizer struct {
	resource string
	action   string
	allowed  bool
	err      error
}

func (authorizer *recordingAuthorizer) Can(
	_ context.Context,
	_ authz.Principal,
	resource string,
	action string,
) (bool, error) {
	authorizer.resource = resource
	authorizer.action = action
	return authorizer.allowed, authorizer.err
}

func (authorizer *recordingAuthorizer) GrantedLocations(
	context.Context,
	authz.Principal,
	string,
) ([]uuid.UUID, error) {
	return nil, authorizer.err
}

func (authorizer *recordingAuthorizer) GrantedAssetTypes(
	context.Context,
	authz.Principal,
	string,
) ([]domain.AssetType, error) {
	return nil, authorizer.err
}

func (authorizer *recordingAuthorizer) IsAdmin(context.Context, authz.Principal) (bool, error) {
	return false, authorizer.err
}

func sessionAuthStub(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user := token.User{ID: "microsoft_session_user", Name: "Session User", Email: "session@woodleigh.test"}
		next.ServeHTTP(writer, token.SetUserInfo(request, user))
	})
}

type stubUsers struct {
	user domain.User
	err  error
}

func (stub stubUsers) GetUserByUPN(context.Context, string) (domain.User, error) {
	if stub.err != nil {
		return domain.User{}, stub.err
	}
	return stub.user, nil
}

func TestNewAPIMiddleware_AuthenticatesAPIKeyRequests(t *testing.T) {
	t.Parallel()

	authzRecorder := &recordingAuthorizer{allowed: true}
	middleware := authhttp.NewAPIMiddleware(
		sessionAuthStub,
		stubAPIKeys{id: uuid.MustParse("11111111-1111-1111-1111-111111111111")},
		stubUsers{},
		authzRecorder,
	)

	called := false
	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/users/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", nil)
	request.Header.Set("X-Api-Key", "woodgate_test_key")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !called {
		t.Fatalf("expected downstream handler to be called")
	}
	if authzRecorder.resource != "users" {
		t.Fatalf("expected resource users, got %q", authzRecorder.resource)
	}
	if authzRecorder.action != "write" {
		t.Fatalf("expected action write, got %q", authzRecorder.action)
	}
}

func TestNewAPIMiddleware_AuthenticatesSessionRequests(t *testing.T) {
	t.Parallel()

	authzRecorder := &recordingAuthorizer{allowed: true}
	middleware := authhttp.NewAPIMiddleware(
		sessionAuthStub,
		nil,
		stubUsers{user: domain.User{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")}},
		authzRecorder,
	)

	called := false
	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/locations", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	if !called {
		t.Fatalf("expected downstream handler to be called")
	}
	if authzRecorder.resource != "locations" {
		t.Fatalf("expected resource locations, got %q", authzRecorder.resource)
	}
	if authzRecorder.action != "read" {
		t.Fatalf("expected action read, got %q", authzRecorder.action)
	}
}

func TestNewAPIMiddleware_ReturnsUnauthorizedForInvalidAPIKey(t *testing.T) {
	t.Parallel()

	middleware := authhttp.NewAPIMiddleware(
		sessionAuthStub,
		stubAPIKeys{err: errors.New("invalid key")},
		stubUsers{},
		&recordingAuthorizer{allowed: true},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/checkins", nil)
	request.Header.Set("X-Api-Key", "invalid")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
	assertAuthErrorBody(t, response, "unauthorized")
}

func TestNewAPIMiddleware_ReturnsForbiddenWhenAuthorizationFails(t *testing.T) {
	t.Parallel()

	middleware := authhttp.NewAPIMiddleware(
		sessionAuthStub,
		nil,
		stubUsers{user: domain.User{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222")}},
		&recordingAuthorizer{allowed: false},
	)

	handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
	assertAuthErrorBody(t, response, "forbidden")
}

func assertAuthErrorBody(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()

	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body["error"] != want {
		t.Fatalf("error body = %q, want %q", body["error"], want)
	}
}
