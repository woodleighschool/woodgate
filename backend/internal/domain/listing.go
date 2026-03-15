package domain

import (
	"time"

	"github.com/google/uuid"
)

type ListOptions struct {
	Limit  int32
	Offset int32
	Search string
	Sort   string
	Order  string
}

type UserListOptions struct {
	ListOptions

	LocationID *uuid.UUID
}

type GroupListOptions struct {
	ListOptions
}

type GroupMembershipListOptions struct {
	ListOptions

	GroupID *uuid.UUID
	UserID  *uuid.UUID
}

type AssetListOptions struct {
	ListOptions

	Types []AssetType
}

type LocationListOptions struct {
	ListOptions

	Enabled *bool
}

type CheckinListOptions struct {
	ListOptions

	LocationID  *uuid.UUID
	UserID      *uuid.UUID
	Direction   *CheckinDirection
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

type APIKeyListOptions struct {
	ListOptions
}
