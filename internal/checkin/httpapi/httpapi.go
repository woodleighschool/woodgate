// Package httpapi exposes the check-in domain to the browser application.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/api/ctxkeys"
	apimiddleware "github.com/woodleighschool/woodgate/internal/api/middleware"
	"github.com/woodleighschool/woodgate/internal/authorization"
	"github.com/woodleighschool/woodgate/internal/checkin"
	"github.com/woodleighschool/woodgate/internal/listing"
	"github.com/woodleighschool/woodgate/internal/storage"
)

const (
	locationPath           = "/api/locations/{id}"
	locationBackgroundPath = "/api/locations/backgrounds"
	locationLogoPath       = "/api/locations/logos"
	checkinPath            = "/api/checkins/{id}"
)

// Dependencies are the services used by the browser check-in API.
type Dependencies struct {
	Service       *checkin.Service
	Authorizer    authorization.Authorizer
	Authenticator apimiddleware.Authenticator
	Logger        *slog.Logger
}

// RegisterAPI mounts browser check-in endpoints.
func RegisterAPI(routes api.AppRoutes, deps Dependencies) {
	registerOperations(routes.Protected, deps)
	locationContent := routes.Transfers.With(
		apimiddleware.RequireHTTPAuth(deps.Authenticator),
		authorization.RequireHTTP(deps.Authorizer, "locations", authorization.View),
	)
	locationContent.Get(locationBackgroundPath+"/{id}/content", deps.backgroundContent)
	locationContent.Get(locationLogoPath+"/{id}/content", deps.logoContent)
	checkinContent := routes.Transfers.With(
		apimiddleware.RequireHTTPAuth(deps.Authenticator),
		authorization.RequireHTTP(deps.Authorizer, "checkins", authorization.View),
	)
	checkinContent.Get(checkinPath+"/photo", deps.checkinPhoto)
}

// RegisterOpenAPI documents browser check-in endpoints without runtime services.
func RegisterOpenAPI(routes api.AppRoutes) { registerOperations(routes.Protected, Dependencies{}) }

type locationListInput struct {
	api.ListQueryInput

	Enabled api.OptionalParam[bool] `query:"enabled,omitempty"`
}
type locationListOutput struct{ Body api.Page[checkin.Location] }
type locationOutput struct{ Body checkin.Location }
type locationCreateInput struct{ Body checkin.LocationMutation }
type locationIDInput struct {
	ID int64 `path:"id"`
}
type locationPutInput struct {
	ID   int64 `path:"id"`
	Body checkin.LocationMutation
}

type checkinListInput struct {
	api.ListQueryInput

	LocationID    int64                        `query:"location_id,omitempty" minimum:"1"`
	UserID        int64                        `query:"user_id,omitempty" minimum:"1"`
	Direction     checkin.Direction            `query:"direction,omitempty" enum:"check_in,check_out"`
	Department    string                       `query:"department,omitempty"`
	CreatedFrom   api.OptionalParam[time.Time] `query:"created_from,omitempty"`
	CreatedBefore api.OptionalParam[time.Time] `query:"created_before,omitempty"`
}
type checkinListOutput struct{ Body api.Page[checkin.Checkin] }
type checkinOutput struct{ Body checkin.Checkin }
type checkinCreateInput struct{ Body checkin.CheckinCreate }
type checkinIDInput struct {
	ID int64 `path:"id"`
}

type attachmentListInput struct{ api.ListQueryInput }
type attachmentListOutput struct {
	Body api.Page[attachmentObjectView]
}
type attachmentUploadInput struct {
	Body struct {
		Filename string `json:"filename" minLength:"1"`
	}
}
type attachmentMutation struct {
	ObjectID int64 `json:"object_id" minimum:"1"`
}
type attachmentPutInput struct {
	ID   int64 `path:"id"`
	Body attachmentMutation
}
type directUploadAction struct {
	Strategy string            `json:"strategy" enum:"direct-put"`
	URL      string            `json:"url"`
	Method   string            `json:"method" enum:"PUT"`
	Headers  map[string]string `json:"headers,omitempty"`
}
type directUploadTarget struct {
	ObjectID int64              `json:"object_id"`
	Upload   directUploadAction `json:"upload"`
}
type directUploadOutput struct{ Body directUploadTarget }
type attachmentObjectView struct {
	ID          int64   `json:"id"`
	Filename    string  `json:"filename"`
	ContentType string  `json:"content_type"`
	SizeBytes   *int64  `json:"size_bytes,omitempty"`
	SHA256      *string `json:"sha256,omitempty"`
	ContentURL  string  `json:"content_url"`
}
type attachmentObjectOutput struct{ Body attachmentObjectView }

func registerOperations(routes huma.API, deps Dependencies) {
	registerLocations(routes, deps)
	registerCheckins(routes, deps)
	registerLocationAttachments(routes, deps)
}

func registerLocations(routes huma.API, deps Dependencies) {
	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.View, huma.Operation{
		OperationID: "list-locations", Method: http.MethodGet, Path: "/api/locations", Tags: []string{api.TagLocations}, Summary: "List locations",
	}), func(ctx context.Context, input *locationListInput) (*locationListOutput, error) {
		items, count, err := deps.Service.ListLocations(ctx, checkin.LocationListParams{ListParams: input.Params(), Enabled: input.Enabled.Pointer()})
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "list-locations", "location", err)
		}
		return &locationListOutput{Body: api.Page[checkin.Location]{Items: items, Count: count}}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.Edit, huma.Operation{
		OperationID: "create-location", Method: http.MethodPost, Path: "/api/locations", Tags: []string{api.TagLocations}, Summary: "Create a location", DefaultStatus: http.StatusCreated,
	}), func(ctx context.Context, input *locationCreateInput) (*locationOutput, error) {
		item, err := deps.Service.CreateLocation(ctx, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "create-location", "location", err)
		}
		return &locationOutput{Body: *item}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.View, huma.Operation{
		OperationID: "get-location", Method: http.MethodGet, Path: locationPath, Tags: []string{api.TagLocations}, Summary: "Get a location",
	}), func(ctx context.Context, input *locationIDInput) (*locationOutput, error) {
		item, err := deps.Service.GetLocation(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "get-location", "location", err, "location_id", input.ID)
		}
		return &locationOutput{Body: *item}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.Edit, huma.Operation{
		OperationID: "update-location", Method: http.MethodPut, Path: locationPath, Tags: []string{api.TagLocations}, Summary: "Update a location",
	}), func(ctx context.Context, input *locationPutInput) (*locationOutput, error) {
		item, err := deps.Service.UpdateLocation(ctx, input.ID, input.Body)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "update-location", "location", err, "location_id", input.ID)
		}
		return &locationOutput{Body: *item}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.Edit, huma.Operation{
		OperationID: "delete-location", Method: http.MethodDelete, Path: locationPath, Tags: []string{api.TagLocations}, Summary: "Delete a location", DefaultStatus: http.StatusNoContent,
	}), func(ctx context.Context, input *locationIDInput) (*struct{}, error) {
		if err := deps.Service.DeleteLocation(ctx, input.ID); err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "delete-location", "location", err, "location_id", input.ID)
		}
		return &struct{}{}, nil
	})
}

func registerCheckins(routes huma.API, deps Dependencies) {
	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "checkins", authorization.View, huma.Operation{OperationID: "list-checkins", Method: http.MethodGet, Path: "/api/checkins", Tags: []string{api.TagCheckins}, Summary: "List check-ins"}), func(ctx context.Context, input *checkinListInput) (*checkinListOutput, error) {
		items, count, err := deps.Service.ListCheckins(ctx, checkin.CheckinListParams{ListParams: input.Params(), LocationID: input.LocationID, UserID: input.UserID, Direction: input.Direction, Department: input.Department, CreatedFrom: input.CreatedFrom.Pointer(), CreatedBefore: input.CreatedBefore.Pointer()})
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "list-checkins", "check-in", err)
		}
		return &checkinListOutput{Body: api.Page[checkin.Checkin]{Items: items, Count: count}}, nil
	})
	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "checkins", authorization.View, huma.Operation{OperationID: "get-checkin", Method: http.MethodGet, Path: checkinPath, Tags: []string{api.TagCheckins}, Summary: "Get a check-in"}), func(ctx context.Context, input *checkinIDInput) (*checkinOutput, error) {
		item, err := deps.Service.GetCheckin(ctx, input.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "get-checkin", "check-in", err, "checkin_id", input.ID)
		}
		return &checkinOutput{Body: *item}, nil
	})
	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "checkins", authorization.Edit, huma.Operation{OperationID: "create-checkin", Method: http.MethodPost, Path: "/api/checkins", Tags: []string{api.TagCheckins}, Summary: "Create a check-in", DefaultStatus: http.StatusCreated}), func(ctx context.Context, input *checkinCreateInput) (*checkinOutput, error) {
		user, err := ctxkeys.RequireUser(ctx)
		if err != nil {
			return nil, err
		}
		item, err := deps.Service.CreateCheckin(ctx, input.Body, user.ID)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, "create-checkin", "check-in", err)
		}
		return &checkinOutput{Body: *item}, nil
	})
}

func registerLocationAttachments(routes huma.API, deps Dependencies) {
	registerLocationAttachment(routes, deps, locationAttachmentRegistration{
		kind: "background", path: locationBackgroundPath,
		listOperation: "list-location-backgrounds", uploadOperation: "create-location-background-upload",
		setOperation: "set-location-background", list: deps.Service.ListLocationBackgrounds,
		begin: deps.Service.BeginLocationBackgroundUpload, set: deps.Service.SetLocationBackground,
	})
	registerLocationAttachment(routes, deps, locationAttachmentRegistration{
		kind: "logo", path: locationLogoPath,
		listOperation: "list-location-logos", uploadOperation: "create-location-logo-upload",
		setOperation: "set-location-logo", list: deps.Service.ListLocationLogos,
		begin: deps.Service.BeginLocationLogoUpload, set: deps.Service.SetLocationLogo,
	})
}

type locationAttachmentRegistration struct {
	kind            string
	path            string
	listOperation   string
	uploadOperation string
	setOperation    string
	list            func(context.Context, listing.Params) ([]storage.Object, int, error)
	begin           func(context.Context, string) (*storage.Object, storage.UploadTarget, error)
	set             func(context.Context, int64, int64) (*storage.Object, error)
}

func registerLocationAttachment(routes huma.API, deps Dependencies, registration locationAttachmentRegistration) {
	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.View, huma.Operation{
		OperationID: registration.listOperation, Method: http.MethodGet, Path: registration.path,
		Tags: []string{api.TagLocations}, Summary: "List location " + registration.kind + "s",
	}), func(ctx context.Context, input *attachmentListInput) (*attachmentListOutput, error) {
		objects, count, err := registration.list(ctx, input.Params())
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, registration.listOperation, "location "+registration.kind, err)
		}
		items := make([]attachmentObjectView, len(objects))
		for i, object := range objects {
			items[i] = newAttachmentObjectView(object, registration.path)
		}
		return &attachmentListOutput{Body: api.Page[attachmentObjectView]{Items: items, Count: count}}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.Edit, huma.Operation{
		OperationID: registration.uploadOperation, Method: http.MethodPost, Path: registration.path,
		Tags: []string{api.TagLocations}, Summary: "Create a location " + registration.kind + " upload", DefaultStatus: http.StatusCreated,
	}), func(ctx context.Context, input *attachmentUploadInput) (*directUploadOutput, error) {
		object, target, err := registration.begin(ctx, input.Body.Filename)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, registration.uploadOperation, "location "+registration.kind+" upload", err)
		}
		return &directUploadOutput{Body: directUploadTarget{ObjectID: object.ID, Upload: directUploadAction{
			Strategy: "direct-put", URL: target.URL, Method: target.Method, Headers: target.Headers,
		}}}, nil
	})

	huma.Register(routes, authorization.Require(routes, deps.Authorizer, "locations", authorization.Edit, huma.Operation{
		OperationID: registration.setOperation, Method: http.MethodPut, Path: locationPath + "/" + registration.kind,
		Tags: []string{api.TagLocations}, Summary: "Set a location " + registration.kind,
	}), func(ctx context.Context, input *attachmentPutInput) (*attachmentObjectOutput, error) {
		object, err := registration.set(ctx, input.ID, input.Body.ObjectID)
		if err != nil {
			return nil, api.ResourceError(ctx, deps.Logger, registration.setOperation, "location "+registration.kind, err,
				"location_id", input.ID, "object_id", input.Body.ObjectID)
		}
		return &attachmentObjectOutput{Body: newAttachmentObjectView(*object, registration.path)}, nil
	})
}

func newAttachmentObjectView(object storage.Object, basePath string) attachmentObjectView {
	return attachmentObjectView{ID: object.ID, Filename: object.Filename, ContentType: object.ContentType,
		SizeBytes: object.SizeBytes, SHA256: object.SHA256,
		ContentURL: basePath + "/" + strconv.FormatInt(object.ID, 10) + "/content"}
}

func (deps Dependencies) backgroundContent(w http.ResponseWriter, r *http.Request) {
	deps.attachmentContent(w, r, "background", deps.Service.DeliverLocationBackground)
}

func (deps Dependencies) logoContent(w http.ResponseWriter, r *http.Request) {
	deps.attachmentContent(w, r, "logo", deps.Service.DeliverLocationLogo)
}

func (deps Dependencies) checkinPhoto(w http.ResponseWriter, r *http.Request) {
	deps.attachmentContent(w, r, "check-in photo", deps.Service.DeliverCheckinPhoto)
}

func (deps Dependencies) attachmentContent(
	w http.ResponseWriter,
	r *http.Request,
	label string,
	deliver func(http.ResponseWriter, *http.Request, int64) error,
) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid attachment id", http.StatusBadRequest)
		return
	}
	if err := deliver(w, r, id); err != nil {
		mapped := api.ResourceMutationError(label, err)
		if statusErr, ok := errors.AsType[huma.StatusError](mapped); ok {
			http.Error(w, http.StatusText(statusErr.GetStatus()), statusErr.GetStatus())
			return
		}
		deps.Logger.ErrorContext(r.Context(), "attachment delivery failed", "attachment", label, "id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
