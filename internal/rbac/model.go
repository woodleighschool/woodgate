// Package rbac owns the application's resource catalogue and role policy.
package rbac

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/openapischema"
)

// Application authorization resources.
const (
	ResourceUsers     authz.Resource = "users"
	ResourceGroups    authz.Resource = "groups"
	ResourceDirectory authz.Resource = "directory"
	ResourceLocations authz.Resource = "locations"
	ResourceCheckins  authz.Resource = "checkins"
	ResourceStations  authz.Resource = "stations"
	ResourceRoles     authz.Resource = "authz.roles"
)

var definitions = [...]Definition{
	{Resource: ResourceUsers, DisplayName: "Users"},
	{Resource: ResourceGroups, DisplayName: "Groups"},
	{Resource: ResourceDirectory, DisplayName: "Directory sync"},
	{Resource: ResourceLocations, DisplayName: "Locations"},
	{Resource: ResourceCheckins, DisplayName: "Check-ins"},
	{Resource: ResourceStations, DisplayName: "Stations"},
	{Resource: ResourceRoles, DisplayName: "Roles"},
}

// Definition describes one resource exposed to role-management clients.
type Definition struct {
	Resource    authz.Resource `json:"resource"`
	DisplayName string         `json:"display_name"`
}

// Definitions returns the application's immutable resource catalogue.
func Definitions() []Definition {
	return slices.Clone(definitions[:])
}

// Role is one named collection of resource permissions.
type Role struct {
	ID          int64                           `json:"id"`
	Key         string                          `json:"key"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Builtin     bool                            `json:"builtin"`
	Permissions map[authz.Resource]authz.Access `json:"permissions" db:"-"`
	CreatedAt   time.Time                       `json:"created_at"`
	UpdatedAt   time.Time                       `json:"updated_at"`
}

// RoleMutation contains the writable policy of a custom role.
type RoleMutation struct {
	Key         string                          `json:"key,omitempty"`
	Name        string                          `json:"name"`
	Description string                          `json:"description"`
	Permissions map[authz.Resource]authz.Access `json:"permissions"`
}

func (mutation *RoleMutation) normalize() {
	mutation.Key = strings.ToLower(strings.TrimSpace(mutation.Key))
	mutation.Name = strings.TrimSpace(mutation.Name)
	mutation.Description = strings.TrimSpace(mutation.Description)
	if mutation.Permissions == nil {
		mutation.Permissions = map[authz.Resource]authz.Access{}
	}
}

func (mutation *RoleMutation) validate(requireKey bool) error {
	if requireKey && mutation.Key == "" {
		return fmt.Errorf("%w: key is required", fault.ErrInvalidInput)
	}
	if mutation.Name == "" {
		return fmt.Errorf("%w: name is required", fault.ErrInvalidInput)
	}
	for resource, access := range mutation.Permissions {
		if !validResource(resource) {
			return fmt.Errorf("%w: unknown resource %q", fault.ErrInvalidInput, resource)
		}
		if !validAccess(access) {
			return fmt.Errorf("%w: invalid access %q", fault.ErrInvalidInput, access)
		}
	}
	return nil
}

// Resources returns the application's resource catalogue.
func Resources() []authz.Resource {
	resources := make([]authz.Resource, len(definitions))
	for i, definition := range definitions {
		resources[i] = definition.Resource
	}
	return resources
}

// ResourceSchema describes the application catalogue for API clients.
func ResourceSchema() *huma.Schema { return openapischema.StringEnum(Resources()...) }

// TransformSchema preserves the catalogue enum on resource descriptions.
func (Definition) TransformSchema(registry huma.Registry, schema *huma.Schema) *huma.Schema {
	registry.Map()["AuthzResource"] = ResourceSchema()
	schema.Properties["resource"] = &huma.Schema{Ref: "#/components/schemas/AuthzResource"}
	return schema
}

func validResource(resource authz.Resource) bool {
	for _, definition := range definitions {
		if definition.Resource == resource {
			return true
		}
	}
	return false
}

func validAccess(access authz.Access) bool {
	return access == authz.None || access == authz.View || access == authz.Edit
}
