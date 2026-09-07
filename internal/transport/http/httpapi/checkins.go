package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listCheckins(ctx context.Context, input *ListCheckinsParams) (*response[CheckinListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	locationScope, err := checkinScope(
		ctx,
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	var allowedLocationIDs []uuid.UUID
	if !locationScope.All {
		allowedLocationIDs = locationScope.Values
	}

	items, total, err := handler.admin.ListCheckins(ctx, domain.CheckinListOptions{
		ListOptions: listOptions,
		LocationID:  uuidPointer(params.LocationID.Value),
		UserID:      uuidPointer(params.UserID.Value),
		Direction:   checkinDirectionPointer(params.Direction.Value),
		Department:  optionalString(params.Department.Value),
		CreatedFrom: timePointer(params.CreatedFrom.Value),
		CreatedTo:   timePointer(params.CreatedTo.Value),
	}, allowedLocationIDs)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[CheckinListResponse]{Body: CheckinListResponse{Rows: mapSliceValue(items, mapCheckin), Total: total}}, nil
}

func (handler *Server) listCheckinDepartments(ctx context.Context, _ *struct{}) (*response[DepartmentOptionListResponse], error) {
	locationScope, err := checkinScope(
		ctx,
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	var allowedLocationIDs []uuid.UUID
	if !locationScope.All {
		allowedLocationIDs = locationScope.Values
	}

	items, err := handler.admin.ListCheckinDepartments(ctx, allowedLocationIDs)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[DepartmentOptionListResponse]{Body: DepartmentOptionListResponse{
		Rows:  mapSliceValue(items, mapDepartmentOption),
		Total: safeInt32(len(items)),
	}}, nil
}

func (handler *Server) createCheckin(ctx context.Context, input *RequestInput) (*response[Checkin], error) {
	writer, request := input.writer, input.request

	body, err := parseCheckinCreateRequest(writer, request)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	principal, err := principalFromContext(ctx)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	if principal.Bootstrap {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}

	subjectKind, subjectID, err := principalSubject(principal)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	locationScope, err := checkinScope(
		ctx,
		handler.authorizer,
		domain.PermissionActionCreate,
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	if !locationScope.Contains(body.LocationID) {
		return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
	}

	location, err := handler.admin.GetLocation(ctx, body.LocationID)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "location not found"})
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
		return nil, classifiedError(validationErr, apiErrorOptions{})
	}

	item, err := handler.admin.CreateCheckin(
		ctx,
		body.UserID,
		body.LocationID,
		body.Direction,
		body.Notes,
		body.PhotoContent,
		subjectKind,
		subjectID,
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[Checkin]{Body: mapCheckin(item)}, nil
}

func (handler *Server) getCheckin(ctx context.Context, input *ItemInput) (*response[Checkin], error) {
	id := input.ID

	locationScope, err := checkinScope(
		ctx,
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	var allowedLocationIDs []uuid.UUID
	if !locationScope.All {
		allowedLocationIDs = locationScope.Values
	}

	item, err := handler.admin.GetCheckin(ctx, id, allowedLocationIDs)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "checkin not found"})
	}

	return &response[Checkin]{Body: mapCheckin(item)}, nil
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
