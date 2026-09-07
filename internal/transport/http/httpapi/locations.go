package httpapi

import (
	"context"
	"strings"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listLocations(ctx context.Context, input *ListLocationsParams) (*response[LocationListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	items, total, err := handler.admin.ListLocations(ctx, domain.LocationListOptions{
		ListOptions: listOptions,
		Enabled:     boolPointer(params.Enabled.Value),
	})
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[LocationListResponse]{Body: LocationListResponse{Rows: mapSliceValue(items, mapLocation), Total: total}}, nil
}

func (handler *Server) createLocation(ctx context.Context, input *BodyInput[LocationWriteRequest]) (*response[Location], error) {
	body := input.Body.Value

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		return nil, classifiedError(validationErr, apiErrorOptions{})
	}

	item, err := handler.admin.CreateLocation(
		ctx,
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetID),
		uuidPointer(body.LogoAssetID),
		uuidSlice(body.GroupIDs),
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[Location]{Body: mapLocation(item)}, nil
}

func (handler *Server) getLocation(ctx context.Context, input *ItemInput) (*response[Location], error) {
	id := input.ID

	item, err := handler.admin.GetLocation(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "location not found"})
	}
	return &response[Location]{Body: mapLocation(item)}, nil
}

func (handler *Server) patchLocation(ctx context.Context, input *patchInput[LocationWriteRequest]) (*response[Location], error) {
	id := input.ID
	body := input.Body.Value

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		return nil, classifiedError(validationErr, apiErrorOptions{})
	}

	item, err := handler.admin.UpdateLocation(
		ctx,
		id,
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetID),
		uuidPointer(body.LogoAssetID),
		uuidSlice(body.GroupIDs),
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "location not found"})
	}
	return &response[Location]{Body: mapLocation(item)}, nil
}

func (handler *Server) deleteLocation(ctx context.Context, input *ItemInput) (*struct{}, error) {
	id := input.ID

	if err := handler.admin.DeleteLocation(ctx, id); err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "location not found"})
	}
	return nil, nil
}
