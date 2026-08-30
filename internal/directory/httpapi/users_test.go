package httpapi

import (
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/fault"
)

func TestRegisterOpenAPIDeclaresDirectoryAuthorization(t *testing.T) {
	t.Parallel()

	humaAPI, routes := api.NewSchema("test")
	RegisterOpenAPI(routes)

	tests := []struct {
		method   string
		path     string
		resource string
		access   string
	}{
		{method: http.MethodGet, path: "/api/users", resource: "users", access: "view"},
		{method: http.MethodPost, path: "/api/users", resource: "users", access: "edit"},
		{method: http.MethodGet, path: "/api/groups", resource: "groups", access: "view"},
		{method: http.MethodGet, path: "/api/directory/sync", resource: "directory", access: "view"},
		{method: http.MethodPost, path: "/api/directory/sync", resource: "directory", access: "edit"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			path := humaAPI.OpenAPI().Paths[tt.path]
			if path == nil {
				t.Fatalf("path not registered")
			}
			var operation *huma.Operation
			switch tt.method {
			case http.MethodGet:
				operation = path.Get
			case http.MethodPost:
				operation = path.Post
			default:
				t.Fatalf("unsupported test method %q", tt.method)
			}
			if operation == nil {
				t.Fatalf("operation not registered")
			}
			requirement, ok := operation.Extensions["x-authz"].(map[string]any)
			if !ok {
				t.Fatalf("x-authz = %#v, want object", operation.Extensions["x-authz"])
			}
			if got := requirement["resource"]; got != tt.resource {
				t.Fatalf("resource = %v, want %q", got, tt.resource)
			}
			if got := requirement["access"]; got != tt.access {
				t.Fatalf("access = %v, want %q", got, tt.access)
			}
		})
	}
}

func TestUserMutationErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "not found", err: fault.ErrNotFound, wantStatus: 404},
		{name: "already exists", err: fault.ErrAlreadyExists, wantStatus: 409},
		{name: "weak password", err: directory.ErrWeakPassword, wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mapped := userMutationError(tt.err)
			status, ok := errors.AsType[huma.StatusError](mapped)
			if !ok {
				t.Fatalf("not a huma.StatusError: %v", mapped)
			}
			if status.GetStatus() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status.GetStatus(), tt.wantStatus)
			}
		})
	}
}
