package authz

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

type PrincipalKind string

const (
	PrincipalKindUser   PrincipalKind = "user"
	PrincipalKindAPIKey PrincipalKind = "api_key"
)

type Principal struct {
	Kind      PrincipalKind
	ID        string
	Bootstrap bool
}

type Scope[T comparable] struct {
	All    bool
	Values []T
}

func (scope Scope[T]) AllowsAny() bool {
	return scope.All || len(scope.Values) > 0
}

func (scope Scope[T]) Contains(value T) bool {
	return scope.All || slices.Contains(scope.Values, value)
}

type Authorizer interface {
	Can(ctx context.Context, principal Principal, resource string, action string) (bool, error)
	CheckinScope(ctx context.Context, principal Principal, action string) (Scope[uuid.UUID], error)
	AssetScope(ctx context.Context, principal Principal, action string) (Scope[domain.AssetType], error)
	IsAdmin(ctx context.Context, principal Principal) (bool, error)
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
