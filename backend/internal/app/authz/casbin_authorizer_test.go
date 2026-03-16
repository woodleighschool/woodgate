package authz_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

type stubPolicyStore struct {
	roles  []domain.PrincipalRole
	grants []domain.PrincipalPermissionGrant
}

func (store stubPolicyStore) ListPrincipalRoles(context.Context) ([]domain.PrincipalRole, error) {
	return store.roles, nil
}

func (store stubPolicyStore) ListPrincipalPermissionGrants(context.Context) ([]domain.PrincipalPermissionGrant, error) {
	return store.grants, nil
}

func TestCasbinAuthorizer_AssetScope_AllForAdmin(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), stubPolicyStore{
		roles: []domain.PrincipalRole{{
			PrincipalKind: domain.PermissionSubjectKindUser,
			PrincipalID:   userID,
			Role:          "admin",
		}},
	})
	if err != nil {
		t.Fatalf("NewCasbinAuthorizer returned error: %v", err)
	}

	scope, err := authorizer.AssetScope(context.Background(), authz.Principal{
		Kind: authz.PrincipalKindUser,
		ID:   userID.String(),
	}, "create")
	if err != nil {
		t.Fatalf("AssetScope returned error: %v", err)
	}
	if !scope.All {
		t.Fatalf("expected admin asset scope to be unrestricted")
	}
}

func TestCasbinAuthorizer_CheckinScope_ReturnsGrantedLocations(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), stubPolicyStore{
		grants: []domain.PrincipalPermissionGrant{{
			PrincipalKind: domain.PermissionSubjectKindUser,
			PrincipalID:   userID,
			Grant: domain.PermissionGrant{
				Resource:   domain.PermissionResourceCheckins,
				Action:     domain.PermissionActionCreate,
				LocationID: &locationID,
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewCasbinAuthorizer returned error: %v", err)
	}

	scope, err := authorizer.CheckinScope(context.Background(), authz.Principal{
		Kind: authz.PrincipalKindUser,
		ID:   userID.String(),
	}, "create")
	if err != nil {
		t.Fatalf("CheckinScope returned error: %v", err)
	}
	if scope.All {
		t.Fatalf("expected scoped checkin access, got unrestricted scope")
	}
	if len(scope.Values) != 1 || scope.Values[0] != locationID {
		t.Fatalf("scope.Values = %v, want [%s]", scope.Values, locationID)
	}
}

func TestCasbinAuthorizer_Can_AllowsScopedAssetAccess(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	assetType := domain.AssetTypeAsset
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), stubPolicyStore{
		grants: []domain.PrincipalPermissionGrant{{
			PrincipalKind: domain.PermissionSubjectKindUser,
			PrincipalID:   userID,
			Grant: domain.PermissionGrant{
				Resource:  domain.PermissionResourceAssets,
				Action:    domain.PermissionActionRead,
				AssetType: &assetType,
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewCasbinAuthorizer returned error: %v", err)
	}

	allowed, err := authorizer.Can(context.Background(), authz.Principal{
		Kind: authz.PrincipalKindUser,
		ID:   userID.String(),
	}, "assets", "read")
	if err != nil {
		t.Fatalf("Can returned error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected scoped asset grant to allow route access")
	}
}

func TestCasbinAuthorizer_AssetScope_AllForGlobalGrant(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), stubPolicyStore{
		grants: []domain.PrincipalPermissionGrant{{
			PrincipalKind: domain.PermissionSubjectKindUser,
			PrincipalID:   userID,
			Grant: domain.PermissionGrant{
				Resource: domain.PermissionResourceAssets,
				Action:   domain.PermissionActionRead,
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewCasbinAuthorizer returned error: %v", err)
	}

	scope, err := authorizer.AssetScope(context.Background(), authz.Principal{
		Kind: authz.PrincipalKindUser,
		ID:   userID.String(),
	}, "read")
	if err != nil {
		t.Fatalf("AssetScope returned error: %v", err)
	}
	if !scope.All {
		t.Fatalf("expected global asset grant to produce unrestricted scope")
	}
}

func TestCasbinAuthorizer_CheckinScope_AllForGlobalGrant(t *testing.T) {
	t.Parallel()

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	authorizer, err := authz.NewCasbinAuthorizer(context.Background(), stubPolicyStore{
		grants: []domain.PrincipalPermissionGrant{{
			PrincipalKind: domain.PermissionSubjectKindUser,
			PrincipalID:   userID,
			Grant: domain.PermissionGrant{
				Resource: domain.PermissionResourceCheckins,
				Action:   domain.PermissionActionRead,
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewCasbinAuthorizer returned error: %v", err)
	}

	scope, err := authorizer.CheckinScope(context.Background(), authz.Principal{
		Kind: authz.PrincipalKindUser,
		ID:   userID.String(),
	}, "read")
	if err != nil {
		t.Fatalf("CheckinScope returned error: %v", err)
	}
	if !scope.All {
		t.Fatalf("expected global checkin grant to produce unrestricted scope")
	}
}
