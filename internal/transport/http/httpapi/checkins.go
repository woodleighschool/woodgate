package httpapi

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListCheckins(writer http.ResponseWriter, request *http.Request, params ListCheckinsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	locationScope, err := checkinScope(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	var allowedLocationIDs []uuid.UUID
	if !locationScope.All {
		allowedLocationIDs = locationScope.Values
	}

	items, total, err := handler.admin.ListCheckins(request.Context(), domain.CheckinListOptions{
		ListOptions: listOptions,
		LocationID:  uuidPointer(params.LocationId),
		UserID:      uuidPointer(params.UserId),
		Direction:   checkinDirectionPointer(params.Direction),
		CreatedFrom: timePointer(params.CreatedFrom),
		CreatedTo:   timePointer(params.CreatedTo),
	}, allowedLocationIDs)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, CheckinListResponse{Rows: mapSliceValue(items, mapCheckin), Total: total})
}

func (handler *Server) CreateCheckin(writer http.ResponseWriter, request *http.Request) {
	body, err := parseCheckinCreateRequest(writer, request)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	principal, err := principalFromContext(request.Context())
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if principal.Bootstrap {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	subjectKind, subjectID, err := principalSubject(principal)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	locationScope, err := checkinScope(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionCreate,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if !locationScope.Contains(body.LocationID) {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	location, err := handler.admin.GetLocation(request.Context(), body.LocationID)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Checkin is invalid."}
	if !location.Enabled {
		validationErr.Add("location_id", "must reference an enabled location", "invalid")
	}
	if location.Photo && len(body.PhotoContent) == 0 {
		validationErr.Add("photo", "is required when the location requires a photo", "required")
	}
	if !location.Notes && body.Notes != "" {
		validationErr.Add("notes", "must be empty when notes are disabled for the location", "invalid")
	}
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateCheckin(
		request.Context(),
		body.UserID,
		body.LocationID,
		body.Direction,
		body.Notes,
		body.PhotoContent,
		subjectKind,
		subjectID,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapCheckin(item))
}

func (handler *Server) GetCheckin(writer http.ResponseWriter, request *http.Request, id Id) {
	locationScope, err := checkinScope(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	var allowedLocationIDs []uuid.UUID
	if !locationScope.All {
		allowedLocationIDs = locationScope.Values
	}

	item, err := handler.admin.GetCheckin(request.Context(), id, allowedLocationIDs)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "checkin not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapCheckin(item))
}

type checkinCreateRequest struct {
	UserID       uuid.UUID
	LocationID   uuid.UUID
	Direction    domain.CheckinDirection
	Notes        string
	PhotoContent []byte
}

func parseCheckinCreateRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (checkinCreateRequest, error) {
	parseErr := parseMultipartForm(writer, request)
	if parseErr != nil {
		return checkinCreateRequest{}, parseErr
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Checkin is invalid."}

	userID, userIDErr := uuid.Parse(strings.TrimSpace(multipartValue(request, "user_id")))
	if userIDErr != nil {
		validationErr.Add("user_id", "is invalid", "invalid")
	}

	locationID, locationIDErr := uuid.Parse(strings.TrimSpace(multipartValue(request, "location_id")))
	if locationIDErr != nil {
		validationErr.Add("location_id", "is invalid", "invalid")
	}

	direction, directionErr := domain.ParseCheckinDirection(strings.TrimSpace(multipartValue(request, "direction")))
	if directionErr != nil {
		validationErr.Add("direction", "is invalid", "invalid")
	}

	notes := strings.TrimSpace(multipartValue(request, "notes"))
	photoContent, fileErr := readMultipartFile(request, "photo", false, validationErr)
	if fileErr != nil {
		return checkinCreateRequest{}, fileErr
	}

	if validationErr.HasFieldErrors() {
		return checkinCreateRequest{}, validationErr
	}

	return checkinCreateRequest{
		UserID:       userID,
		LocationID:   locationID,
		Direction:    direction,
		Notes:        notes,
		PhotoContent: photoContent,
	}, nil
}
