package domain

import (
	"time"

	"github.com/google/uuid"
)

type PrincipalSource string

const (
	PrincipalSourceLocal PrincipalSource = "local"
	PrincipalSourceEntra PrincipalSource = "entra"
)

type PermissionSubjectKind string

const (
	PermissionSubjectKindUser   PermissionSubjectKind = "user"
	PermissionSubjectKindAPIKey PermissionSubjectKind = "api_key"
)

type PermissionResource string

const (
	PermissionResourceUsers     PermissionResource = "users"
	PermissionResourceGroups    PermissionResource = "groups"
	PermissionResourceLocations PermissionResource = "locations"
	PermissionResourceCheckins  PermissionResource = "checkins"
	PermissionResourceAssets    PermissionResource = "assets"
	PermissionResourceAPIKeys   PermissionResource = "api_keys"
)

type PermissionAction string

const (
	PermissionActionRead   PermissionAction = "read"
	PermissionActionCreate PermissionAction = "create"
	PermissionActionWrite  PermissionAction = "write"
	PermissionActionDelete PermissionAction = "delete"
)

type CheckinDirection string

const (
	CheckinDirectionIn  CheckinDirection = "check_in"
	CheckinDirectionOut CheckinDirection = "check_out"
)

type AssetType string

const (
	AssetTypeAsset AssetType = "asset"
	AssetTypePhoto AssetType = "photo"
)

type User struct {
	ID          uuid.UUID
	UPN         string
	DisplayName string
	Department  string
	Source      PrincipalSource
	Admin       bool
	Access      []PermissionGrant
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Group struct {
	ID          uuid.UUID
	Name        string
	Description string
	MemberCount int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GroupMembership struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Asset struct {
	ID            uuid.UUID
	Name          *string
	Type          AssetType
	ContentType   string
	FileExtension string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Location struct {
	ID                uuid.UUID
	Name              string
	Description       string
	Enabled           bool
	Notes             bool
	Photo             bool
	BackgroundAssetID *uuid.UUID
	LogoAssetID       *uuid.UUID
	GroupIDs          []uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Checkin struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	LocationID    uuid.UUID
	Direction     CheckinDirection
	Notes         string
	AssetID       *uuid.UUID
	CreatedByKind PermissionSubjectKind
	CreatedByID   uuid.UUID
	CreatedAt     time.Time
}

type APIKey struct {
	ID         uuid.UUID
	Name       string
	KeyPrefix  string
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
	Admin      bool
	Access     []PermissionGrant
	CreatedAt  time.Time
}

type PermissionGrant struct {
	Resource   PermissionResource
	Action     PermissionAction
	LocationID *uuid.UUID
	AssetType  *AssetType
}

type PrincipalRole struct {
	PrincipalKind PermissionSubjectKind
	PrincipalID   uuid.UUID
	Role          string
}

type PrincipalPermissionGrant struct {
	PrincipalKind PermissionSubjectKind
	PrincipalID   uuid.UUID
	Grant         PermissionGrant
}
