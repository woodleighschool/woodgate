package authorization

import (
	"context"
	"maps"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/woodleighschool/woodgate/internal/api/ctxkeys"
	apimiddleware "github.com/woodleighschool/woodgate/internal/api/middleware"
)

// Authorizer is the consumer contract shared by HTTP boundaries.
type Authorizer interface {
	Can(ctx context.Context, userID int64, resource Resource, required Access) (bool, error)
}

// Require decorates one protected operation with its explicit RBAC requirement.
func Require(
	api huma.API,
	service Authorizer,
	resource Resource,
	required Access,
	op huma.Operation,
) huma.Operation {
	op.Extensions = mergeExtensions(op.Extensions, map[string]any{
		"x-authz": map[string]any{"resource": string(resource), "access": string(required)},
	})
	op.Middlewares = append(op.Middlewares, func(ctx huma.Context, next func(huma.Context)) {
		user, err := ctxkeys.RequireUser(ctx.Context())
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "not authenticated")
			return
		}
		allowed, err := service.Can(ctx.Context(), user.ID, resource, required)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusInternalServerError, "authorization failed")
			return
		}
		if !allowed {
			_ = huma.WriteErr(api, ctx, http.StatusForbidden, "forbidden")
			return
		}
		next(ctx)
	})
	apimiddleware.DeclareErrorResponse(api, &op, http.StatusForbidden)
	return op
}

// RequireHTTP enforces one RBAC requirement on a non-Huma HTTP route.
func RequireHTTP(service Authorizer, resource Resource, required Access) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := ctxkeys.RequireUser(r.Context())
			if err != nil {
				http.Error(w, "not authenticated", http.StatusUnauthorized)
				return
			}
			allowed, err := service.Can(r.Context(), user.ID, resource, required)
			if err != nil {
				http.Error(w, "authorization failed", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func mergeExtensions(current map[string]any, extra map[string]any) map[string]any {
	if current == nil {
		current = map[string]any{}
	}
	maps.Copy(current, extra)
	return current
}
