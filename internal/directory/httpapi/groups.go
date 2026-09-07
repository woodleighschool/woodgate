package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/rbac"
)

const (
	groupResource = "group"
	groupIDPath   = "/api/groups/{id}"
)

type groupListInput struct {
	api.ListQueryInput

	Values []string `query:"values,omitempty"`
}

type groupGetInput struct {
	ID int64 `path:"id"`
}

type groupPutInput struct {
	ID   int64 `path:"id"`
	Body directory.GroupMutation
}

type groupListOutput struct {
	Body api.Page[directory.Group]
}

func registerPutGroup(
	humaAPI huma.API,
	groupStore *directory.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.RequireAll(humaAPI, authorizer, logger, huma.Operation{
		OperationID: "update-group",
		Method:      http.MethodPut,
		Path:        groupIDPath,
		Tags:        []string{api.TagDirectoryGroups},
		Summary:     "Update a group",
		Errors:      []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict},
	},
		authz.Requirement{Resource: rbac.ResourceGroups, Access: authz.Edit},
		authz.Requirement{Resource: rbac.ResourceRoles, Access: authz.Edit},
	), func(ctx context.Context, input *groupPutInput) (*groupOutput, error) {
		group, err := groupStore.UpdateGroup(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-group", groupResource, err, "group_id", input.ID)
		}
		return &groupOutput{Body: *group}, nil
	})
}

type groupOutput struct {
	Body directory.Group
}

func (i groupListInput) params() directory.GroupListParams {
	return directory.GroupListParams{
		ListParams: i.Params(),
		Values:     listing.NormalizeValues(i.Values),
	}
}

func registerListGroups(
	humaAPI huma.API,
	groupStore *directory.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger, rbac.ResourceGroups, authz.View, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        "/api/groups",
		Tags:        []string{api.TagDirectoryGroups},
		Summary:     "List groups",
	}), func(ctx context.Context, input *groupListInput) (*groupListOutput, error) {
		list, count, err := groupStore.ListGroups(ctx, input.params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-groups", groupResource, err)
		}
		return &groupListOutput{Body: api.Page[directory.Group]{Items: list, Count: count}}, nil
	})
}

func registerGetGroup(
	humaAPI huma.API,
	groupStore *directory.Store,
	authorizer authhuma.Authorizer,
	logger *slog.Logger,
) {
	huma.Register(humaAPI, authhuma.Require(humaAPI, authorizer, logger, rbac.ResourceGroups, authz.View, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        groupIDPath,
		Tags:        []string{api.TagDirectoryGroups},
		Summary:     "Get a group",
		Errors:      []int{http.StatusNotFound},
	}), func(ctx context.Context, input *groupGetInput) (*groupOutput, error) {
		group, err := groupStore.GetGroupByID(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-group", groupResource, err, "group_id", input.ID)
		}
		return &groupOutput{Body: *group}, nil
	})
}
