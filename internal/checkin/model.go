package checkin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/validation"
)

// Direction identifies whether a person is arriving or leaving.
type Direction string

const (
	DirectionIn  Direction = "check_in"
	DirectionOut Direction = "check_out"
)

const (
	BackgroundObjectPrefix = "checkin/backgrounds"
	LogoObjectPrefix       = "checkin/logos"
	PhotoObjectPrefix      = "checkin/photos"
)

// AttachmentFile describes bytes owned by a check-in resource.
type AttachmentFile struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
}

// Location controls one Station check-in workflow.
type Location struct {
	ID                 int64           `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Enabled            bool            `json:"enabled"`
	Notes              bool            `json:"notes"`
	Photo              bool            `json:"photo"`
	BackgroundObjectID *int64          `json:"background_object_id,omitempty"`
	BackgroundFile     *AttachmentFile `json:"background_file,omitempty"`
	BackgroundURL      string          `json:"background_url,omitempty"`
	LogoObjectID       *int64          `json:"logo_object_id,omitempty"`
	LogoFile           *AttachmentFile `json:"logo_file,omitempty"`
	LogoURL            string          `json:"logo_url,omitempty"`
	GroupIDs           []int64         `json:"group_ids" nullable:"false"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// LocationListParams filters the paginated location list.
type LocationListParams struct {
	ListParams listing.Params
	Enabled    *bool
}

// LocationMutation replaces the writable location configuration.
type LocationMutation struct {
	Name               string  `json:"name" validate:"required,notblank"`
	Description        string  `json:"description"`
	Enabled            bool    `json:"enabled"`
	Notes              bool    `json:"notes"`
	Photo              bool    `json:"photo"`
	BackgroundObjectID *int64  `json:"background_object_id,omitempty" minimum:"1"`
	LogoObjectID       *int64  `json:"logo_object_id,omitempty" minimum:"1"`
	GroupIDs           []int64 `json:"group_ids" nullable:"false"`
}

// Checkin is an immutable arrival or departure event.
type Checkin struct {
	ID              int64           `json:"id"`
	UserID          int64           `json:"user_id"`
	UserName        string          `json:"user_name"`
	Department      string          `json:"department"`
	LocationID      int64           `json:"location_id"`
	LocationName    string          `json:"location_name"`
	Direction       Direction       `json:"direction"`
	Notes           string          `json:"notes"`
	PhotoObjectID   *int64          `json:"photo_object_id,omitempty"`
	PhotoFile       *AttachmentFile `json:"photo_file,omitempty"`
	PhotoURL        string          `json:"photo_url,omitempty"`
	StationID       *int64          `json:"station_id,omitempty"`
	CreatedByUserID *int64          `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CheckinListParams filters the paginated check-in list.
type CheckinListParams struct {
	ListParams    listing.Params
	LocationID    int64     `validate:"gte=0"`
	UserID        int64     `validate:"gte=0"`
	Direction     Direction `validate:"omitempty,oneof=check_in check_out"`
	Department    string
	CreatedFrom   *time.Time
	CreatedBefore *time.Time
}

// CheckinCreate contains a human-authenticated check-in event.
type CheckinCreate struct {
	UserID     int64     `json:"user_id"     minimum:"1" validate:"gte=1"`
	LocationID int64     `json:"location_id" minimum:"1" validate:"gte=1"`
	Direction  Direction `json:"direction"   enum:"check_in,check_out" validate:"oneof=check_in check_out"`
	Notes      string    `json:"notes"`
}

func (mutation *LocationMutation) normalize() {
	mutation.Name = strings.TrimSpace(mutation.Name)
	mutation.Description = strings.TrimSpace(mutation.Description)
	mutation.GroupIDs = uniquePositiveIDs(mutation.GroupIDs)
}

func (mutation *LocationMutation) validate() error {
	if err := validation.Struct(mutation); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	for _, id := range append(append([]int64{}, mutation.GroupIDs...), pointerIDs(mutation.BackgroundObjectID, mutation.LogoObjectID)...) {
		if id < 1 {
			return fmt.Errorf("%w: identifiers must be positive", fault.ErrInvalidInput)
		}
	}
	return nil
}

func backgroundURL(objectID *int64) string {
	return attachmentURL("/api/locations/backgrounds", objectID)
}

func logoURL(objectID *int64) string {
	return attachmentURL("/api/locations/logos", objectID)
}

func photoURL(checkinID int64, objectID *int64) string {
	if objectID == nil || *objectID <= 0 {
		return ""
	}
	return "/api/checkins/" + strconv.FormatInt(checkinID, 10) + "/photo"
}

func attachmentURL(basePath string, objectID *int64) string {
	if objectID == nil || *objectID <= 0 {
		return ""
	}
	return basePath + "/" + strconv.FormatInt(*objectID, 10) + "/content"
}

func (params *CheckinListParams) normalize() {
	params.ListParams = listing.Normalize(params.ListParams)
	params.Department = strings.TrimSpace(params.Department)
}

func (params *CheckinListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	if params.CreatedFrom != nil && params.CreatedBefore != nil && !params.CreatedFrom.Before(*params.CreatedBefore) {
		return fmt.Errorf("%w: created_from must be before created_before", fault.ErrInvalidInput)
	}
	return listing.Validate(params.ListParams)
}

func (create *CheckinCreate) normalize() { create.Notes = strings.TrimSpace(create.Notes) }

func (create *CheckinCreate) validate() error {
	if err := validation.Struct(create); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func uniquePositiveIDs(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func pointerIDs(values ...*int64) []int64 {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}
