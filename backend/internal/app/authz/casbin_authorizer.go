package authz

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

const (
	adminRoleName = "role:admin"
	globalDomain  = "global"
	anyDomain     = "*"
	locationScope = "location:"
	assetScope    = "asset_type:"
)

type PolicyStore interface {
	ListPrincipalRoles(context.Context) ([]domain.PrincipalRole, error)
	ListPrincipalPermissionGrants(context.Context) ([]domain.PrincipalPermissionGrant, error)
}

type CasbinAuthorizer struct {
	mu       sync.RWMutex
	enforcer *casbin.Enforcer
	store    PolicyStore
}

func NewCasbinAuthorizer(ctx context.Context, store PolicyStore) (*CasbinAuthorizer, error) {
	enforcer, err := casbin.NewEnforcer(newCasbinModel())
	if err != nil {
		return nil, fmt.Errorf("create casbin enforcer: %w", err)
	}

	authorizer := &CasbinAuthorizer{
		enforcer: enforcer,
		store:    store,
	}
	reloadErr := authorizer.Reload(ctx)
	if reloadErr != nil {
		return nil, reloadErr
	}

	return authorizer, nil
}

func (authorizer *CasbinAuthorizer) Reload(ctx context.Context) error {
	roles, err := authorizer.store.ListPrincipalRoles(ctx)
	if err != nil {
		return fmt.Errorf("list principal roles: %w", err)
	}
	grants, err := authorizer.store.ListPrincipalPermissionGrants(ctx)
	if err != nil {
		return fmt.Errorf("list principal permission grants: %w", err)
	}

	authorizer.mu.Lock()
	defer authorizer.mu.Unlock()

	authorizer.enforcer.ClearPolicy()
	_, addAdminPolicyErr := authorizer.enforcer.AddPolicy(adminRoleName, "*", "*", anyDomain)
	if addAdminPolicyErr != nil {
		return fmt.Errorf("seed admin policy: %w", addAdminPolicyErr)
	}

	for _, role := range roles {
		if !strings.EqualFold(strings.TrimSpace(role.Role), "admin") {
			continue
		}

		_, addGroupingErr := authorizer.enforcer.AddGroupingPolicy(
			subjectName(role.PrincipalKind, role.PrincipalID),
			adminRoleName,
			anyDomain,
		)
		if addGroupingErr != nil {
			return fmt.Errorf("add admin grouping policy: %w", addGroupingErr)
		}
	}

	for _, grant := range grants {
		_, addPolicyErr := authorizer.enforcer.AddPolicy(
			subjectName(grant.PrincipalKind, grant.PrincipalID),
			string(grant.Grant.Resource),
			string(grant.Grant.Action),
			scopeName(grant.Grant),
		)
		if addPolicyErr != nil {
			return fmt.Errorf("add permission policy: %w", addPolicyErr)
		}
	}

	return nil
}

func (authorizer *CasbinAuthorizer) Can(
	_ context.Context,
	principal Principal,
	resource string,
	action string,
) (bool, error) {
	if principal.Bootstrap {
		return true, nil
	}

	subject, resourceName, actionName, err := principalRequest(principal, resource, action)
	if err != nil {
		return false, err
	}

	authorizer.mu.RLock()
	defer authorizer.mu.RUnlock()

	allowed, err := authorizer.enforcer.Enforce(subject, resourceName, actionName, globalDomain)
	if err != nil {
		return false, fmt.Errorf("enforce policy: %w", err)
	}
	if allowed {
		return true, nil
	}

	switch resourceName {
	case string(domain.PermissionResourceCheckins):
		for _, domainName := range locationDomains(authorizer.enforcer, subject, actionName) {
			if domainName != globalDomain {
				return true, nil
			}
		}
	case string(domain.PermissionResourceAssets):
		for _, domainName := range assetTypeDomains(authorizer.enforcer, subject, actionName) {
			if domainName != globalDomain {
				return true, nil
			}
		}
	}

	return false, nil
}

func (authorizer *CasbinAuthorizer) GrantedLocations(
	_ context.Context,
	principal Principal,
	action string,
) ([]uuid.UUID, error) {
	if principal.Bootstrap {
		return nil, nil
	}

	subject, err := subjectFromPrincipal(principal)
	if err != nil {
		return nil, err
	}
	actionName, err := mapAction(action)
	if err != nil {
		return nil, err
	}

	authorizer.mu.RLock()
	defer authorizer.mu.RUnlock()

	domains := locationDomains(authorizer.enforcer, subject, actionName)
	locationIDs := make([]uuid.UUID, 0, len(domains))
	for _, domainName := range domains {
		locationID, parseErr := parseScopeLocationID(domainName)
		if parseErr != nil {
			return nil, parseErr
		}
		locationIDs = append(locationIDs, locationID)
	}

	return locationIDs, nil
}

func (authorizer *CasbinAuthorizer) GrantedAssetTypes(
	_ context.Context,
	principal Principal,
	action string,
) ([]domain.AssetType, error) {
	if principal.Bootstrap {
		return nil, nil
	}

	subject, err := subjectFromPrincipal(principal)
	if err != nil {
		return nil, err
	}
	actionName, err := mapAction(action)
	if err != nil {
		return nil, err
	}

	authorizer.mu.RLock()
	defer authorizer.mu.RUnlock()

	domains := assetTypeDomains(authorizer.enforcer, subject, actionName)
	assetTypes := make([]domain.AssetType, 0, len(domains))
	for _, domainName := range domains {
		assetType, parseErr := parseScopeAssetType(domainName)
		if parseErr != nil {
			return nil, parseErr
		}
		assetTypes = append(assetTypes, assetType)
	}

	return assetTypes, nil
}

func (authorizer *CasbinAuthorizer) IsAdmin(_ context.Context, principal Principal) (bool, error) {
	if principal.Bootstrap {
		return true, nil
	}

	subject, err := subjectFromPrincipal(principal)
	if err != nil {
		return false, err
	}

	authorizer.mu.RLock()
	defer authorizer.mu.RUnlock()

	return authorizer.enforcer.HasGroupingPolicy(subject, adminRoleName, anyDomain)
}

func newCasbinModel() model.Model {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act, dom")
	m.AddDef("p", "p", "sub, obj, act, dom")
	m.AddDef("g", "g", "_, _, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", `
    (r.sub == p.sub || g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) &&
    (p.obj == "*" || p.obj == r.obj) &&
    (p.act == "*" || p.act == r.act) &&
    (p.dom == "*" || p.dom == r.dom)
`)
	return m
}

func principalRequest(principal Principal, resource string, action string) (string, string, string, error) {
	subject, err := subjectFromPrincipal(principal)
	if err != nil {
		return "", "", "", err
	}
	resourceName, err := mapResource(resource)
	if err != nil {
		return "", "", "", err
	}
	actionName, err := mapAction(action)
	if err != nil {
		return "", "", "", err
	}
	return subject, resourceName, actionName, nil
}

func subjectFromPrincipal(principal Principal) (string, error) {
	subjectID, err := uuid.Parse(principal.ID)
	if err != nil {
		return "", fmt.Errorf("parse principal id: %w", err)
	}

	subjectKind, err := mapSubjectKind(principal.Kind)
	if err != nil {
		return "", err
	}

	return subjectName(subjectKind, subjectID), nil
}

func subjectName(kind domain.PermissionSubjectKind, id uuid.UUID) string {
	return fmt.Sprintf("%s:%s", kind, id)
}

func scopeName(grant domain.PermissionGrant) string {
	if grant.LocationID != nil {
		return locationScope + grant.LocationID.String()
	}
	if grant.AssetType != nil {
		return assetScope + string(*grant.AssetType)
	}
	if grant.LocationID == nil && grant.AssetType == nil {
		return globalDomain
	}
	return globalDomain
}

func parseScopeLocationID(value string) (uuid.UUID, error) {
	locationValue := strings.TrimPrefix(value, locationScope)
	return uuid.Parse(locationValue)
}

func parseScopeAssetType(value string) (domain.AssetType, error) {
	return domain.ParseAssetType(strings.TrimPrefix(value, assetScope))
}

func locationDomains(enforcer *casbin.Enforcer, subject string, action string) []string {
	policies, err := enforcer.GetFilteredPolicy(0, subject, string(domain.PermissionResourceCheckins), action)
	if err != nil {
		return nil
	}
	locationValues := make([]string, 0, len(policies))
	for _, policy := range policies {
		if len(policy) < 4 || policy[3] == globalDomain {
			continue
		}
		locationValues = append(locationValues, policy[3])
	}
	slices.Sort(locationValues)
	return slices.Compact(locationValues)
}

func assetTypeDomains(enforcer *casbin.Enforcer, subject string, action string) []string {
	policies, err := enforcer.GetFilteredPolicy(0, subject, string(domain.PermissionResourceAssets), action)
	if err != nil {
		return nil
	}
	values := make([]string, 0, len(policies))
	for _, policy := range policies {
		if len(policy) < 4 || policy[3] == globalDomain {
			continue
		}
		values = append(values, policy[3])
	}
	slices.Sort(values)
	return slices.Compact(values)
}
