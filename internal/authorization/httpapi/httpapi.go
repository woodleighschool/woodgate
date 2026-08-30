package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/authorization"
)

type resourcesOutput struct {
	Body struct {
		Items []authorization.Definition `json:"items"`
	}
}

type rolesOutput struct {
	Body struct {
		Items []authorization.Role `json:"items"`
	}
}

type roleOutput struct {
	Body authorization.Role
}

type roleInput struct {
	Body authorization.RoleMutation
}

type roleIDInput struct {
	ID int64 `path:"id"`
}

type roleMutationInput struct {
	ID   int64 `path:"id"`
	Body authorization.RoleMutation
}

type assignmentsOutput struct {
	Body struct {
		Items []authorization.Assignment `json:"items"`
	}
}

type assignmentsInput struct {
	Body authorization.AssignmentMutation
}

func RegisterAPI(routes api.AppRoutes, service *authorization.Service, logger *slog.Logger) {
	register(routes.Protected, service, logger)
}

func RegisterOpenAPI(routes api.AppRoutes) {
	register(routes.Protected, nil, nil)
}

func register(humaAPI huma.API, service *authorization.Service, logger *slog.Logger) {
	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.View,
		huma.Operation{
			OperationID: "list-authorization-resources",
			Method:      http.MethodGet,
			Path:        "/api/authz/resources",
			Tags:        []string{api.TagAuthorization},
			Summary:     "List authorization resources",
		},
	), func(context.Context, *struct{}) (*resourcesOutput, error) {
		output := &resourcesOutput{}
		output.Body.Items = authorization.Definitions
		return output, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.View,
		huma.Operation{
			OperationID: "list-roles",
			Method:      http.MethodGet,
			Path:        "/api/authz/roles",
			Tags:        []string{api.TagAuthorization},
			Summary:     "List roles",
		},
	), func(ctx context.Context, _ *struct{}) (*rolesOutput, error) {
		roles, err := service.ListRoles(ctx)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-roles", err)
		}
		output := &rolesOutput{}
		output.Body.Items = roles
		return output, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.Edit,
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
		role, err := service.CreateRole(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.View,
		huma.Operation{
			OperationID: "get-role",
			Method:      http.MethodGet,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Get a role",
			Errors:      []int{http.StatusNotFound},
		},
	), func(ctx context.Context, input *roleIDInput) (*roleOutput, error) {
		role, err := service.GetRole(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.Edit,
		huma.Operation{
			OperationID: "update-role",
			Method:      http.MethodPut,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Update a role",
			Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
		},
	), func(ctx context.Context, input *roleMutationInput) (*roleOutput, error) {
		role, err := service.UpdateRole(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-role", "role", err)
		}
		return &roleOutput{Body: *role}, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.roles"),
		authorization.Edit,
		huma.Operation{
			OperationID: "delete-role",
			Method:      http.MethodDelete,
			Path:        "/api/authz/roles/{id}",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Delete a role",
			Errors:      []int{http.StatusNotFound, http.StatusConflict},
		},
	), func(ctx context.Context, input *roleIDInput) (*struct{}, error) {
		if err := service.DeleteRole(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-role", "role", err)
		}
		return &struct{}{}, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.assignments"),
		authorization.View,
		huma.Operation{
			OperationID: "list-role-assignments",
			Method:      http.MethodGet,
			Path:        "/api/authz/assignments",
			Tags:        []string{api.TagAuthorization},
			Summary:     "List role assignments",
		},
	), func(ctx context.Context, _ *struct{}) (*assignmentsOutput, error) {
		assignments, err := service.ListAssignments(ctx)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "list-role-assignments", err)
		}
		output := &assignmentsOutput{}
		output.Body.Items = assignments
		return output, nil
	})

	huma.Register(humaAPI, authorization.Require(
		humaAPI,
		service,
		authorization.Resource("authz.assignments"),
		authorization.Edit,
		huma.Operation{
			OperationID: "replace-role-assignments",
			Method:      http.MethodPut,
			Path:        "/api/authz/assignments",
			Tags:        []string{api.TagAuthorization},
			Summary:     "Replace a subject's role assignments",
			Errors:      []int{http.StatusBadRequest, http.StatusNotFound},
		},
	), func(ctx context.Context, input *assignmentsInput) (*struct{}, error) {
		if err := service.ReplaceAssignments(ctx, input.Body); err != nil {
			return nil, api.ResourceError(ctx, logger, "replace-role-assignments", "assignment", err)
		}
		return &struct{}{}, nil
	})
}
