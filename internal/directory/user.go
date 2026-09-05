package directory

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/validation"
)

// User is an application account row.
type User struct {
	ID                int64         `json:"id"`
	Email             string        `json:"email"                         format:"email"`
	Name              string        `json:"name"`
	PasswordHash      string        `json:"-"`
	Roles             []RoleSummary `json:"roles" db:"-"`
	EffectiveRoles    []RoleSummary `json:"effective_roles" db:"-"`
	Source            Source        `json:"source"`
	ExternalID        string        `json:"external_id,omitempty"`
	UserPrincipalName string        `json:"user_principal_name,omitempty"`
	MailNickname      string        `json:"mail_nickname,omitempty"`
	GivenName         string        `json:"given_name,omitempty"`
	FamilyName        string        `json:"family_name,omitempty"`
	Department        string        `json:"department,omitempty"`
	DeletedAt         *time.Time    `json:"deleted_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// UserSummary is the small user projection attached to another resource.
type UserSummary struct {
	ID    int64  `json:"id"`
	Email string `json:"email" format:"email"`
	Name  string `json:"name"`
}

// RoleSummary is the small role identity projected onto a user or group.
type RoleSummary struct {
	ID   int64  `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// UserListParams filters paginated user lists.
type UserListParams struct {
	ListParams listing.Params

	Values     []string
	RoleValues []string
	Source     string `validate:"omitempty,oneof=local entra"`
	GroupID    int64  `validate:"gte=0"`
}

// UserCreate contains fields needed to create a user.
type UserCreate struct {
	Email    string  `json:"email"          format:"email" validate:"required,email"`
	Name     string  `json:"name,omitempty"`
	Password string  `json:"password"       minLength:"12"`
	RoleIDs  []int64 `json:"role_ids"`
}

// UserMutation replaces the writable fields of a user.
type UserMutation struct {
	Name     string  `json:"name"`
	Password *string `json:"password,omitempty"`
	RoleIDs  []int64 `json:"role_ids"`
}

func (params *UserListParams) normalize() {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Values = listing.NormalizeValues(params.Values)
	params.RoleValues = listing.NormalizeValues(params.RoleValues)
	params.Source = strings.TrimSpace(params.Source)
}

func (params *UserListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	for _, value := range params.RoleValues {
		if value == "none" {
			continue
		}
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			return fmt.Errorf("%w: invalid role %q", fault.ErrInvalidInput, value)
		}
	}
	return nil
}

func (params *UserCreate) normalize() {
	params.Email = strings.TrimSpace(params.Email)
	params.Name = strings.TrimSpace(params.Name)
	params.RoleIDs = normalizeRoleIDs(params.RoleIDs)
}

func (params *UserCreate) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return validateRoleIDs(params.RoleIDs)
}

func (params *UserMutation) normalize() {
	params.Name = strings.TrimSpace(params.Name)
	params.RoleIDs = normalizeRoleIDs(params.RoleIDs)
}

func (params *UserMutation) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return validateRoleIDs(params.RoleIDs)
}

// Account is the signed-in user's self-view, including their retrievable API key.
type Account struct {
	User            User       `json:"user"`
	APIKey          string     `json:"api_key,omitempty"`
	APIKeyCreatedAt *time.Time `json:"api_key_created_at,omitempty"`
}
