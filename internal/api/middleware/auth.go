package middleware

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api/ctxkeys"
	"github.com/woodleighschool/woodgate/internal/auth"
	"github.com/woodleighschool/woodgate/internal/directory"
)

// Authenticator resolves a browser session into an application user.
type Authenticator interface {
	Authenticate(context.Context) (*directory.User, error)
}

func OptionalHumaAuth(api huma.API, authenticator Authenticator) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authenticator.Authenticate(ctx.Context())
		if err == nil {
			next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user)))
			return
		}
		if errors.Is(err, auth.ErrNotAuthenticated) {
			next(ctx)
			return
		}
		_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
	}
}

func RequireHumaAuth(api huma.API, authenticator Authenticator) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		user, err := authenticator.Authenticate(ctx.Context())
		if err != nil {
			if errors.Is(err, auth.ErrNotAuthenticated) {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
				return
			}
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "request failed")
			return
		}

		next(huma.WithContext(ctx, ctxkeys.WithUser(ctx.Context(), user)))
	}
}

// ProtectedOperation declares the browser session contract shared by protected operations.
func ProtectedOperation(api huma.API) func(*huma.Operation, func(*huma.Operation)) {
	return func(op *huma.Operation, next func(*huma.Operation)) {
		op.Security = []map[string][]string{{"cookieAuth": {}}}
		DeclareErrorResponse(api, op, http.StatusUnauthorized)
		next(op)
	}
}

func RequireHTTPAuth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			user, err := authenticator.Authenticate(req.Context())
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, auth.ErrNotAuthenticated) {
					status = http.StatusUnauthorized
				}
				http.Error(w, http.StatusText(status), status)
				return
			}
			next.ServeHTTP(w, req.WithContext(ctxkeys.WithUser(req.Context(), user)))
		})
	}
}

// DeclareErrorResponse adds Huma's standard problem response for status.
func DeclareErrorResponse(api huma.API, op *huma.Operation, status int) {
	if op.Responses == nil {
		op.Responses = map[string]*huma.Response{}
	}
	key := strconv.Itoa(status)
	if op.Responses[key] != nil {
		return
	}
	op.Responses[key] = &huma.Response{
		Description: http.StatusText(status),
		Content: map[string]*huma.MediaType{
			"application/problem+json": {
				Schema: api.OpenAPI().Components.Schemas.Schema(
					reflect.TypeFor[huma.ErrorModel](),
					true,
					"Error",
				),
			},
		},
	}
}
