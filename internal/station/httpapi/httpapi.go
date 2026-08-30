// Package httpapi exposes Station administration to the browser application.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/authorization"
	"github.com/woodleighschool/woodgate/internal/station"
)

type listInput struct {
	api.ListQueryInput

	LocationID int64                   `query:"location_id,omitempty" minimum:"1"`
	Enabled    api.OptionalParam[bool] `query:"enabled,omitempty"`
}

type listOutput struct{ Body api.Page[station.Station] }
type output struct{ Body station.Station }
type secretOutput struct{ Body station.StationSecret }
type createInput struct{ Body station.StationMutation }
type idInput struct {
	ID int64 `path:"id"`
}
type putInput struct {
	ID   int64 `path:"id"`
	Body station.StationMutation
}

// RegisterAPI mounts Station administration endpoints.
func RegisterAPI(routes api.AppRoutes, service *station.Service, authorizer authorization.Authorizer, logger *slog.Logger) {
	register(routes.Protected, service, authorizer, logger)
}

// RegisterOpenAPI documents Station administration endpoints.
func RegisterOpenAPI(routes api.AppRoutes) { register(routes.Protected, nil, nil, nil) }

func register(humaAPI huma.API, service *station.Service, authorizer authorization.Authorizer, logger *slog.Logger) {
	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.View, huma.Operation{
		OperationID: "list-stations", Method: http.MethodGet, Path: "/api/stations", Tags: []string{api.TagStations}, Summary: "List Stations",
	}), func(ctx context.Context, input *listInput) (*listOutput, error) {
		items, count, err := service.List(ctx, station.StationListParams{ListParams: input.Params(), LocationID: input.LocationID, Enabled: input.Enabled.Pointer()})
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "list-stations", "Station", err)
		}
		return &listOutput{Body: api.Page[station.Station]{Items: items, Count: count}}, nil
	})

	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.Edit, huma.Operation{
		OperationID: "create-station", Method: http.MethodPost, Path: "/api/stations", Tags: []string{api.TagStations}, Summary: "Create a Station", DefaultStatus: http.StatusCreated,
	}), func(ctx context.Context, input *createInput) (*secretOutput, error) {
		item, err := service.Create(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "create-station", "Station", err)
		}
		return &secretOutput{Body: *item}, nil
	})

	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.View, huma.Operation{
		OperationID: "get-station", Method: http.MethodGet, Path: "/api/stations/{id}", Tags: []string{api.TagStations}, Summary: "Get a Station",
	}), func(ctx context.Context, input *idInput) (*output, error) {
		item, err := service.Get(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "get-station", "Station", err, "station_id", input.ID)
		}
		return &output{Body: *item}, nil
	})

	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.Edit, huma.Operation{
		OperationID: "update-station", Method: http.MethodPut, Path: "/api/stations/{id}", Tags: []string{api.TagStations}, Summary: "Update a Station",
	}), func(ctx context.Context, input *putInput) (*output, error) {
		item, err := service.Update(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "update-station", "Station", err, "station_id", input.ID)
		}
		return &output{Body: *item}, nil
	})

	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.Edit, huma.Operation{
		OperationID: "delete-station", Method: http.MethodDelete, Path: "/api/stations/{id}", Tags: []string{api.TagStations}, Summary: "Delete a Station", DefaultStatus: http.StatusNoContent,
	}), func(ctx context.Context, input *idInput) (*struct{}, error) {
		if err := service.Delete(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, logger, "delete-station", "Station", err, "station_id", input.ID)
		}
		return &struct{}{}, nil
	})

	huma.Register(humaAPI, authorization.Require(humaAPI, authorizer, "stations", authorization.Edit, huma.Operation{
		OperationID: "rotate-station-secret", Method: http.MethodPost, Path: "/api/stations/{id}/key", Tags: []string{api.TagStations}, Summary: "Rotate a Station key",
	}), func(ctx context.Context, input *idInput) (*secretOutput, error) {
		item, err := service.RotateSecret(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, logger, "rotate-station-secret", "Station", err, "station_id", input.ID)
		}
		return &secretOutput{Body: *item}, nil
	})
}
