package httpapi

import (
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/woodgate/internal/directory"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/fault"
)

func TestRegisterOpenAPIDeclaresDirectoryAuthorization(t *testing.T) {
	t.Parallel()

	humaAPI, routes := api.NewSchema("test")
	RegisterOpenAPI(routes)

	tests := []struct {
		method       string
		path         string
		requirements []map[string]any
	}{
		{method: http.MethodGet, path: "/api/users", requirements: requirements("users", "view")},
		{method: http.MethodGet, path: "/api/users/roles", requirements: requirements("users", "view")},
		{method: http.MethodPost, path: "/api/users", requirements: requirements("users", "edit", "authz.roles", "edit")},
		{method: http.MethodPut, path: "/api/users/{id}", requirements: requirements("users", "edit", "authz.roles", "edit")},
		{method: http.MethodDelete, path: "/api/users/{id}", requirements: requirements("users", "edit", "authz.roles", "edit")},
		{method: http.MethodGet, path: "/api/groups", requirements: requirements("groups", "view")},
		{method: http.MethodPut, path: "/api/groups/{id}", requirements: requirements("groups", "edit", "authz.roles", "edit")},
		{method: http.MethodGet, path: "/api/directory/sync", requirements: requirements("directory", "view")},
		{method: http.MethodPost, path: "/api/directory/sync", requirements: requirements("directory", "edit")},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			path := humaAPI.OpenAPI().Paths[tt.path]
			if path == nil {
				t.Fatalf("path not registered")
			}
			var operation *huma.Operation
			switch tt.method {
			case http.MethodDelete:
				operation = path.Delete
			case http.MethodGet:
				operation = path.Get
			case http.MethodPost:
				operation = path.Post
			case http.MethodPut:
				operation = path.Put
			default:
				t.Fatalf("unsupported test method %q", tt.method)
			}
			if operation == nil {
				t.Fatalf("operation not registered")
			}
			extension, ok := operation.Extensions["x-authz"].(map[string]any)
			if !ok {
				t.Fatalf("x-authz = %#v, want object", operation.Extensions["x-authz"])
			}
			got := []map[string]any{extension}
			if all, exists := extension["all"]; exists {
				got, ok = all.([]map[string]any)
				if !ok {
					t.Fatalf("x-authz.all = %#v, want requirement list", all)
				}
			}
			if !reflect.DeepEqual(got, tt.requirements) {
				t.Fatalf("requirements = %#v, want %#v", got, tt.requirements)
			}
		})
	}
}

func requirements(values ...string) []map[string]any {
	out := make([]map[string]any, 0, len(values)/2)
	for len(values) >= 2 {
		out = append(out, map[string]any{"resource": values[0], "access": values[1]})
		values = values[2:]
	}
	return out
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
