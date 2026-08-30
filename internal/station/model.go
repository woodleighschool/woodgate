package station

import (
	"fmt"
	"strings"
	"time"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/validation"
)

// Station is the administrative view of one location-bound companion app.
// Its secret hash is never part of this model.
type Station struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	LocationID      int64      `json:"location_id"`
	Enabled         bool       `json:"enabled"`
	SecretPrefix    string     `json:"secret_prefix"`
	Online          bool       `json:"online"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	AppBuild        string     `json:"app_build,omitempty"`
	ProtocolVersion *int       `json:"protocol_version,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// StationMutation is the complete administrative write surface for a Station.
type StationMutation struct {
	Name       string `json:"name"        validate:"required,notblank"`
	LocationID int64  `json:"location_id" validate:"gte=1"`
	Enabled    bool   `json:"enabled"`
}

// StationListParams filters the paginated administrative station list.
type StationListParams struct {
	ListParams listing.Params

	LocationID int64 `validate:"gte=0"`
	Enabled    *bool
}

// StationSecret is returned only when a station is created or its secret is rotated.
type StationSecret struct {
	Station Station `json:"station"`
	Secret  string  `json:"secret"`
}

// Direction is the check-in action submitted by a station.
type Direction string

const (
	DirectionIn  Direction = "check_in"
	DirectionOut Direction = "check_out"
)

// LocationConfiguration is the station-facing projection of its bound location.
type LocationConfiguration struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Enabled            bool      `json:"enabled"`
	Notes              bool      `json:"notes"`
	Photo              bool      `json:"photo"`
	BackgroundObjectID *int64    `json:"background_object_id,omitempty"`
	LogoObjectID       *int64    `json:"logo_object_id,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Configuration is the complete location-derived configuration for a station.
type Configuration struct {
	StationID   int64                 `json:"station_id"`
	StationName string                `json:"station_name"`
	Location    LocationConfiguration `json:"location"`
}

// Person is the station-facing directory projection used for local search.
type Person struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// People is the station-facing directory response.
type People struct {
	Items []Person `json:"items"`
	Count int      `json:"count"`
}

// CheckinSubmission is a station-authenticated check-in request.
type CheckinSubmission struct {
	StationID   int64
	LocationID  int64
	PersonID    int64
	Direction   Direction
	Notes       string
	Photo       []byte
	ContentType string
}

// CheckinReceipt identifies a completed station check-in.
type CheckinReceipt struct {
	ID         int64     `json:"id"`
	PersonID   int64     `json:"person_id"`
	LocationID int64     `json:"location_id"`
	Direction  Direction `json:"direction"`
}

func (params *StationListParams) normalize() {
	params.ListParams = listing.Normalize(params.ListParams)
}

func (params *StationListParams) validate() error {
	if err := validation.Struct(params); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func (mutation *StationMutation) normalize() {
	mutation.Name = strings.TrimSpace(mutation.Name)
}

func (mutation *StationMutation) validate() error {
	if err := validation.Struct(mutation); err != nil {
		return fmt.Errorf("%w: %w", fault.ErrInvalidInput, err)
	}
	return nil
}

func parseDirection(value string) (Direction, error) {
	switch Direction(strings.TrimSpace(value)) {
	case DirectionIn:
		return DirectionIn, nil
	case DirectionOut:
		return DirectionOut, nil
	default:
		return "", fmt.Errorf("%w: unsupported check-in direction %q", fault.ErrInvalidInput, value)
	}
}
