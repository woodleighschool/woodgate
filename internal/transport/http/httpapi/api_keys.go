package httpapi

import (
	"context"
	"strings"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) listAPIKeys(ctx context.Context, input *ListAPIKeysParams) (*response[APIKeyListResponse], error) {
	params := input

	listOptions, err := parseListOptions(params.Limit.Value, params.Offset.Value, params.Search.Value, params.Sort.Value, params.Order.Value)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	items, total, err := handler.admin.ListAPIKeys(
		ctx,
		domain.APIKeyListOptions{ListOptions: listOptions},
	)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[APIKeyListResponse]{Body: APIKeyListResponse{Rows: mapSliceValue(items, mapAPIKey), Total: total}}, nil
}

func (handler *Server) createAPIKey(ctx context.Context, input *BodyInput[APIKeyCreateRequest]) (*response[CreateAPIKeyData], error) {
	body := input.Body.Value

	name := strings.TrimSpace(body.Name)
	if name == "" {
		return nil, classifiedError(&domain.ValidationError{
			Code:   "validation_error",
			Detail: "API key is invalid.",
			FieldErrors: []domain.FieldError{{
				Field:   "name",
				Message: "must not be empty",
				Code:    "required",
			}},
		}, apiErrorOptions{})
	}

	secret, prefix, secretHash, err := generateAPISecret()
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	item, err := handler.admin.CreateAPIKey(ctx, name, prefix, secretHash, body.ExpiresAt)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	return &response[CreateAPIKeyData]{Body: CreateAPIKeyData{
		ID:         idFromUUID(item.ID),
		Name:       item.Name,
		KeyPrefix:  item.KeyPrefix,
		LastUsedAt: item.LastUsedAt,
		ExpiresAt:  item.ExpiresAt,
		Admin:      false,
		Access:     []PermissionGrant{},
		CreatedAt:  item.CreatedAt,
		Secret:     secret,
	}}, nil
}

func (handler *Server) getAPIKey(ctx context.Context, input *ItemInput) (*response[APIKey], error) {
	id := input.ID

	item, err := handler.admin.GetAPIKey(ctx, id)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "api key not found"})
	}
	return &response[APIKey]{Body: mapAPIKey(item)}, nil
}

func (handler *Server) patchAPIKey(ctx context.Context, input *patchInput[APIKeyAccessWriteRequest]) (*response[APIKey], error) {
	id := input.ID
	body := input.Body.Value

	permissions, err := validatePermissionGrants(body.Access, "API key access is invalid.")
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{})
	}

	item, err := handler.admin.UpdateAPIKeyAccess(ctx, id, body.Admin, permissions)
	if err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "api key not found"})
	}

	return &response[APIKey]{Body: mapAPIKey(item)}, nil
}

func (handler *Server) deleteAPIKey(ctx context.Context, input *ItemInput) (*struct{}, error) {
	id := input.ID

	if err := handler.admin.DeleteAPIKey(ctx, id); err != nil {
		return nil, classifiedError(err, apiErrorOptions{NotFoundMessage: "api key not found"})
	}
	return nil, nil
}
