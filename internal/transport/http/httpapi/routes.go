package httpapi

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type response[T any] struct{ Body T }
type ItemInput struct {
	ID uuid.UUID `path:"id"`
}
type BodyInput[T any] struct{ Body bodyValue[T] }
type patchInput[T any] struct {
	ItemInput
	BodyInput[T]
}

// bodyValue keeps the v1 JSON decoding contract while Huma owns the wire schema.
type bodyValue[T any] struct{ Value T }

func (body *bodyValue[T]) UnmarshalJSON(data []byte) error {
	return decodeJSONBody(bytes.NewReader(data), &body.Value)
}
func (body bodyValue[T]) Schema(registry huma.Registry) *huma.Schema {
	return registry.Schema(reflect.TypeFor[T](), true, "")
}

type parameter[T any] struct{ Value *T }

func (param *parameter[T]) UnmarshalText(text []byte) error {
	var value T
	switch dest := any(&value).(type) {
	case *int32:
		parsed, err := strconv.ParseInt(string(text), 10, 32)
		if err != nil {
			return err
		}
		*dest = int32(parsed)
	case *bool:
		parsed, err := strconv.ParseBool(string(text))
		if err != nil {
			return err
		}
		*dest = parsed
	case *string:
		*dest = string(text)
	default:
		if dest, ok := any(&value).(encoding.TextUnmarshaler); ok {
			if err := dest.UnmarshalText(text); err != nil {
				return err
			}
		} else if reflect.TypeFor[T]().Kind() == reflect.String {
			reflect.ValueOf(&value).Elem().SetString(string(text))
		} else {
			return fmt.Errorf("unsupported parameter %T", value)
		}
	}
	param.Value = &value
	return nil
}
func (param parameter[T]) Schema(registry huma.Registry) *huma.Schema {
	return registry.Schema(reflect.TypeFor[T](), false, "")
}

// Huma omits empty query values. V1 distinguishes an omitted parameter from an
// explicitly empty value and rejects repeated scalar parameters.
func (param *parameter[T]) Resolve(ctx huma.Context, path *huma.PathBuffer) []error {
	name := strings.TrimPrefix(path.String(), "query.")
	url := ctx.URL()
	values, present := url.Query()[name]
	if !present {
		return nil
	}
	if len(values) != 1 {
		return []error{fmt.Errorf("invalid query parameter")}
	}
	if values[0] == "" {
		if err := param.UnmarshalText(nil); err != nil {
			return []error{err}
		}
	}
	return nil
}

type RequestInput struct {
	writer  http.ResponseWriter
	request *http.Request
}

func (input *RequestInput) Resolve(ctx huma.Context) []error {
	input.request, input.writer = humachi.Unwrap(ctx)
	return nil
}

type itemRequestInput struct {
	ItemInput
	RequestInput
}

func (problem *Problem) Error() string             { return problem.Detail }
func (problem *Problem) GetStatus() int            { return int(problem.Status) }
func (problem *Problem) ContentType(string) string { return "application/json" }

//nolint:gochecknoinits // Huma requires its error factory before registering any routes.
func init() {
	//nolint:reassign // Huma exposes the error factory as a package variable.
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		detail := "Request parameters are invalid."
		if message == "request body is required" {
			detail = message
		}
		if status == http.StatusRequestEntityTooLarge {
			detail = "request body is invalid"
		}
		for _, err := range errs {
			if problem, ok := errors.AsType[*Problem](err); ok {
				return problem
			}
			item, ok := errors.AsType[*huma.ErrorDetail](err)
			if !ok || item.Location != "body" {
				detail = "Request parameters are invalid."
				break
			}
			detail = item.Message
		}
		return newProblem(http.StatusBadRequest, problemSpec{
			Type: "urn:woodgate:problem:invalid-request", Title: "Invalid request", Code: "invalid_request", Detail: detail,
		})
	}
}

// RegisterRoutes serves the API under the caller's /api/v1 router.
func (handler *Server) RegisterRoutes(router chi.Router) huma.API {
	config := huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info:    &huma.Info{Title: "WoodGate API", Version: "0.1.0"},
			Servers: []*huma.Server{{URL: "/api/v1"}},
			Components: &huma.Components{SecuritySchemes: map[string]*huma.SecurityScheme{
				"apiKeyAuth":  {Type: "apiKey", In: "header", Name: "X-API-Key"},
				"sessionAuth": {Type: "apiKey", In: "cookie", Name: "woodgate_session"},
			}},
			Security: []map[string][]string{{"apiKeyAuth": {}}, {"sessionAuth": {}}},
		},
		Formats: map[string]huma.Format{"application/json": {
			Marshal: func(writer io.Writer, value any) error { return json.NewEncoder(writer).Encode(value) },
			Unmarshal: func(data []byte, value any) error {
				if decoder, ok := value.(json.Unmarshaler); ok {
					return decoder.UnmarshalJSON(data)
				}
				return json.Unmarshal(data, value)
			},
		}},
		DefaultFormat: "application/json",
	}
	api := jsonAPI{humachi.New(router, config)}
	api.OpenAPI().Components.Schemas.RegisterTypeAlias(reflect.TypeFor[uuid.UUID](), reflect.TypeFor[uuidSchema]())
	register(api, "createAPIKey", "POST", "/api-keys", http.StatusCreated, handler.createAPIKey)
	register(api, "createAsset", "POST", "/assets", http.StatusCreated, handler.createAsset)
	register(api, "createCheckin", "POST", "/checkins", http.StatusCreated, handler.createCheckin)
	register(api, "createLocation", "POST", "/locations", http.StatusCreated, handler.createLocation)
	register(api, "deleteAPIKey", "DELETE", "/api-keys/{id}", http.StatusNoContent, handler.deleteAPIKey)
	register(api, "deleteAsset", "DELETE", "/assets/{id}", http.StatusNoContent, handler.deleteAsset)
	register(api, "deleteLocation", "DELETE", "/locations/{id}", http.StatusNoContent, handler.deleteLocation)
	register(api, "getAPIKey", "GET", "/api-keys/{id}", http.StatusOK, handler.getAPIKey)
	register(api, "getAsset", "GET", "/assets/{id}", http.StatusOK, handler.getAsset)
	register(api, "getCheckin", "GET", "/checkins/{id}", http.StatusOK, handler.getCheckin)
	register(api, "getGroup", "GET", "/groups/{id}", http.StatusOK, handler.getGroup)
	register(api, "getGroupMembership", "GET", "/group-memberships/{id}", http.StatusOK, handler.getGroupMembership)
	register(api, "getLocation", "GET", "/locations/{id}", http.StatusOK, handler.getLocation)
	register(api, "getUser", "GET", "/users/{id}", http.StatusOK, handler.getUser)
	register(api, "listAPIKeys", "GET", "/api-keys", http.StatusOK, handler.listAPIKeys)
	register(api, "listAssets", "GET", "/assets", http.StatusOK, handler.listAssets)
	register(api, "listCheckinDepartments", "GET", "/checkins/departments", http.StatusOK, handler.listCheckinDepartments)
	register(api, "listCheckins", "GET", "/checkins", http.StatusOK, handler.listCheckins)
	register(api, "listGroupMemberships", "GET", "/group-memberships", http.StatusOK, handler.listGroupMemberships)
	register(api, "listGroups", "GET", "/groups", http.StatusOK, handler.listGroups)
	register(api, "listLocations", "GET", "/locations", http.StatusOK, handler.listLocations)
	register(api, "listUsers", "GET", "/users", http.StatusOK, handler.listUsers)
	register(api, "patchAPIKey", "PATCH", "/api-keys/{id}", http.StatusOK, handler.patchAPIKey)
	register(api, "patchAsset", "PATCH", "/assets/{id}", http.StatusOK, handler.patchAsset)
	register(api, "patchLocation", "PATCH", "/locations/{id}", http.StatusOK, handler.patchLocation)
	register(api, "patchUser", "PATCH", "/users/{id}", http.StatusOK, handler.patchUser)
	register(api, "getAssetContent", "GET", "/assets/{id}/content", http.StatusOK, func(_ context.Context, input *itemRequestInput) (*binaryResponse, error) {
		return &binaryResponse{Body: func(huma.Context) { handler.getAssetContent(input.writer, input.request, input.ID) }}, nil
	})
	documentMultipart[AssetCreateRequest](api, "POST", "/assets")
	documentMultipart[AssetUpdateRequest](api, "PATCH", "/assets/{id}")
	documentMultipart[CheckinCreateRequest](api, "POST", "/checkins")
	content := api.OpenAPI().Paths["/assets/{id}/content"].Get.Responses["200"]
	content.Content = map[string]*huma.MediaType{"application/octet-stream": {Schema: &huma.Schema{Type: "string", Format: "binary"}}}
	return api
}

// V1 accepts JSON independently of the client's Content-Type header.
type jsonAPI struct{ huma.API }

func (api jsonAPI) Unmarshal(_ string, data []byte, value any) error {
	return api.API.Unmarshal("application/json", data, value)
}

type binaryResponse struct{ Body func(huma.Context) }

func register[I, O any](api huma.API, id, method, path string, status int, handler func(context.Context, *I) (*O, error)) {
	schema := api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[Problem](), true, "Problem")
	responses := map[string]*huma.Response{}
	for _, status := range []int{400, 401, 403, 404, 409, 422, 500} {
		responses[strconv.Itoa(status)] = &huma.Response{Description: http.StatusText(status), Content: map[string]*huma.MediaType{"application/json": {Schema: schema}}}
	}
	huma.Register(api, huma.Operation{
		OperationID: id, Method: method, Path: path, DefaultStatus: status,
		SkipValidateParams: true, SkipValidateBody: true,
		// Huma rejects bodies that exactly reach its read limit. The decoder
		// retains the v1 limit while allowing a complete object at that boundary.
		MaxBodyBytes: maxJSONBodyBytes + 1, BodyReadTimeout: -1,
		Responses: responses,
	}, handler)
}

// Upload authorization and location-specific validation run before reading files.
// The multipart models document the existing parser without Huma consuming it first.
func documentMultipart[T any](api huma.API, method, path string) {
	item := api.OpenAPI().Paths[path]
	operation := item.Post
	if method == "PATCH" {
		operation = item.Patch
	}
	operation.RequestBody = &huma.RequestBody{Required: true, Content: map[string]*huma.MediaType{
		"multipart/form-data": {Schema: api.OpenAPI().Components.Schemas.Schema(reflect.TypeFor[T](), true, "")},
	}}
}

// OpenAPI returns the schema from the same registrations used by the service.
func OpenAPI() *huma.OpenAPI { return New(nil, nil).RegisterRoutes(chi.NewRouter()).OpenAPI() }

type ListParams struct {
	Limit  parameter[int32]  `query:"limit" minimum:"1"`
	Offset parameter[int32]  `query:"offset" minimum:"0"`
	Search parameter[string] `query:"search"`
	Sort   parameter[string] `query:"sort"`
	Order  parameter[string] `query:"order" enum:"asc,desc"`
}

type ListUsersParams struct {
	ListParams

	LocationID parameter[uuid.UUID] `query:"location_id"`
}

type ListGroupsParams struct {
	ListParams
}

type ListGroupMembershipsParams struct {
	ListParams

	GroupID parameter[uuid.UUID] `query:"group_id"`
	UserID  parameter[uuid.UUID] `query:"user_id"`
}

type ListLocationsParams struct {
	ListParams

	Enabled parameter[bool] `query:"enabled"`
}

type ListAssetsParams struct {
	ListParams

	Type parameter[AssetType] `query:"type" enum:"asset,photo"`
}

type ListCheckinsParams struct {
	ListParams

	LocationID  parameter[uuid.UUID]        `query:"location_id"`
	UserID      parameter[uuid.UUID]        `query:"user_id"`
	Direction   parameter[CheckinDirection] `query:"direction" enum:"check_in,check_out"`
	Department  parameter[string]           `query:"department"`
	CreatedFrom parameter[time.Time]        `query:"created_from"`
	CreatedTo   parameter[time.Time]        `query:"created_to"`
}

type ListAPIKeysParams struct {
	ListParams
}

type uuidSchema struct{}

func (uuidSchema) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{Type: "string", Format: "uuid"}
}
