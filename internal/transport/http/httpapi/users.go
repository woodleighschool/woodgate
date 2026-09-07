package httpapi

import (
	"context"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listUsers(ctx context.Context, input *ListUsersParams) (*response[UserListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	locationID := uuidPointer(params.LocationID.Value)
	principal, err := principalFromContext(ctx)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}
	if principal.Kind == authz.PrincipalKindAPIKey {
		if locationID == nil {
			return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
		}

		locationScope, locationsErr := checkinScope(
			ctx,
			handler.authorizer,
			domain.PermissionActionCreate,
		)
		if locationsErr != nil {
			return nil, classifiedError(locationsErr, apiErrorOptions{})
		}
		if !locationScope.Contains(*locationID) {
			return nil, classifiedError(domain.ErrPermissionDenied, apiErrorOptions{})
		}
	}

	items, total, err := handler.admin.ListUsers(ctx, domain.UserListOptions{
		ListOptions: listOptions,
		LocationID:  locationID,
	})
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[UserListResponse]{Body: UserListResponse{Rows: mapSliceValue(items, mapUser), Total: total}}, nil
}

func (handler *Server) getUser(ctx context.Context, input *ItemInput) (*response[User], error) {
	id := input.ID

	item, err := handler.admin.GetUser(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "user not found"})
	}

	return &response[User]{Body: mapUser(item)}, nil
}

func (handler *Server) patchUser(ctx context.Context, input *patchInput[UserAccessWriteRequest]) (*response[User], error) {
	id := input.ID
	body := input.Body.Value

	permissions, err := validatePermissionGrants(body.Access, "User access is invalid.")
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	item, err := handler.admin.UpdateUserAccess(ctx, id, body.Admin, permissions)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "user not found"})
	}

	return &response[User]{Body: mapUser(item)}, nil
}
