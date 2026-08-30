package authorization

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodgate/internal/fault"
)

type Resource string

type Access string

const (
	None Access = "none"
	View Access = "view"
	Edit Access = "edit"
)

func (access Access) level() int16 {
	switch access {
	case None:
		return 0
	case View:
		return 1
	case Edit:
		return 2
	default:
		return 0
	}
}

func accessFromLevel(level int16) Access {
	switch level {
	case 1:
		return View
	case 2:
		return Edit
	default:
		return None
	}
}

type Definition struct {
	Resource    Resource `json:"resource"`
	DisplayName string   `json:"display_name"`
}

var Definitions = []Definition{
	{Resource: "users", DisplayName: "Users"},
	{Resource: "groups", DisplayName: "Groups"},
	{Resource: "directory", DisplayName: "Directory sync"},
	{Resource: "locations", DisplayName: "Locations"},
	{Resource: "checkins", DisplayName: "Check-ins"},
	{Resource: "stations", DisplayName: "Stations"},
	{Resource: "authz.roles", DisplayName: "Roles"},
	{Resource: "authz.assignments", DisplayName: "Role assignments"},
}

func validResource(resource Resource) bool {
	for _, definition := range Definitions {
		if definition.Resource == resource {
			return true
		}
	}
	return false
}

type Role struct {
	ID          int64               `json:"id"`
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Builtin     bool                `json:"builtin"`
	Permissions map[Resource]Access `json:"permissions" db:"-"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type RoleMutation struct {
	Key         string              `json:"key,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Permissions map[Resource]Access `json:"permissions"`
}

func (mutation *RoleMutation) normalize() {
	mutation.Key = strings.ToLower(strings.TrimSpace(mutation.Key))
	mutation.Name = strings.TrimSpace(mutation.Name)
	mutation.Description = strings.TrimSpace(mutation.Description)
	if mutation.Permissions == nil {
		mutation.Permissions = map[Resource]Access{}
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
		if access != None && access != View && access != Edit {
			return fmt.Errorf("%w: invalid access %q", fault.ErrInvalidInput, access)
		}
	}
	return nil
}

type SubjectKind string

const (
	SubjectUser  SubjectKind = "user"
	SubjectGroup SubjectKind = "group"
)

type Assignment struct {
	SubjectKind SubjectKind `json:"subject_kind"`
	SubjectID   int64       `json:"subject_id"`
	RoleID      int64       `json:"role_id"`
}

type AssignmentMutation struct {
	SubjectKind SubjectKind `json:"subject_kind"`
	SubjectID   int64       `json:"subject_id" minimum:"1"`
	RoleIDs     []int64     `json:"role_ids"`
}

func (mutation AssignmentMutation) validate() error {
	if mutation.SubjectKind != SubjectUser && mutation.SubjectKind != SubjectGroup {
		return fmt.Errorf("%w: invalid subject kind %q", fault.ErrInvalidInput, mutation.SubjectKind)
	}
	if mutation.SubjectID <= 0 {
		return fmt.Errorf("%w: subject_id must be positive", fault.ErrInvalidInput)
	}
	for _, roleID := range mutation.RoleIDs {
		if roleID <= 0 {
			return fmt.Errorf("%w: role_ids must be positive", fault.ErrInvalidInput)
		}
	}
	return nil
}
