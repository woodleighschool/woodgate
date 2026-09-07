package httpapi

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type APIKey struct {
	Access     []PermissionGrant `json:"access" nullable:"false"`
	Admin      bool              `json:"admin"`
	CreatedAt  time.Time         `json:"created_at"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty" nullable:"true"`
	ID         uuid.UUID         `json:"id"`
	KeyPrefix  string            `json:"key_prefix"`
	LastUsedAt *time.Time        `json:"last_used_at,omitempty" nullable:"true"`
	Name       string            `json:"name"`
}

type APIKeyAccessWriteRequest struct {
	Access []PermissionGrant `json:"access" nullable:"false"`
	Admin  bool              `json:"admin"`
}

type APIKeyCreateRequest struct {
	ExpiresAt *time.Time `json:"expires_at,omitempty" nullable:"true"`
	Name      string     `json:"name"`
}

type APIKeyListResponse struct {
	Rows  []APIKey `json:"rows" nullable:"false"`
	Total int32    `json:"total"`
}

type Asset struct {
	CreatedAt time.Time `json:"created_at"`
	ID        uuid.UUID `json:"id"`
	Name      *string   `json:"name,omitempty" nullable:"true"`
	Type      AssetType `json:"type"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url" format:"uri-reference"`
}

type AssetCreateRequest struct {
	File string  `json:"file" format:"binary"`
	Name *string `json:"name,omitempty"`
}

type AssetListResponse struct {
	Rows  []Asset `json:"rows" nullable:"false"`
	Total int32   `json:"total"`
}

type AssetType string

type AssetUpdateRequest struct {
	File *string `json:"file,omitempty" format:"binary"`
	Name *string `json:"name,omitempty"`
}

type Checkin struct {
	AssetID         *uuid.UUID            `json:"asset_id,omitempty" nullable:"true"`
	CreatedAt       time.Time             `json:"created_at"`
	CreatedByID     uuid.UUID             `json:"created_by_id"`
	CreatedByKind   PermissionSubjectKind `json:"created_by_kind"`
	Department      string                `json:"department"`
	Direction       CheckinDirection      `json:"direction"`
	ID              uuid.UUID             `json:"id"`
	LocationID      uuid.UUID             `json:"location_id"`
	LocationName    string                `json:"location_name"`
	Notes           string                `json:"notes"`
	PhotoURL        *string               `json:"photo_url,omitempty" nullable:"true"`
	UserDisplayName string                `json:"user_display_name"`
	UserID          uuid.UUID             `json:"user_id"`
}

type CheckinCreateRequest struct {
	Direction  CheckinDirection `json:"direction"`
	LocationID uuid.UUID        `json:"location_id"`
	Notes      *string          `json:"notes,omitempty"`
	Photo      *string          `json:"photo,omitempty" format:"binary"`
	UserID     uuid.UUID        `json:"user_id"`
}

type CheckinDirection string

type CheckinListResponse struct {
	Rows  []Checkin `json:"rows" nullable:"false"`
	Total int32     `json:"total"`
}

type CreateAPIKeyData struct {
	Access     []PermissionGrant `json:"access" nullable:"false"`
	Admin      bool              `json:"admin"`
	CreatedAt  time.Time         `json:"created_at"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty" nullable:"true"`
	ID         uuid.UUID         `json:"id"`
	KeyPrefix  string            `json:"key_prefix"`
	LastUsedAt *time.Time        `json:"last_used_at,omitempty" nullable:"true"`
	Name       string            `json:"name"`
	Secret     string            `json:"secret"`
}

type DepartmentOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DepartmentOptionListResponse struct {
	Rows  []DepartmentOption `json:"rows" nullable:"false"`
	Total int32              `json:"total"`
}

type FieldError struct {
	Code    *string `json:"code,omitempty"`
	Field   string  `json:"field"`
	Message string  `json:"message"`
}

type Group struct {
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	ID          uuid.UUID `json:"id"`
	MemberCount int32     `json:"member_count"`
	Name        string    `json:"name"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GroupListResponse struct {
	Rows  []Group `json:"rows" nullable:"false"`
	Total int32   `json:"total"`
}

type GroupMembership struct {
	CreatedAt time.Time `json:"created_at"`
	GroupID   uuid.UUID `json:"group_id"`
	ID        uuid.UUID `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uuid.UUID `json:"user_id"`
}

type GroupMembershipListResponse struct {
	Rows  []GroupMembership `json:"rows" nullable:"false"`
	Total int32             `json:"total"`
}

type Location struct {
	BackgroundAssetID *uuid.UUID  `json:"background_asset_id,omitempty" nullable:"true"`
	CreatedAt         time.Time   `json:"created_at"`
	Description       string      `json:"description"`
	Enabled           bool        `json:"enabled"`
	GroupIDs          []uuid.UUID `json:"group_ids" nullable:"false"`
	ID                uuid.UUID   `json:"id"`
	LogoAssetID       *uuid.UUID  `json:"logo_asset_id,omitempty" nullable:"true"`
	Name              string      `json:"name"`
	Notes             bool        `json:"notes"`
	Photo             bool        `json:"photo"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type LocationListResponse struct {
	Rows  []Location `json:"rows" nullable:"false"`
	Total int32      `json:"total"`
}

type LocationWriteRequest struct {
	BackgroundAssetID *uuid.UUID  `json:"background_asset_id,omitempty" nullable:"true"`
	Description       string      `json:"description"`
	Enabled           bool        `json:"enabled"`
	GroupIDs          []uuid.UUID `json:"group_ids" nullable:"false"`
	LogoAssetID       *uuid.UUID  `json:"logo_asset_id,omitempty" nullable:"true"`
	Name              string      `json:"name"`
	Notes             bool        `json:"notes"`
	Photo             bool        `json:"photo"`
}

type PermissionAction string

type PermissionGrant struct {
	Action     PermissionAction   `json:"action"`
	AssetType  *AssetType         `json:"asset_type,omitempty"`
	LocationID *uuid.UUID         `json:"location_id,omitempty" nullable:"true"`
	Resource   PermissionResource `json:"resource"`
}

type PermissionResource string

type PermissionSubjectKind string

//nolint:errname // Problem is the established v1 wire schema name.
type Problem struct {
	Code        string        `json:"code"`
	Detail      string        `json:"detail"`
	FieldErrors *[]FieldError `json:"field_errors,omitempty"`
	Status      int32         `json:"status"`
	Title       string        `json:"title"`
	Type        string        `json:"type"`
}

type Source string

type User struct {
	Access      []PermissionGrant `json:"access" nullable:"false"`
	Admin       bool              `json:"admin"`
	CreatedAt   time.Time         `json:"created_at"`
	Department  string            `json:"department"`
	DisplayName string            `json:"display_name"`
	ID          uuid.UUID         `json:"id"`
	Source      Source            `json:"source"`
	UpdatedAt   time.Time         `json:"updated_at"`
	UPN         string            `json:"upn"`
}

type UserAccessWriteRequest struct {
	Access []PermissionGrant `json:"access" nullable:"false"`
	Admin  bool              `json:"admin"`
}

type UserListResponse struct {
	Rows  []User `json:"rows" nullable:"false"`
	Total int32  `json:"total"`
}

func enumSchema(registry huma.Registry, name string, values ...any) *huma.Schema {
	registry.Map()[name] = &huma.Schema{Type: "string", Enum: values}
	return &huma.Schema{Ref: "#/components/schemas/" + name}
}

func (AssetType) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "AssetType", "asset", "photo")
}

func (CheckinDirection) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "CheckinDirection", "check_in", "check_out")
}

func (PermissionAction) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "PermissionAction", "read", "create", "write", "delete")
}

func (PermissionResource) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "PermissionResource", "users", "groups", "locations", "checkins", "assets", "api_keys")
}

func (PermissionSubjectKind) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "PermissionSubjectKind", "user", "api_key")
}

func (Source) Schema(registry huma.Registry) *huma.Schema {
	return enumSchema(registry, "Source", "entra", "local")
}

func (PermissionGrant) TransformSchema(_ huma.Registry, schema *huma.Schema) *huma.Schema {
	schema.Properties["asset_type"] = &huma.Schema{AnyOf: []*huma.Schema{
		{Ref: "#/components/schemas/AssetType"}, {Type: "null"},
	}}
	return schema
}
