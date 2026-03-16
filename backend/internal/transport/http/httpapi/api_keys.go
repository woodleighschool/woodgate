package httpapi

import (
	"net/http"
	"strings"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListAPIKeys(writer http.ResponseWriter, request *http.Request, params ListAPIKeysParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListAPIKeys(
		request.Context(),
		domain.APIKeyListOptions{ListOptions: listOptions},
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, APIKeyListResponse{Rows: mapSliceValue(items, mapAPIKey), Total: total})
}

func (handler *Server) CreateAPIKey(writer http.ResponseWriter, request *http.Request) {
	var body CreateAPIKeyJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeClassifiedError(writer, &domain.ValidationError{
			Code:   "validation_error",
			Detail: "API key is invalid.",
			FieldErrors: []domain.FieldError{{
				Field:   "name",
				Message: "must not be empty",
				Code:    "required",
			}},
		}, apiErrorOptions{})
		return
	}

	secret, prefix, secretHash, err := generateAPISecret()
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateAPIKey(request.Context(), name, prefix, secretHash, body.ExpiresAt)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, CreateAPIKeyData{
		Id:         idFromUUID(item.ID),
		Name:       item.Name,
		KeyPrefix:  item.KeyPrefix,
		LastUsedAt: item.LastUsedAt,
		ExpiresAt:  item.ExpiresAt,
		Admin:      false,
		Access:     []PermissionGrant{},
		CreatedAt:  item.CreatedAt,
		Secret:     secret,
	})
}

func (handler *Server) GetAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetAPIKey(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapAPIKey(item))
}

func (handler *Server) PatchAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchAPIKeyJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	permissions, err := validatePermissionGrants(body.Access, "API key access is invalid.")
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateAPIKeyAccess(request.Context(), id, body.Admin, permissions)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapAPIKey(item))
}

func (handler *Server) DeleteAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	if err := handler.admin.DeleteAPIKey(request.Context(), id); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
