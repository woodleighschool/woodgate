package authhttp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-pkgz/auth/token"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
	"github.com/woodleighschool/woodgate/internal/transport/http/apierrors"
)

type APIKeyAuthenticator interface {
	AuthenticateAPIKey(context.Context, string) (uuid.UUID, error)
}

type SessionUserResolver interface {
	GetUserByUPN(context.Context, string) (domain.User, error)
}

func NewAPIMiddleware(
	sessionAuth func(http.Handler) http.Handler,
	apiKeys APIKeyAuthenticator,
	users SessionUserResolver,
	authorizer authz.Authorizer,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return NewPrincipalMiddleware(sessionAuth, apiKeys, users)(NewAuthorizationMiddleware(authorizer)(next))
	}
}

func NewPrincipalMiddleware(
	sessionAuth func(http.Handler) http.Handler,
	apiKeys APIKeyAuthenticator,
	users SessionUserResolver,
) func(http.Handler) http.Handler {
	if sessionAuth == nil {
		panic("sessionAuth middleware is required")
	}

	return func(next http.Handler) http.Handler {
		sessionProtected := makeSessionProtected(next, sessionAuth, users)

		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			apiKey := apiKeyFromHeader(request)
			if apiKey == "" {
				sessionProtected.ServeHTTP(writer, request)
				return
			}

			handleAPIKey(writer, request, next, apiKeys, apiKey)
		})
	}
}

func NewAuthorizationMiddleware(authorizer authz.Authorizer) func(http.Handler) http.Handler {
	if authorizer == nil {
		panic("authorizer is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			principal, ok := authz.PrincipalFromContext(request.Context())
			if !ok {
				writeError(writer, http.StatusUnauthorized, "unauthorized")
				return
			}

			if !authorizeRequest(writer, request, authorizer, principal) {
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func makeSessionProtected(
	next http.Handler,
	sessionAuth func(http.Handler) http.Handler,
	users SessionUserResolver,
) http.Handler {
	return sessionAuth(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, err := token.GetUserInfo(request)
		if err != nil {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}

		if isBootstrapUser(user) {
			serveWithPrincipal(
				writer,
				request,
				next,
				authz.Principal{
					Kind:      authz.PrincipalKindUser,
					ID:        user.ID,
					Bootstrap: true,
				},
			)
			return
		}

		if users == nil {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}

		mappedUser, resolveErr := users.GetUserByUPN(request.Context(), user.Email)
		if resolveErr != nil {
			if errors.Is(resolveErr, pgx.ErrNoRows) {
				writeError(writer, http.StatusForbidden, "forbidden")
				return
			}
			writeError(writer, http.StatusInternalServerError, "internal error")
			return
		}

		serveWithPrincipal(
			writer,
			request,
			next,
			authz.Principal{
				Kind: authz.PrincipalKindUser,
				ID:   mappedUser.ID.String(),
			},
		)
	}))
}

func handleAPIKey(
	writer http.ResponseWriter,
	request *http.Request,
	next http.Handler,
	apiKeys APIKeyAuthenticator,
	apiKey string,
) {
	if apiKeys == nil {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID, err := apiKeys.AuthenticateAPIKey(request.Context(), apiKey)
	if err != nil {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return
	}

	serveWithPrincipal(
		writer,
		request,
		next,
		authz.Principal{
			Kind: authz.PrincipalKindAPIKey,
			ID:   keyID.String(),
		},
	)
}

func isBootstrapUser(user token.User) bool {
	return strings.HasPrefix(user.ID, "local_") || strings.TrimSpace(user.Email) == ""
}

func serveWithPrincipal(
	writer http.ResponseWriter,
	request *http.Request,
	next http.Handler,
	principal authz.Principal,
) {
	next.ServeHTTP(
		writer,
		request.WithContext(authz.WithPrincipal(request.Context(), principal)),
	)
}

func authorizeRequest(
	writer http.ResponseWriter,
	request *http.Request,
	authorizer authz.Authorizer,
	principal authz.Principal,
) bool {
	resource := resolveResource(request.URL.Path)
	action := resolveAction(request.Method)

	allowed, err := authorizer.Can(request.Context(), principal, resource, action)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal error")
		return false
	}
	if !allowed {
		writeError(writer, http.StatusForbidden, "forbidden")
		return false
	}

	return true
}

func apiKeyFromHeader(request *http.Request) string {
	headerName := http.CanonicalHeaderKey("X-" + "API-Key")
	return strings.TrimSpace(request.Header.Get(headerName))
}

func resolveResource(path string) string {
	trimmed := strings.TrimPrefix(strings.TrimSpace(path), "/api/v1")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "unknown"
	}
	segments := strings.Split(trimmed, "/")
	return segments[0]
}

func resolveAction(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodDelete:
		return "delete"
	default:
		return "write"
	}
}

func writeError(writer http.ResponseWriter, statusCode int, message string) {
	apierrors.Write(writer, statusCode, message)
}
