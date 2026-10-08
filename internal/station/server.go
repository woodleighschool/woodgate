package station

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/httpx"
)

const (
	deviceBasePath      = "/api/station/v1"
	maxCheckinBodyBytes = 12 << 20
	maxPhotoBytes       = 10 << 20
)

// PeopleProvider resolves people eligible for a station's bound location.
type PeopleProvider interface {
	ListStationPeople(context.Context, int64) ([]Person, error)
}

// CheckinSubmitter persists an already station-scoped check-in.
type CheckinSubmitter interface {
	SubmitStationCheckin(context.Context, CheckinSubmission) (*CheckinReceipt, error)
}

// BrandingDelivery sends location-owned artwork to a Station.
type BrandingDelivery interface {
	DeliverStationBackground(http.ResponseWriter, *http.Request, int64) error
	DeliverStationLogo(http.ResponseWriter, *http.Request, int64) error
}

// Dependencies are application-owned adapters used by the Station v1 surface.
type Dependencies struct {
	Locations LocationProvider
	People    PeopleProvider
	Checkins  CheckinSubmitter
	Branding  BrandingDelivery
}

// Server owns Station v1 HTTP authentication and its auxiliary control plane.
type Server struct {
	store  deviceStore
	deps   Dependencies
	hub    *Hub
	build  string
	logger *slog.Logger
}

// NewServer returns the Station v1 protocol server.
func NewServer(store *Store, deps Dependencies, build string, logger *slog.Logger) (*Server, error) {
	return newServer(store, deps, build, logger)
}

func newServer(
	store deviceStore,
	deps Dependencies,
	build string,
	logger *slog.Logger,
) (*Server, error) {
	if !validBuild(build) {
		return nil, fmt.Errorf("invalid server build %q", build)
	}
	if logger == nil {
		logger = slog.Default()
	}
	server := &Server{store: store, deps: deps, build: build, logger: logger}
	server.hub = newHub(store, build, logger)
	return server, nil
}

// RegisterRoutes mounts ordinary HTTP and WebSocket Station v1 routes.
// Mount these routers outside human session and RBAC middleware.
func (s *Server) RegisterRoutes(ordinary chi.Router, webSockets chi.Router) {
	ordinary.Get(deviceBasePath+"/configuration", s.configuration)
	ordinary.Get(deviceBasePath+"/people", s.people)
	ordinary.Post(deviceBasePath+"/checkins", s.checkin)
	ordinary.Get(deviceBasePath+"/configuration/background", s.background)
	ordinary.Get(deviceBasePath+"/configuration/logo", s.logo)
	webSockets.Get(deviceBasePath+"/connect", s.connect)
}

// ConfigurationChanged queues a refresh and updates the connection's location binding.
func (s *Server) ConfigurationChanged(stationID int64, locationID int64) {
	s.hub.configurationChanged(stationID, locationID)
}

// ConfigurationChangedForLocation queues a refresh for Stations bound to a location.
func (s *Server) ConfigurationChangedForLocation(locationID int64) {
	s.hub.configurationChangedForLocation(locationID)
}

// Disconnect closes one Station control connection.
func (s *Server) Disconnect(stationID int64, reason string) {
	s.hub.disconnect(stationID, reason)
}

// Close disconnects every Station control connection.
func (s *Server) Close() {
	s.hub.Close()
}

func (s *Server) configuration(w http.ResponseWriter, r *http.Request) {
	station, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if s.deps.Locations == nil {
		s.fail(w, r, "configuration", errors.New("station location provider is required"))
		return
	}
	location, err := s.deps.Locations.GetStationLocation(r.Context(), station.LocationID)
	if err != nil {
		s.writeDeviceError(w, r, "configuration", err)
		return
	}
	httpx.Write(w, http.StatusOK, Configuration{
		StationID:   station.ID,
		StationName: station.Name,
		Location:    *location,
	})
}

func (s *Server) people(w http.ResponseWriter, r *http.Request) {
	station, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if s.deps.People == nil {
		s.fail(w, r, "people", errors.New("station people provider is required"))
		return
	}
	people, err := s.deps.People.ListStationPeople(r.Context(), station.LocationID)
	if err != nil {
		s.writeDeviceError(w, r, "people", err)
		return
	}
	if people == nil {
		people = []Person{}
	}
	httpx.Write(w, http.StatusOK, People{Items: people, Count: len(people)})
}

func (s *Server) checkin(w http.ResponseWriter, r *http.Request) {
	station, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if s.deps.Locations == nil || s.deps.Checkins == nil {
		s.fail(w, r, "checkin", errors.New("station check-in dependencies are required"))
		return
	}
	location, err := s.deps.Locations.GetStationLocation(r.Context(), station.LocationID)
	if err != nil {
		s.writeDeviceError(w, r, "checkin", err)
		return
	}
	if !location.Enabled {
		httpx.WriteError(w, http.StatusConflict, "location is disabled")
		return
	}
	submission, err := parseCheckin(w, r, station, location)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	receipt, err := s.deps.Checkins.SubmitStationCheckin(r.Context(), submission)
	if err != nil {
		s.writeDeviceError(w, r, "checkin", err)
		return
	}
	httpx.Write(w, http.StatusCreated, receipt)
}

func (s *Server) background(w http.ResponseWriter, r *http.Request) {
	station, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if s.deps.Branding == nil {
		s.fail(w, r, "background", errors.New("station branding delivery is required"))
		return
	}
	if err := s.deps.Branding.DeliverStationBackground(w, r, station.LocationID); err != nil {
		s.writeDeviceError(w, r, "background", err)
	}
}

func (s *Server) logo(w http.ResponseWriter, r *http.Request) {
	station, _, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	if s.deps.Branding == nil {
		s.fail(w, r, "logo", errors.New("station branding delivery is required"))
		return
	}
	if err := s.deps.Branding.DeliverStationLogo(w, r, station.LocationID); err != nil {
		s.writeDeviceError(w, r, "logo", err)
	}
}

func (s *Server) connect(w http.ResponseWriter, r *http.Request) {
	station, secret, ok := s.authenticate(w, r)
	if !ok {
		return
	}
	protocols := offeredSubprotocols(r.Header.Values("Sec-WebSocket-Protocol"))
	buildHeaders := r.Header.Values(AppBuildHeader)
	protocolVersion := incompatibleProtocolVersion(protocols)
	appBuild := compatibleBuild(buildHeaders)

	w.Header().Set(ServerBuildHeader, s.build)
	if len(protocols) != 1 || protocols[0] != Subprotocol || len(buildHeaders) != 1 || appBuild == "" {
		w.Header().Set(ProtocolHeader, Subprotocol)
		if err := s.store.ObserveRejectedClient(r.Context(), station.ID, secret, protocolVersion, appBuild); err != nil &&
			!errors.Is(err, ErrSessionInvalid) {
			s.log(r, "negotiate", err)
		}
		http.Error(w, "incompatible Station protocol", http.StatusUpgradeRequired)
		return
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{Subprotocol},
	})
	if err != nil {
		s.log(r, "connect", err)
		return
	}
	if err := s.hub.serve(r.Context(), ws, station, secret, ProtocolVersion, appBuild); err != nil && !isExpectedClose(err) {
		s.log(r, "connect", err)
		_ = ws.Close(websocket.StatusInternalError, "control error")
		return
	}
	_ = ws.Close(websocket.StatusNormalClosure, "")
}

func (s *Server) authenticate(
	w http.ResponseWriter,
	r *http.Request,
) (*Station, string, bool) {
	secret, ok := httpx.BearerToken(r.Header.Get("Authorization"))
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return nil, "", false
	}
	station, err := s.store.authenticate(r.Context(), secret)
	if errors.Is(err, ErrUnauthorized) {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return nil, "", false
	}
	if err != nil {
		s.fail(w, r, "authenticate", err)
		return nil, "", false
	}
	return station, secret, true
}

func (s *Server) writeDeviceError(
	w http.ResponseWriter,
	r *http.Request,
	operation string,
	err error,
) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, fault.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, fault.ErrInvalidInput):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, fault.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	default:
		s.fail(w, r, operation, err)
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, operation string, err error) {
	s.log(r, operation, err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal error")
}

func (s *Server) log(r *http.Request, operation string, err error) {
	s.logger.WarnContext(r.Context(), "station protocol request failed",
		"operation", operation,
		"path", r.URL.Path,
		"err", err,
	)
}

func parseCheckin(
	w http.ResponseWriter,
	r *http.Request,
	station *Station,
	location *LocationConfiguration,
) (CheckinSubmission, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCheckinBodyBytes)
	// MaxBytesReader bounds the request before the standard multipart parser allocates form storage.
	//nolint:gosec // G120 cannot infer the wrapper above.
	if err := r.ParseMultipartForm(maxCheckinBodyBytes); err != nil {
		return CheckinSubmission{}, fmt.Errorf("invalid check-in form: %w", err)
	}
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}

	personID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("person_id")), 10, 64)
	if err != nil || personID < 1 {
		return CheckinSubmission{}, errors.New("person_id is invalid")
	}
	direction, err := parseDirection(r.FormValue("direction"))
	if err != nil {
		return CheckinSubmission{}, errors.New("direction is invalid")
	}
	notes := strings.TrimSpace(r.FormValue("notes"))
	if !location.Notes && notes != "" {
		return CheckinSubmission{}, errors.New("notes are disabled for this location")
	}

	photo, contentType, err := readPhoto(r.MultipartForm)
	if err != nil {
		return CheckinSubmission{}, err
	}
	if location.Photo && len(photo) == 0 {
		return CheckinSubmission{}, errors.New("photo is required for this location")
	}
	return CheckinSubmission{
		StationID:   station.ID,
		LocationID:  station.LocationID,
		PersonID:    personID,
		Direction:   direction,
		Notes:       notes,
		Photo:       photo,
		ContentType: contentType,
	}, nil
}

func readPhoto(form *multipart.Form) ([]byte, string, error) {
	if form == nil || len(form.File["photo"]) == 0 {
		return nil, "", nil
	}
	if len(form.File["photo"]) != 1 {
		return nil, "", errors.New("photo must contain one file")
	}
	header := form.File["photo"][0]
	file, err := header.Open()
	if err != nil {
		return nil, "", errors.New("photo is invalid")
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
	if err != nil || len(content) > maxPhotoBytes {
		return nil, "", errors.New("photo is too large")
	}
	if len(content) == 0 {
		return nil, "", errors.New("photo is empty")
	}
	contentType := http.DetectContentType(content)
	if contentType != "image/jpeg" {
		return nil, "", errors.New("photo must be a JPEG image")
	}
	return content, contentType, nil
}

func compatibleBuild(headers []string) string {
	if len(headers) != 1 || !validBuild(headers[0]) {
		return ""
	}
	return headers[0]
}

func incompatibleProtocolVersion(protocols []string) *int {
	if len(protocols) != 1 {
		return nil
	}
	version, ok := parseSubprotocolVersion(protocols[0])
	if !ok {
		return nil
	}
	return &version
}

func isExpectedClose(err error) bool {
	if err == nil || errors.Is(err, errHubClosed) || errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrSessionInvalid) ||
		errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return true
	}
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure ||
		status == websocket.StatusGoingAway ||
		status == websocket.StatusPolicyViolation
}
