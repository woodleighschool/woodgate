package appv1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/woodleighschool/goodies/bloby"
	"github.com/woodleighschool/woodgate/internal/appkey"
	"github.com/woodleighschool/woodgate/internal/checkin"
	"github.com/woodleighschool/woodgate/internal/fault"
)

// Dependencies are the shared capabilities used by the companion protocol.
type Dependencies struct {
	Store    *Store
	Keys     *appkey.Store
	Checkins *checkin.Service
	Objects  *bloby.Service
	Logger   *slog.Logger
}

type keyContext struct{}

// RegisterRoutes mounts the companion protocol without browser authentication.
func RegisterRoutes(ordinary, transfers chi.Router, deps Dependencies) {
	ordinary = ordinary.With(deps.authenticate)
	transfers = transfers.With(deps.authenticate)
	ordinary.Get("/auth/me", deps.me)
	ordinary.Get("/api/v1/locations", deps.listLocations)
	ordinary.Get("/api/v1/locations/{id}", deps.getLocation)
	ordinary.Get("/api/v1/users", deps.listPeople)
	transfers.Post("/api/v1/checkins", deps.createCheckin)
	transfers.Get("/api/v1/assets/{id}/content", deps.assetContent)
}

func (deps Dependencies) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := deps.Keys.Authenticate(r.Context(), r.Header.Get("X-Api-Key"))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), keyContext{}, *key)))
	})
}
func requestKey(r *http.Request) appkey.Key {
	key, _ := r.Context().Value(keyContext{}).(appkey.Key)
	return key
}

type listResponse[T any] struct {
	Rows  []T `json:"rows"`
	Total int `json:"total"`
}

type grant struct {
	Resource   string     `json:"resource"`
	Action     string     `json:"action"`
	LocationID *uuid.UUID `json:"location_id,omitempty"`
	AssetType  string     `json:"asset_type,omitempty"`
}
type capability struct {
	Read   bool `json:"read"`
	Create bool `json:"create"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}
type meResponse struct {
	Principal struct {
		Type string    `json:"type"`
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"principal"`
	Admin        bool                  `json:"admin"`
	Access       []grant               `json:"access"`
	Capabilities map[string]capability `json:"capabilities"`
}

func (deps Dependencies) me(w http.ResponseWriter, r *http.Request) {
	key := requestKey(r)
	locations, err := deps.Store.locations(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	response := meResponse{Access: []grant{}, Capabilities: map[string]capability{"users": {}, "groups": {}, "locations": {}, "checkins": {}, "assets": {}, "api_keys": {}}}
	response.Principal.Type = "api_key"
	response.Principal.ID = key.AppID
	response.Principal.Name = key.Name

	if key.ReadUsers {
		response.Access = append(response.Access, grant{Resource: "users", Action: "read"})
		response.Capabilities["users"] = capability{Read: true}
	}
	if key.ReadLocations {
		response.Access = append(response.Access, grant{Resource: "locations", Action: "read"})
		response.Capabilities["locations"] = capability{Read: true}
	}
	if key.ReadBranding {
		response.Access = append(response.Access, grant{Resource: "assets", Action: "read", AssetType: "asset"})
		response.Capabilities["assets"] = capability{Read: true}
	}
	if len(locations) > 0 || key.AllLocations {
		response.Capabilities["checkins"] = capability{Create: true}
	}

	for _, item := range locations {
		response.Access = append(response.Access, grant{Resource: "checkins", Action: "create", LocationID: &item.ID})
	}
	writeJSON(w, http.StatusOK, response)
}

func (deps Dependencies) listLocations(w http.ResponseWriter, r *http.Request) {
	if !requestKey(r).ReadLocations {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	limit, offset, problem := pagination(r)
	if problem != nil {
		writeProblem(w, *problem)
		return
	}
	items, err := deps.Store.locations(r.Context(), requestKey(r))
	if err != nil {
		deps.fail(w, r, err, "")
		return
	}
	count := len(items)
	start := min(offset, count)
	end := count
	if limit > 0 {
		end = start + min(limit, count-start)
	}
	writeJSON(w, http.StatusOK, listResponse[location]{Rows: items[start:end], Total: count})
}
func (deps Dependencies) getLocation(w http.ResponseWriter, r *http.Request) {
	if !requestKey(r).ReadLocations {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, invalidRequest("invalid path parameter: id"))
		return
	}
	item, err := deps.Store.location(r.Context(), id)
	if err != nil {
		deps.fail(w, r, err, "location not found")
		return
	}
	if !requestKey(r).Allows(item.InternalID) {
		writeProblem(w, forbidden())
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (deps Dependencies) listPeople(w http.ResponseWriter, r *http.Request) {
	if !requestKey(r).ReadUsers {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	limit, offset, problem := pagination(r)
	if problem != nil {
		writeProblem(w, *problem)
		return
	}
	id, err := uuid.Parse(r.URL.Query().Get("location_id"))
	if err != nil {
		writeProblem(w, invalidRequest("invalid query parameter: location_id"))
		return
	}
	location, err := deps.Store.location(r.Context(), id)
	if err != nil {
		deps.fail(w, r, err, "location not found")
		return
	}
	if !requestKey(r).Allows(location.InternalID) {
		writeProblem(w, forbidden())
		return
	}
	items, count, err := deps.Store.people(r.Context(), location.InternalID, limit, offset)
	if err != nil {
		deps.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusOK, listResponse[person]{Rows: items, Total: count})
}

type checkinResponse struct {
	ID              uuid.UUID         `json:"id"`
	UserID          uuid.UUID         `json:"user_id"`
	UserDisplayName string            `json:"user_display_name"`
	Department      string            `json:"department"`
	LocationID      uuid.UUID         `json:"location_id"`
	LocationName    string            `json:"location_name"`
	Direction       checkin.Direction `json:"direction"`
	Notes           string            `json:"notes"`
	CreatedByKind   string            `json:"created_by_kind"`
	CreatedByID     uuid.UUID         `json:"created_by_id"`
	CreatedAt       time.Time         `json:"created_at"`
}

func (deps Dependencies) createCheckin(w http.ResponseWriter, r *http.Request) {
	key := requestKey(r)
	if !key.AllLocations && len(key.Locations) == 0 {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	body, problem := parseSubmission(w, r)
	if r.MultipartForm != nil {
		defer func() { _ = r.MultipartForm.RemoveAll() }()
	}
	if problem != nil {
		writeProblem(w, *problem)
		return
	}
	location, err := deps.Store.location(r.Context(), body.LocationID)
	if err != nil {
		if errors.Is(err, fault.ErrNotFound) && !key.AllLocations {
			writeProblem(w, forbidden())
			return
		}
		deps.fail(w, r, err, "location not found")
		return
	}
	if !key.Allows(location.InternalID) {
		writeProblem(w, forbidden())
		return
	}
	fields := []fieldError{}
	if !location.Enabled {
		fields = append(fields, fieldError{Field: "location_id", Message: "must reference an enabled location", Code: "invalid"})
	}
	if location.Photo && len(body.Photo) == 0 {
		fields = append(fields, fieldError{Field: "photo", Message: "is required when the location requires a photo", Code: "required"})
	}
	if !location.Notes && body.Notes != "" {
		fields = append(fields, fieldError{Field: "notes", Message: "must be empty when notes are disabled for the location", Code: "invalid"})
	}
	if len(fields) > 0 {
		writeProblem(w, validationProblem("Checkin is invalid.", fields))
		return
	}
	if len(body.Photo) > 0 {
		contentType := mimetype.Detect(body.Photo)
		if !contentType.Is("image/png") && !contentType.Is("image/jpeg") {
			writeProblem(w, validationProblem("Asset is invalid.", []fieldError{{Field: "file", Message: "must be a PNG or JPEG image", Code: "invalid"}}))
			return
		}
	}
	person, err := deps.Store.person(r.Context(), body.UserID)
	if err != nil {
		if errors.Is(err, fault.ErrNotFound) {
			writeProblem(w, validationProblem("Referenced resource does not exist.", nil))
			return
		}
		deps.fail(w, r, err, "")
		return
	}

	item, err := deps.Checkins.Submit(r.Context(), checkin.CheckinCreate{UserID: person.InternalID, LocationID: location.InternalID, Direction: body.Direction, Notes: body.Notes}, checkin.Actor{Kind: "api_key", AppID: key.AppID, AppKeyID: &key.ID}, body.Photo)
	if err != nil {
		deps.fail(w, r, err, "")
		return
	}
	writeJSON(w, http.StatusCreated, checkinResponse{ID: item.AppID, UserID: person.ID, UserDisplayName: person.DisplayName, Department: person.Department, LocationID: location.ID, LocationName: location.Name, Direction: item.Direction, Notes: item.Notes, CreatedByKind: item.ActorKind, CreatedByID: item.ActorAppID, CreatedAt: item.CreatedAt})
}

func (deps Dependencies) assetContent(w http.ResponseWriter, r *http.Request) {
	if !requestKey(r).ReadBranding {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, invalidRequest("invalid path parameter: id"))
		return
	}
	attachment, err := deps.Store.attachment(r.Context(), id)
	if err != nil {
		deps.fail(w, r, err, "asset not found")
		return
	}
	if !requestKey(r).Allows(attachment.LocationID) {
		writeProblem(w, forbidden())
		return
	}
	object, err := deps.Objects.GetByID(r.Context(), attachment.ObjectID)
	if err != nil {
		deps.fail(w, r, err, "asset not found")
		return
	}
	if !object.Available() || (object.Prefix != checkin.BackgroundObjectPrefix && object.Prefix != checkin.LogoObjectPrefix) {
		deps.fail(w, r, fault.ErrNotFound, "asset not found")
		return
	}
	if err := deps.Objects.Deliver(w, r, *object, bloby.DeliveryOptions{CacheControl: "private, max-age=3600"}); err != nil {
		deps.fail(w, r, err, "asset not found")
	}
}

const maxMultipartBytes = 10 << 20

type submission struct {
	UserID     uuid.UUID
	LocationID uuid.UUID
	Direction  checkin.Direction
	Notes      string
	Photo      []byte
}

func parseSubmission(w http.ResponseWriter, r *http.Request) (submission, *problem) {
	fail := func(detail string) (submission, *problem) { p := invalidRequest(detail); return submission{}, &p }
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		return fail("request body must be multipart/form-data")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartBytes)
	//nolint:gosec // MaxBytesReader above caps the entire multipart request at 10 MiB.
	if err := r.ParseMultipartForm(maxMultipartBytes); err != nil {
		return fail("request body is invalid")
	}
	body := submission{Direction: checkin.Direction(strings.TrimSpace(multipartValue(r, "direction"))), Notes: strings.TrimSpace(multipartValue(r, "notes"))}
	fields := []fieldError{}
	var err error
	body.UserID, err = uuid.Parse(strings.TrimSpace(multipartValue(r, "user_id")))
	if err != nil {
		fields = append(fields, fieldError{Field: "user_id", Message: "is invalid", Code: "invalid"})
	}
	body.LocationID, err = uuid.Parse(strings.TrimSpace(multipartValue(r, "location_id")))
	if err != nil {
		fields = append(fields, fieldError{Field: "location_id", Message: "is invalid", Code: "invalid"})
	}
	if body.Direction != checkin.DirectionIn && body.Direction != checkin.DirectionOut {
		fields = append(fields, fieldError{Field: "direction", Message: "is invalid", Code: "invalid"})
	}
	file, _, err := r.FormFile("photo")
	switch {
	case errors.Is(err, http.ErrMissingFile):
	case err != nil:
		return fail("request body is invalid")
	default:
		defer func() { _ = file.Close() }()
		body.Photo, err = io.ReadAll(io.LimitReader(file, maxMultipartBytes+1))
		if err != nil {
			return fail("request body is invalid")
		}
		if len(body.Photo) == 0 {
			fields = append(fields, fieldError{Field: "photo", Message: "must not be empty", Code: "required"})
		}
		if len(body.Photo) > maxMultipartBytes {
			fields = append(fields, fieldError{Field: "photo", Message: "is too large", Code: "invalid"})
		}
	}
	if len(fields) > 0 {
		p := validationProblem("Checkin is invalid.", fields)
		return submission{}, &p
	}
	return body, nil
}

func pagination(r *http.Request) (int, int, *problem) {
	values := r.URL.Query()
	limit, offset := 0, 0
	for _, item := range []struct {
		name   string
		target *int
	}{{"limit", &limit}, {"offset", &offset}} {
		raw, ok := values[item.name]
		if !ok {
			continue
		}
		value, err := strconv.ParseInt(raw[0], 10, 32)
		if err != nil {
			p := invalidRequest("invalid query parameter: " + item.name)
			return 0, 0, &p
		}
		if item.name == "limit" && value < 1 {
			p := invalidRequest("limit must be >= 1")
			return 0, 0, &p
		}
		if item.name == "offset" && value < 0 {
			p := invalidRequest("offset must be >= 0")
			return 0, 0, &p
		}
		*item.target = int(value)
	}
	return limit, offset, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
type problem struct {
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Status      int          `json:"status"`
	Detail      string       `json:"detail"`
	Code        string       `json:"code"`
	FieldErrors []fieldError `json:"field_errors,omitempty"`
}

func invalidRequest(detail string) problem {
	return problem{Type: "urn:woodgate:problem:invalid-request", Title: "Invalid request", Status: http.StatusBadRequest, Detail: detail, Code: "invalid_request"}
}
func forbidden() problem {
	return problem{Type: "urn:woodgate:problem:forbidden", Title: "Forbidden", Status: http.StatusForbidden, Detail: "Permission denied.", Code: "forbidden"}
}
func validationProblem(detail string, fields []fieldError) problem {
	return problem{Type: "urn:woodgate:problem:validation-error", Title: "Validation failed", Status: http.StatusUnprocessableEntity, Detail: detail, Code: "validation_error", FieldErrors: fields}
}
func writeProblem(w http.ResponseWriter, p problem) { writeJSON(w, p.Status, p) }
func (deps Dependencies) fail(w http.ResponseWriter, r *http.Request, err error, notFound string) {
	switch {
	case (errors.Is(err, fault.ErrNotFound) || errors.Is(err, bloby.ErrNotFound)) && notFound != "":
		writeProblem(w, problem{Type: "urn:woodgate:problem:not-found", Title: "Not found", Status: http.StatusNotFound, Detail: notFound, Code: "not_found"})
	case errors.Is(err, fault.ErrNotFound):
		writeProblem(w, validationProblem("Referenced resource does not exist.", nil))
	case errors.Is(err, fault.ErrInvalidInput), errors.Is(err, bloby.ErrInvalidInput):
		writeProblem(w, validationProblem("Checkin is invalid.", nil))
	default:
		if deps.Logger != nil {
			deps.Logger.ErrorContext(r.Context(), "companion request failed", "err", err)
		}
		writeProblem(w, problem{Type: "urn:woodgate:problem:internal-error", Title: "Internal server error", Status: http.StatusInternalServerError, Detail: "An internal server error occurred.", Code: "internal_error"})
	}
}

func multipartValue(r *http.Request, name string) string {
	values := r.MultipartForm.Value[name]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
