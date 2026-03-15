package authz

import (
	"context"

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

type Authorizer interface {
	Can(ctx context.Context, principal Principal, resource string, action string) (bool, error)
	GrantedLocations(ctx context.Context, principal Principal, action string) ([]uuid.UUID, error)
	GrantedAssetTypes(ctx context.Context, principal Principal, action string) ([]domain.AssetType, error)
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
