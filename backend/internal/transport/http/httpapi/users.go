package httpapi

import (
	"net/http"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListUsers(writer http.ResponseWriter, request *http.Request, params ListUsersParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	locationID := uuidPointer(params.LocationId)
	principal, err := principalFromContext(request.Context())
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if principal.Kind == authz.PrincipalKindAPIKey {
		if locationID == nil {
			writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
			return
		}

		locationScope, locationsErr := checkinScope(
			request.Context(),
			handler.authorizer,
			domain.PermissionActionCreate,
		)
		if locationsErr != nil {
			writeClassifiedError(writer, locationsErr, apiErrorOptions{})
			return
		}
		if !locationScope.Contains(*locationID) {
			writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
			return
		}
	}

	items, total, err := handler.admin.ListUsers(request.Context(), domain.UserListOptions{
		ListOptions: listOptions,
		LocationID:  locationID,
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusOK, UserListResponse{Rows: mapSliceValue(items, mapUser), Total: total})
}

func (handler *Server) GetUser(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetUser(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "user not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapUser(item))
}

func (handler *Server) PatchUser(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchUserJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	permissions, err := validatePermissionGrants(body.Access, "User access is invalid.")
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateUserAccess(request.Context(), id, body.Admin, permissions)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "user not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapUser(item))
}
