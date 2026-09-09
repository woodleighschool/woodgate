package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/directory/entra"
	"github.com/woodleighschool/woodgate/internal/fault"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/rbac"
)

const (
	userResource = "user"
	userIDPath   = "/api/users/{id}"
)

type userListOutput struct {
	Body api.Page[directory.User]
}

type userRoleListOutput struct {
	Body struct {
		Items []directory.RoleSummary `json:"items"`
	}
}

type userOutput struct {
	Body directory.User
}

type userListInput struct {
	api.ListQueryInput

	Values  []string `query:"values,omitempty"`
	Role    []string `query:"role,omitempty"`
	Source  string   `query:"source,omitempty" enum:"local,entra"`
	GroupID int64    `query:"group_id,omitempty" minimum:"1"`
}

type userCreateInput struct {
	Body directory.UserCreate
}

type userGetInput struct {
	ID int64 `path:"id"`
}

type userPutInput struct {
	ID   int64 `path:"id"`
	Body directory.UserMutation
}

type userDeleteInput struct {
	ID int64 `path:"id"`
}

// RegisterAPI mounts directory management endpoints.
func RegisterAPI(
	routes api.AppRoutes,
	userService *directory.UserService,
	store *directory.Store,
	syncJobs *entra.SyncJobs,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	registerAPI(routes, userService, store, syncJobs, authorizer, logger)
}

// RegisterOpenAPI documents directory endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) {
	registerAPI(routes, nil, nil, nil, nil, nil)
}

func registerAPI(
	routes api.AppRoutes,
	userService *directory.UserService,
	store *directory.Store,
	syncJobs *entra.SyncJobs,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	humaAPI := routes.Protected
	registerDirectorySync(humaAPI, syncJobs, authorizer, logger)
	registerListUsers(humaAPI, userService, authorizer, logger)
	registerListUserRoles(humaAPI, userService, authorizer, logger)
	registerCreateUser(humaAPI, userService, authorizer, logger)
	registerGetUser(humaAPI, userService, authorizer, logger)
	registerPutUser(humaAPI, userService, authorizer, logger)
	registerDeleteUser(humaAPI, userService, authorizer, logger)
	registerListGroups(humaAPI, store, authorizer, logger)
	registerGetGroup(humaAPI, store, authorizer, logger)
	registerPutGroup(humaAPI, store, authorizer, logger)
}

func registerListUserRoles(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger, rbac.ResourceUsers, authz.View, huma.Operation{
		OperationID: "list-user-roles",
		Method:      http.MethodGet,
		Path:        "/api/users/roles",
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "List user role choices",
	}), func(ctx context.Context, _ *struct{}) (*userRoleListOutput, error) {
		roles, err := userService.ListRoleSummaries(ctx)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-user-roles", "role", err)
		}
		output := &userRoleListOutput{}
		output.Body.Items = roles
		return output, nil
	})
}

func (i userListInput) params() directory.UserListParams {
	return directory.UserListParams{
		ListParams: i.Params(),
		Values:     listing.NormalizeValues(i.Values),
		RoleValues: listing.NormalizeValues(i.Role),
		Source:     i.Source,
		GroupID:    i.GroupID,
	}
}

func registerListUsers(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger, rbac.ResourceUsers, authz.View, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/users",
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "List users",
	}), func(ctx context.Context, input *userListInput) (*userListOutput, error) {
		list, count, err := userService.List(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-users", userResource, err)
		}
		return &userListOutput{Body: api.Page[directory.User]{Items: list, Count: count}}, nil
	})
}

func registerCreateUser(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.RequireAll(humaAPI, authorizer, logger, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/api/users",
		Tags:          []string{api.TagDirectoryUsers},
		Summary:       "Create a user",
		DefaultStatus: http.StatusCreated,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusConflict,
		},
	},
		authz.Requirement{Resource: rbac.ResourceUsers, Access: authz.Edit},
		authz.Requirement{Resource: rbac.ResourceRoles, Access: authz.Edit},
	), func(ctx context.Context, input *userCreateInput) (*userOutput, error) {
		user, err := userService.Create(ctx, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "create-user", userMutationError(err))
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerGetUser(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger, rbac.ResourceUsers, authz.View, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Get a user",
		Errors:      []int{http.StatusNotFound},
	}), func(ctx context.Context, input *userGetInput) (*userOutput, error) {
		user, err := userService.Get(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-user", userResource, err, "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerPutUser(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.RequireAll(humaAPI, authorizer, logger, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPut,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Update a user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusNotFound,
			http.StatusConflict,
		},
	},
		authz.Requirement{Resource: rbac.ResourceUsers, Access: authz.Edit},
		authz.Requirement{Resource: rbac.ResourceRoles, Access: authz.Edit},
	), func(ctx context.Context, input *userPutInput) (*userOutput, error) {
		user, err := userService.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.HandlerError(ctx, logger, "update-user", userMutationError(err), "user_id", input.ID)
		}
		return &userOutput{Body: *user}, nil
	})
}

func registerDeleteUser(
	humaAPI huma.API,
	userService *directory.UserService,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.RequireAll(humaAPI, authorizer, logger, huma.Operation{
		OperationID: "delete-user",
		Method:      http.MethodDelete,
		Path:        userIDPath,
		Tags:        []string{api.TagDirectoryUsers},
		Summary:     "Delete a user",
		Errors: []int{
			http.StatusNotFound,
			http.StatusConflict,
		},
	},
		authz.Requirement{Resource: rbac.ResourceUsers, Access: authz.Edit},
		authz.Requirement{Resource: rbac.ResourceRoles, Access: authz.Edit},
	), func(ctx context.Context, input *userDeleteInput) (*struct{}, error) {
		if err := userService.Delete(ctx, input.ID); err != nil {
			return nil, api.HandlerError(ctx, logger, "delete-user", userMutationError(err), "user_id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func userMutationError(err error) error {
	switch {
	case errors.Is(err, fault.ErrAlreadyExists):
		return huma.Error409Conflict("email already in use")
	case errors.Is(err, directory.ErrWeakPassword):
		return huma.Error400BadRequest(directory.ErrWeakPassword.Error())
	default:
		return api.ResourceMutationError(userResource, err)
	}
}
