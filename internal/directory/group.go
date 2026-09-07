package directory

import (
	"time"

	"github.com/woodleighschool/woodgate/internal/listing"
)

// Group is one directory group.
type Group struct {
	ID           int64         `json:"id"`
	Source       Source        `json:"source"`
	ExternalID   string        `json:"external_id"`
	DisplayName  string        `json:"display_name"`
	MailNickname string        `json:"mail_nickname,omitempty"`
	MemberCount  int32         `json:"member_count"`
	Roles        []RoleSummary `json:"roles" db:"-"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// GroupSummary is the small directory group projection attached to another resource.
type GroupSummary struct {
	ID           int64  `json:"id"`
	Source       Source `json:"source"`
	DisplayName  string `json:"display_name"`
	MailNickname string `json:"mail_nickname,omitempty"`
}

// GroupMutation replaces application-owned fields on a directory group.
type GroupMutation struct {
	RoleIDs []int64 `json:"role_ids"`
}

func (mutation *GroupMutation) normalize() {
	mutation.RoleIDs = normalizeRoleIDs(mutation.RoleIDs)
}

func (mutation *GroupMutation) validate() error {
	return validateRoleIDs(mutation.RoleIDs)
}

// GroupListParams filters paginated group lists.
type GroupListParams struct {
	ListParams listing.Params

	Values []string
}
