// Package httpapi exposes machine-credential administration to the browser.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/woodleighschool/goodies/auth/authz"
	authhuma "github.com/woodleighschool/goodies/auth/huma"
	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/appkey"
	"github.com/woodleighschool/woodgate/internal/rbac"
)

func RegisterAPI(routes api.AppRoutes, store *appkey.Store, authorizer authhuma.Authorizer, logger *slog.Logger) {
	register(routes.Protected, store, authorizer, logger)
}
func RegisterOpenAPI(routes api.AppRoutes) { register(routes.Protected, nil, nil, nil) }

type listInput struct{ api.ListQueryInput }
type listOutput struct{ Body api.Page[appkey.Key] }
type itemInput struct {
	ID int64 `path:"id" minimum:"1"`
}
type itemOutput struct{ Body appkey.Key }
type createInput struct{ Body appkey.Mutation }
type createOutput struct{ Body appkey.CreatedKey }
type updateInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body appkey.Mutation
}

func register(routes huma.API, store *appkey.Store, authorizer authhuma.Authorizer, logger *slog.Logger) {
	operation := func(id, method, path, summary string, access authz.Access) huma.Operation {
		return authhuma.Require(routes, authorizer, logger, rbac.ResourceAppKeys, access, huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{api.TagAppKeys}})
	}
	huma.Register(routes, operation("list-app-key-locations", http.MethodGet, "/api/app-keys/locations", "List app key location choices", authz.View), func(ctx context.Context, input *listInput) (*locationsOutput, error) {
		items, count, err := store.Locations(ctx, input.Params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-app-key-locations", "location", err)
		}
		return &locationsOutput{Body: api.Page[appkey.LocationSummary]{Items: items, Count: count}}, nil
	})

	huma.Register(routes, operation("list-app-keys", http.MethodGet, "/api/app-keys", "List app keys", authz.View), func(ctx context.Context, input *listInput) (*listOutput, error) {
		items, count, err := store.List(ctx, input.Params())
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-app-keys", "app key", err)
		}
		return &listOutput{Body: api.Page[appkey.Key]{Items: items, Count: count}}, nil
	})
	huma.Register(routes, operation("get-app-key", http.MethodGet, "/api/app-keys/{id}", "Get an app key", authz.View), func(ctx context.Context, input *itemInput) (*itemOutput, error) {
		item, err := store.Get(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-app-key", "app key", err)
		}
		return &itemOutput{Body: *item}, nil
	})
	createOperation := operation("create-app-key", http.MethodPost, "/api/app-keys", "Create an app key", authz.Edit)
	createOperation.DefaultStatus = http.StatusCreated
	huma.Register(routes, createOperation, func(ctx context.Context, input *createInput) (*createOutput, error) {
		item, err := store.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-app-key", "app key", err)
		}
		return &createOutput{Body: *item}, nil
	})
	huma.Register(routes, operation("update-app-key", http.MethodPut, "/api/app-keys/{id}", "Update an app key", authz.Edit), func(ctx context.Context, input *updateInput) (*itemOutput, error) {
		item, err := store.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-app-key", "app key", err)
		}
		return &itemOutput{Body: *item}, nil
	})
	deleteOperation := operation("delete-app-key", http.MethodDelete, "/api/app-keys/{id}", "Delete an app key", authz.Edit)
	deleteOperation.DefaultStatus = http.StatusNoContent
	huma.Register(routes, deleteOperation, func(ctx context.Context, input *itemInput) (*struct{}, error) {
		if err := store.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-app-key", "app key", err)
		}
		return nil, nil
	})
}

type locationsOutput struct {
	Body api.Page[appkey.LocationSummary]
}
