package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
	pgutil "github.com/woodleighschool/woodgate/internal/store/postgres/shared"
)

const (
	maxJSONBodyBytes      = 1 << 20
	multipartBodyMultiple = 10
	apiKeyBytes           = 32
	apiKeyPrefixLen       = 12
)

type Server struct {
	admin      AdminService
	authorizer authz.Authorizer
}

type AdminService interface {
	ListUsers(context.Context, domain.UserListOptions) ([]domain.User, int32, error)
	GetUser(context.Context, uuid.UUID) (domain.User, error)
	UpdateUserAccess(context.Context, uuid.UUID, bool, []domain.PermissionGrant) (domain.User, error)
	ListGroups(context.Context, domain.GroupListOptions) ([]domain.Group, int32, error)
	GetGroup(context.Context, uuid.UUID) (domain.Group, error)
	ListGroupMemberships(context.Context, domain.GroupMembershipListOptions) ([]domain.GroupMembership, int32, error)
	GetGroupMembership(context.Context, uuid.UUID) (domain.GroupMembership, error)
	ListAssets(context.Context, domain.AssetListOptions) ([]domain.Asset, int32, error)
	CreateAsset(context.Context, *string, []byte) (domain.Asset, error)
	GetAsset(context.Context, uuid.UUID) (domain.Asset, error)
	GetAssetFile(context.Context, uuid.UUID) (domain.Asset, string, error)
	UpdateAsset(context.Context, uuid.UUID, *string, []byte) (domain.Asset, error)
	DeleteAsset(context.Context, uuid.UUID) error
	ListLocations(context.Context, domain.LocationListOptions) ([]domain.Location, int32, error)
	CreateLocation(
		context.Context,
		string,
		string,
		bool,
		bool,
		bool,
		*uuid.UUID,
		*uuid.UUID,
		[]uuid.UUID,
	) (domain.Location, error)
	GetLocation(context.Context, uuid.UUID) (domain.Location, error)
	UpdateLocation(
		context.Context,
		uuid.UUID,
		string,
		string,
		bool,
		bool,
		bool,
		*uuid.UUID,
		*uuid.UUID,
		[]uuid.UUID,
	) (domain.Location, error)
	DeleteLocation(context.Context, uuid.UUID) error
	ListCheckins(context.Context, domain.CheckinListOptions, []uuid.UUID) ([]domain.Checkin, int32, error)
	CreateCheckin(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		domain.CheckinDirection,
		string,
		[]byte,
		domain.PermissionSubjectKind,
		uuid.UUID,
	) (domain.Checkin, error)
	GetCheckin(context.Context, uuid.UUID, []uuid.UUID) (domain.Checkin, error)
	ListAPIKeys(context.Context, domain.APIKeyListOptions) ([]domain.APIKey, int32, error)
	CreateAPIKey(context.Context, string, string, string, *time.Time) (domain.APIKey, error)
	GetAPIKey(context.Context, uuid.UUID) (domain.APIKey, error)
	UpdateAPIKeyAccess(context.Context, uuid.UUID, bool, []domain.PermissionGrant) (domain.APIKey, error)
	DeleteAPIKey(context.Context, uuid.UUID) error
}

func New(adminService AdminService, authorizer authz.Authorizer) *Server {
	return &Server{admin: adminService, authorizer: authorizer}
}

func (handler *Server) RegisterRoutes(router chi.Router) {
	_ = HandlerWithOptions(handler, ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(writer http.ResponseWriter, _ *http.Request, _ error) {
			writeProblem(writer, http.StatusBadRequest, problemSpec{
				Type:   "urn:woodgate:problem:invalid-request",
				Title:  "Invalid request",
				Code:   "invalid_request",
				Detail: "Request parameters are invalid.",
			})
		},
	})
}

func parsePagination(limit *int32, offset *int32) (int32, int32, error) {
	resolvedLimit := int32(0)
	resolvedOffset := int32(0)
	if limit != nil {
		resolvedLimit = *limit
	}
	if offset != nil {
		resolvedOffset = *offset
	}

	switch {
	case limit != nil && resolvedLimit < 1:
		return 0, 0, badRequestError("limit must be >= 1")
	case resolvedOffset < 0:
		return 0, 0, badRequestError("offset must be >= 0")
	default:
		return resolvedLimit, resolvedOffset, nil
	}
}

func optionalString[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func parseListOptions[S ~string, T ~string, O ~string](
	limit *int32,
	offset *int32,
	search *T,
	sort *S,
	order *O,
) (domain.ListOptions, error) {
	resolvedLimit, resolvedOffset, err := parsePagination(limit, offset)
	if err != nil {
		return domain.ListOptions{}, err
	}

	sortValue := ""
	if sort != nil {
		sortValue = strings.TrimSpace(string(*sort))
	}

	return domain.ListOptions{
		Limit:  resolvedLimit,
		Offset: resolvedOffset,
		Search: optionalString(search),
		Sort:   sortValue,
		Order:  optionalString(order),
	}, nil
}

func decodeJSONBody(request *http.Request, dst any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, maxJSONBodyBytes))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return badRequestError("request body is required")
		}
		return badRequestError("request body is invalid")
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return badRequestError("request body must contain a single JSON object")
	}

	return nil
}

func writeJSON(writer http.ResponseWriter, statusCode int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

type problemSpec struct {
	Type        string
	Title       string
	Code        string
	Detail      string
	FieldErrors []domain.FieldError
}

func writeProblem(writer http.ResponseWriter, statusCode int, spec problemSpec) {
	problem := Problem{
		Type:   spec.Type,
		Title:  spec.Title,
		Status: safeInt32(statusCode),
		Detail: spec.Detail,
		Code:   spec.Code,
	}
	if len(spec.FieldErrors) > 0 {
		fieldErrors := make([]FieldError, 0, len(spec.FieldErrors))
		for _, fieldErr := range spec.FieldErrors {
			mapped := FieldError{
				Field:   fieldErr.Field,
				Message: fieldErr.Message,
			}
			if fieldErr.Code != "" {
				code := fieldErr.Code
				mapped.Code = &code
			}
			fieldErrors = append(fieldErrors, mapped)
		}
		problem.FieldErrors = &fieldErrors
	}
	writeJSON(writer, statusCode, problem)
}

func safeInt32(value int) int32 {
	if value < math.MinInt32 {
		return math.MinInt32
	}
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(value)
}

func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

type apiErrorOptions struct {
	NotFoundMessage string
}

func writeClassifiedError(writer http.ResponseWriter, err error, options apiErrorOptions) {
	var validationErr *domain.ValidationError
	switch {
	case options.NotFoundMessage != "" && isNotFound(err):
		writeProblem(writer, http.StatusNotFound, problemSpec{
			Type:   "urn:woodgate:problem:not-found",
			Title:  "Not found",
			Code:   "not_found",
			Detail: options.NotFoundMessage,
		})
		return
	case errors.Is(err, domain.ErrPermissionDenied):
		writeProblem(writer, http.StatusForbidden, problemSpec{
			Type:   "urn:woodgate:problem:forbidden",
			Title:  "Forbidden",
			Code:   "forbidden",
			Detail: "Permission denied.",
		})
		return
	case errors.Is(err, pgutil.ErrInvalidSort), isBadRequestError(err):
		writeProblem(writer, http.StatusBadRequest, problemSpec{
			Type:   "urn:woodgate:problem:invalid-request",
			Title:  "Invalid request",
			Code:   "invalid_request",
			Detail: err.Error(),
		})
		return
	case errors.As(err, &validationErr):
		writeProblem(writer, http.StatusUnprocessableEntity, problemSpec{
			Type:        "urn:woodgate:problem:validation-error",
			Title:       "Validation failed",
			Code:        validationErr.Code,
			Detail:      validationErr.Detail,
			FieldErrors: validationErr.FieldErrors,
		})
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			writeProblem(writer, http.StatusConflict, problemSpec{
				Type:   "urn:woodgate:problem:conflict",
				Title:  "Conflict",
				Code:   "conflict",
				Detail: "Resource already exists.",
			})
			return
		case pgerrcode.ForeignKeyViolation:
			writeProblem(writer, http.StatusUnprocessableEntity, problemSpec{
				Type:   "urn:woodgate:problem:validation-error",
				Title:  "Validation failed",
				Code:   "validation_error",
				Detail: "Referenced resource does not exist.",
			})
			return
		}
	}

	writeProblem(writer, http.StatusInternalServerError, problemSpec{
		Type:   "urn:woodgate:problem:internal-error",
		Title:  "Internal server error",
		Code:   "internal_error",
		Detail: "An internal server error occurred.",
	})
}

func isBadRequestError(err error) bool {
	var requestErr badRequestError
	return errors.As(err, &requestErr)
}

func generateAPISecret() (string, string, string, error) {
	random := make([]byte, apiKeyBytes)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", fmt.Errorf("generate api key secret: %w", err)
	}

	secret := "woodgate_" + base64.RawURLEncoding.EncodeToString(random)
	prefix := secret
	if len(prefix) > apiKeyPrefixLen {
		prefix = prefix[:apiKeyPrefixLen]
	}

	hash := sha256.Sum256([]byte(secret))
	return secret, prefix, hex.EncodeToString(hash[:]), nil
}

type badRequestError string

func (err badRequestError) Error() string {
	return string(err)
}

func principalFromContext(ctx context.Context) (authz.Principal, error) {
	principal, ok := authz.PrincipalFromContext(ctx)
	if !ok {
		return authz.Principal{}, errors.New("missing principal")
	}
	return principal, nil
}

func principalSubject(principal authz.Principal) (domain.PermissionSubjectKind, uuid.UUID, error) {
	if principal.Bootstrap {
		return "", uuid.Nil, domain.ErrPermissionDenied
	}

	id, err := uuid.Parse(principal.ID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("parse principal id: %w", err)
	}

	switch principal.Kind {
	case authz.PrincipalKindUser:
		return domain.PermissionSubjectKindUser, id, nil
	case authz.PrincipalKindAPIKey:
		return domain.PermissionSubjectKindAPIKey, id, nil
	default:
		return "", uuid.Nil, errors.New("unsupported principal kind")
	}
}

func checkinScope(
	ctx context.Context,
	authorizer authz.Authorizer,
	required domain.PermissionAction,
) (authz.Scope[uuid.UUID], error) {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return authz.Scope[uuid.UUID]{}, err
	}

	return authorizer.CheckinScope(ctx, principal, string(required))
}

func assetScope(
	ctx context.Context,
	authorizer authz.Authorizer,
	required domain.PermissionAction,
) (authz.Scope[domain.AssetType], error) {
	principal, err := principalFromContext(ctx)
	if err != nil {
		return authz.Scope[domain.AssetType]{}, err
	}

	return authorizer.AssetScope(ctx, principal, string(required))
}

func requireString(field string, value string, validationErr *domain.ValidationError) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		validationErr.Add(field, "must not be empty", "required")
	}
	return trimmed
}
