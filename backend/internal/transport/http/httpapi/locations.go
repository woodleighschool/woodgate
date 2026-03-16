package httpapi

import (
	"net/http"
	"strings"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListLocations(writer http.ResponseWriter, request *http.Request, params ListLocationsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListLocations(request.Context(), domain.LocationListOptions{
		ListOptions: listOptions,
		Enabled:     boolPointer(params.Enabled),
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, LocationListResponse{Rows: mapSliceValue(items, mapLocation), Total: total})
}

func (handler *Server) CreateLocation(writer http.ResponseWriter, request *http.Request) {
	var body CreateLocationJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateLocation(
		request.Context(),
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetId),
		uuidPointer(body.LogoAssetId),
		uuidSlice(body.GroupIds),
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapLocation(item))
}

func (handler *Server) GetLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetLocation(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapLocation(item))
}

func (handler *Server) PatchLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchLocationJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateLocation(
		request.Context(),
		id,
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetId),
		uuidPointer(body.LogoAssetId),
		uuidSlice(body.GroupIds),
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapLocation(item))
}

func (handler *Server) DeleteLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	if err := handler.admin.DeleteLocation(request.Context(), id); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
