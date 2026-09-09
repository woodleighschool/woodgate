package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/rbac"
)

type resourcesOutput struct {
	Body struct {
		Items []rbac.Definition `json:"items"`
	}
}

type rolesOutput struct {
	Body struct {
		Items []rbac.Role `json:"items"`
	}
}

type roleOutput struct {
	Body rbac.Role
}

type roleInput struct {
	Body rbac.RoleMutation
}

type roleIDInput struct {
	ID int64 `path:"id"`
}

type roleMutationInput struct {
	ID   int64 `path:"id"`
	Body rbac.RoleMutation
}

func RegisterAPI(routes api.AppRoutes, roles *rbac.Store, authorizer authhuma.Authorizer, logger *slog.Logger) {
	register(routes.Protected, roles, authorizer, logger)
}

func RegisterOpenAPI(routes api.AppRoutes) {
	register(routes.Protected, nil, nil, nil)
}

func register(humaAPI huma.API, store *rbac.Store, authorizer authhuma.Authorizer, logger *slog.Logger) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.View,
		huma.Operation{
			OperationID: "list-authorization-resources",
			Method:      http.MethodGet,
			Path:        "/api/authz/resources",
			Tags:        []string{api.TagAuthorization},
			Summary:     "List authorization resources",
		},
	), func(context.Context, *struct{}) (*resourcesOutput, error) {
		output := &resourcesOutput{}
		output.Body.Items = rbac.Definitions()
		return output, nil
	})

	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.View,
		huma.Operation{
			OperationID: "list-roles",
			Method:      http.MethodGet,
			Path:        "/api/authz/roles",
			Tags:        []string{api.TagAuthorization},
			Summary:     "List roles",
		},
	), func(ctx context.Context, _ *struct{}) (*rolesOutput, error) {
		roles, err := store.ListRoles(ctx)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-roles", err)
		}
		output := &rolesOutput{}
		output.Body.Items = roles
		return output, nil
	})

	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.Edit,
		huma.Operation{
			OperationID:   "create-role",
			Method:        http.MethodPost,
			Path:          "/api/authz/roles",
			Tags:          []string{api.TagAuthorization},
			Summary:       "Create a role",
			DefaultStatus: http.StatusCreated,
			Errors:        []int{http.StatusBadRequest, http.StatusConflict},
		},
	), func(ctx context.Context, input *roleInput) (*roleOutput, error) {
		role, err := store.CreateRole(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.View,
		huma.Operation{
			OperationID: "get-role",
			Method:      http.MethodGet,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Get a role",
			Errors:      []int{http.StatusNotFound},
		},
	), func(ctx context.Context, input *roleIDInput) (*roleOutput, error) {
		role, err := store.GetRole(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.Edit,
		huma.Operation{
			OperationID: "update-role",
			Method:      http.MethodPut,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Update a role",
			Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
		},
	), func(ctx context.Context, input *roleMutationInput) (*roleOutput, error) {
		role, err := store.UpdateRole(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger,
		rbac.ResourceRoles,
		authz.Edit,
		huma.Operation{
			OperationID: "delete-role",
			Method:      http.MethodDelete,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Delete a role",
			Errors:      []int{http.StatusNotFound, http.StatusConflict},
		},
	), func(ctx context.Context, input *roleIDInput) (*struct{}, error) {
		if err := store.DeleteRole(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-role", "role", err)
		}
		return &struct{}{}, nil
	})
}
