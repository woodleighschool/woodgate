// Package v0 is the removable adapter for companion builds deployed before the
// versioned Station protocol existed.
package v0

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/httpx"
	"github.com/woodleighschool/woodgate/internal/station"
)

const (
	maxBodyBytes  = 12 << 20
	maxPhotoBytes = 10 << 20
)

type locationProvider interface {
	GetStationLocation(context.Context, int64) (*station.LocationConfiguration, error)
}

type peopleProvider interface {
	ListStationPeople(context.Context, int64) ([]station.Person, error)
}

type checkinSubmitter interface {
	SubmitStationCheckin(context.Context, station.CheckinSubmission) (*station.CheckinReceipt, error)
}

type legacyAssetDelivery interface {
	DeliverLegacyStationAsset(http.ResponseWriter, *http.Request, int64, int64) error
}

// Dependencies are the new Station application services used by the v0 translator.
type Dependencies struct {
	Locations locationProvider
	People    peopleProvider
	Checkins  checkinSubmitter
	Assets    legacyAssetDelivery
}

// Server translates the exact API used by unupgraded companion builds.
type Server struct {
	store  repository
	deps   Dependencies
	logger *slog.Logger
}

// NewServer returns the temporary protocol-v0 adapter.
func NewServer(pool *pgxpool.Pool, deps Dependencies, logger *slog.Logger) *Server {
	return newServer(newStore(pool), deps, logger)
}

func newServer(store repository, deps Dependencies, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{store: store, deps: deps, logger: logger}
}

// RegisterRoutes mounts only the five endpoint families consumed by v0.
func (s *Server) RegisterRoutes(router chi.Router) {
	router.Get("/auth/me", s.me)
	router.Get("/api/v1/locations", s.listLocations)
	router.Get("/api/v1/locations/{id}", s.getLocation)
	router.Get("/api/v1/users", s.listPeople)
	router.Post("/api/v1/checkins", s.checkin)
	router.Get("/api/v1/assets/{id}/content", s.assetContent)
}

type legacyPrincipal struct {
	Type string    `json:"type"`
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type legacyGrant struct {
	Resource   string    `json:"resource"`
	Action     string    `json:"action"`
	LocationID uuid.UUID `json:"location_id"`
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	item, keyID, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	locationID, err := s.store.ensureLegacyID(r.Context(), "location", item.LocationID)
	if err != nil {
		s.fail(w, r, "me", err)
		return
	}
	httpx.Write(w, http.StatusOK, struct {
		Principal legacyPrincipal `json:"principal"`
		Admin     bool            `json:"admin"`
		Access    []legacyGrant   `json:"access"`
	}{
		Principal: legacyPrincipal{Type: "api_key", ID: keyID, Name: item.Name},
		Access:    []legacyGrant{{Resource: "checkins", Action: "create", LocationID: locationID}},
	})
}

type legacyLocation struct {
	ID                uuid.UUID   `json:"id"`
	Name              string      `json:"name"`
	Enabled           bool        `json:"enabled"`
	Notes             bool        `json:"notes"`
	Photo             bool        `json:"photo"`
	BackgroundAssetID *uuid.UUID  `json:"background_asset_id"`
	LogoAssetID       *uuid.UUID  `json:"logo_asset_id"`
	GroupIDs          []uuid.UUID `json:"group_ids"`
}

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	location, err := s.location(r.Context(), item.LocationID)
	if err != nil {
		s.writeError(w, r, "list locations", err)
		return
	}
	httpx.Write(w, http.StatusOK, legacyList[legacyLocation]{Rows: []legacyLocation{location}, Total: 1})
}

func (s *Server) getLocation(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	requested, err := uuid.Parse(httpx.PathParam(r, "id"))
	if err != nil {
		s.problem(w, http.StatusBadRequest, "location id is invalid")
		return
	}
	id, err := s.store.objectID(r.Context(), "location", requested)
	if err != nil || id != item.LocationID {
		s.problem(w, http.StatusNotFound, "location not found")
		return
	}
	location, err := s.location(r.Context(), id)
	if err != nil {
		s.writeError(w, r, "get location", err)
		return
	}
	httpx.Write(w, http.StatusOK, location)
}

func (s *Server) location(ctx context.Context, id int64) (legacyLocation, error) {
	location, err := s.deps.Locations.GetStationLocation(ctx, id)
	if err != nil {
		return legacyLocation{}, err
	}
	legacyID, err := s.store.ensureLegacyID(ctx, "location", id)
	if err != nil {
		return legacyLocation{}, err
	}
	backgroundID, err := s.optionalLegacyID(ctx, "asset", location.BackgroundObjectID)
	if err != nil {
		return legacyLocation{}, err
	}
	logoID, err := s.optionalLegacyID(ctx, "asset", location.LogoObjectID)
	if err != nil {
		return legacyLocation{}, err
	}
	return legacyLocation{ID: legacyID, Name: location.Name, Enabled: location.Enabled, Notes: location.Notes,
		Photo: location.Photo, BackgroundAssetID: backgroundID, LogoAssetID: logoID, GroupIDs: []uuid.UUID{}}, nil
}

func (s *Server) optionalLegacyID(ctx context.Context, kind string, id *int64) (*uuid.UUID, error) {
	if id == nil {
		return nil, nil
	}
	legacyID, err := s.store.ensureLegacyID(ctx, kind, *id)
	if err != nil {
		return nil, err
	}
	return &legacyID, nil
}

type legacyPerson struct {
	ID          uuid.UUID `json:"id"`
	UPN         string    `json:"upn"`
	DisplayName string    `json:"display_name"`
}

type legacyList[T any] struct {
	Rows  []T `json:"rows"`
	Total int `json:"total"`
}

func (s *Server) listPeople(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	locationID, err := uuid.Parse(r.URL.Query().Get("location_id"))
	if err != nil {
		s.problem(w, http.StatusBadRequest, "location_id is invalid")
		return
	}
	currentLocationID, err := s.store.objectID(r.Context(), "location", locationID)
	if err != nil || currentLocationID != item.LocationID {
		s.problem(w, http.StatusForbidden, "forbidden")
		return
	}
	people, err := s.deps.People.ListStationPeople(r.Context(), item.LocationID)
	if err != nil {
		s.writeError(w, r, "list people", err)
		return
	}
	rows := make([]legacyPerson, 0, len(people))
	for _, person := range people {
		id, mapErr := s.store.ensureLegacyID(r.Context(), "user", person.ID)
		if mapErr != nil {
			s.fail(w, r, "map person", mapErr)
			return
		}
		rows = append(rows, legacyPerson{ID: id, UPN: person.Email, DisplayName: person.Name})
	}
	total := len(rows)
	offset, limit := page(r, total)
	httpx.Write(w, http.StatusOK, legacyList[legacyPerson]{Rows: rows[offset:min(offset+limit, total)], Total: total})
}

type legacyCheckin struct {
	ID         uuid.UUID         `json:"id"`
	UserID     uuid.UUID         `json:"user_id"`
	LocationID uuid.UUID         `json:"location_id"`
	Direction  station.Direction `json:"direction"`
}

func (s *Server) checkin(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	submission, legacyUserID, legacyLocationID, err := s.parseCheckin(w, r, item)
	if err != nil {
		s.problem(w, http.StatusBadRequest, err.Error())
		return
	}
	receipt, err := s.deps.Checkins.SubmitStationCheckin(r.Context(), submission)
	if err != nil {
		s.writeError(w, r, "create check-in", err)
		return
	}
	id, err := s.store.ensureLegacyID(r.Context(), "checkin", receipt.ID)
	if err != nil {
		s.fail(w, r, "map check-in", err)
		return
	}
	httpx.Write(w, http.StatusCreated, legacyCheckin{ID: id, UserID: legacyUserID, LocationID: legacyLocationID, Direction: receipt.Direction})
}

func (s *Server) parseCheckin(w http.ResponseWriter, r *http.Request, item *station.Station) (station.CheckinSubmission, uuid.UUID, uuid.UUID, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil { //nolint:gosec // The reader is bounded above.
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("invalid check-in form")
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	legacyUserID, err := uuid.Parse(strings.TrimSpace(r.FormValue("user_id")))
	if err != nil {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("user_id is invalid")
	}
	userID, err := s.store.objectID(r.Context(), "user", legacyUserID)
	if err != nil {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("user_id is invalid")
	}
	legacyLocationID, err := uuid.Parse(strings.TrimSpace(r.FormValue("location_id")))
	if err != nil {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("location_id is invalid")
	}
	locationID, err := s.store.objectID(r.Context(), "location", legacyLocationID)
	if err != nil || locationID != item.LocationID {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("location_id is invalid")
	}
	direction := station.Direction(strings.TrimSpace(r.FormValue("direction")))
	if direction != station.DirectionIn && direction != station.DirectionOut {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, errors.New("direction is invalid")
	}
	photo, contentType, err := readPhoto(r.MultipartForm)
	if err != nil {
		return station.CheckinSubmission{}, uuid.Nil, uuid.Nil, err
	}
	return station.CheckinSubmission{StationID: item.ID, LocationID: item.LocationID, PersonID: userID,
		Direction: direction, Notes: strings.TrimSpace(r.FormValue("notes")), Photo: photo, ContentType: contentType}, legacyUserID, legacyLocationID, nil
}

func readPhoto(form *multipart.Form) ([]byte, string, error) {
	if form == nil || len(form.File["photo"]) == 0 {
		return nil, "", nil
	}
	if len(form.File["photo"]) != 1 {
		return nil, "", errors.New("photo is invalid")
	}
	file, err := form.File["photo"][0].Open()
	if err != nil {
		return nil, "", errors.New("photo is invalid")
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
	if err != nil || len(content) > maxPhotoBytes {
		return nil, "", errors.New("photo is too large")
	}
	if len(content) == 0 || http.DetectContentType(content) != "image/jpeg" {
		return nil, "", errors.New("photo must be a JPEG image")
	}
	return content, "image/jpeg", nil
}

func (s *Server) assetContent(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	legacyID, err := uuid.Parse(httpx.PathParam(r, "id"))
	if err != nil {
		s.problem(w, http.StatusBadRequest, "asset id is invalid")
		return
	}
	id, err := s.store.objectID(r.Context(), "asset", legacyID)
	if err != nil {
		s.problem(w, http.StatusNotFound, "asset not found")
		return
	}
	if err := s.deps.Assets.DeliverLegacyStationAsset(w, r, item.LocationID, id); err != nil {
		s.writeError(w, r, "deliver media", err)
	}
}

func (s *Server) authenticate(w http.ResponseWriter, r *http.Request) (*station.Station, uuid.UUID, bool) {
	item, keyID, err := s.store.authenticate(r.Context(), r.Header.Get("X-Api-Key"))
	if errors.Is(err, station.ErrUnauthorized) {
		s.problem(w, http.StatusUnauthorized, "unauthorized")
		return nil, uuid.Nil, false
	}
	if err != nil {
		s.fail(w, r, "authenticate", err)
		return nil, uuid.Nil, false
	}
	return item, keyID, true
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	switch {
	case errors.Is(err, fault.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		s.problem(w, http.StatusNotFound, "not found")
	case errors.Is(err, fault.ErrInvalidInput):
		s.problem(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), fault.ErrInvalidInput.Error()+": "))
	case errors.Is(err, fault.ErrConflict):
		s.problem(w, http.StatusConflict, strings.TrimPrefix(err.Error(), fault.ErrConflict.Error()+": "))
	default:
		s.fail(w, r, operation, err)
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, operation string, err error) {
	s.logger.WarnContext(r.Context(), "legacy Station request failed", "operation", operation, "path", r.URL.Path, "err", err)
	s.problem(w, http.StatusInternalServerError, "internal error")
}

func (*Server) problem(w http.ResponseWriter, status int, detail string) {
	httpx.Write(w, status, struct {
		Status int    `json:"status"`
		Detail string `json:"detail"`
		Code   string `json:"code"`
	}{Status: status, Detail: detail, Code: "station_v0_error"})
}

func page(r *http.Request, total int) (int, int) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset = max(offset, 0)
	offset = min(offset, total)
	if limit < 1 {
		limit = total
	}
	return offset, limit
}
