package authhttp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-pkgz/auth/v2/token"
	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

type MeHandler struct {
	service    meLookupService
	authorizer authz.Authorizer
}

type meLookupService interface {
	GetUser(rctx context.Context, id uuid.UUID) (domain.User, error)
	GetAPIKey(rctx context.Context, id uuid.UUID) (domain.APIKey, error)
}

type meResponse struct {
	Principal    mePrincipal                   `json:"principal"`
	Admin        bool                          `json:"admin"`
	Access       []meGrant                     `json:"access"`
	Capabilities map[string]resourceCapability `json:"capabilities"`
}

type mePrincipal struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Name        string `json:"name,omitempty"`
}

type meGrant struct {
	Resource   string     `json:"resource"`
	Action     string     `json:"action"`
	LocationID *uuid.UUID `json:"location_id,omitempty"`
	AssetType  *string    `json:"asset_type,omitempty"`
}

type resourceCapability struct {
	Read   bool `json:"read"`
	Create bool `json:"create"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}

func NewMeHandler(service meLookupService, authorizer authz.Authorizer) *MeHandler {
	return &MeHandler{service: service, authorizer: authorizer}
}

func (handler *MeHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	principal, ok := authz.PrincipalFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusUnauthorized, "unauthorized")
		return
	}

	response, err := handler.buildResponse(request, principal)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "internal error")
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(response)
}

func (handler *MeHandler) buildResponse(request *http.Request, principal authz.Principal) (meResponse, error) {
	if principal.Bootstrap {
		user, _ := token.GetUserInfo(request)
		return meResponse{
			Principal: mePrincipal{
				Type:        string(authz.PrincipalKindUser),
				ID:          principal.ID,
				DisplayName: user.Name,
				Email:       user.Email,
			},
			Admin:        true,
			Access:       []meGrant{},
			Capabilities: fullCapabilities(),
		}, nil
	}

	admin, err := handler.authorizer.IsAdmin(request.Context(), principal)
	if err != nil {
		return meResponse{}, err
	}

	switch principal.Kind {
	case authz.PrincipalKindUser:
		id, parseErr := uuid.Parse(principal.ID)
		if parseErr != nil {
			return meResponse{}, parseErr
		}
		user, getErr := handler.service.GetUser(request.Context(), id)
		if getErr != nil {
			return meResponse{}, getErr
		}
		return meResponse{
			Principal: mePrincipal{
				Type:        string(principal.Kind),
				ID:          principal.ID,
				DisplayName: user.DisplayName,
				Email:       user.UPN,
			},
			Admin:        admin,
			Access:       mapGrants(user.Access),
			Capabilities: capabilities(admin, user.Access),
		}, nil
	case authz.PrincipalKindAPIKey:
		id, parseErr := uuid.Parse(principal.ID)
		if parseErr != nil {
			return meResponse{}, parseErr
		}
		apiKey, getErr := handler.service.GetAPIKey(request.Context(), id)
		if getErr != nil {
			return meResponse{}, getErr
		}
		return meResponse{
			Principal: mePrincipal{
				Type: string(principal.Kind),
				ID:   principal.ID,
				Name: apiKey.Name,
			},
			Admin:        admin,
			Access:       mapGrants(apiKey.Access),
			Capabilities: capabilities(admin, apiKey.Access),
		}, nil
	default:
		return meResponse{}, http.ErrNotSupported
	}
}

func mapGrants(grants []domain.PermissionGrant) []meGrant {
	items := make([]meGrant, 0, len(grants))
	for _, grant := range grants {
		items = append(items, meGrant{
			Resource:   string(grant.Resource),
			Action:     string(grant.Action),
			LocationID: grant.LocationID,
			AssetType:  stringPointer(grant.AssetType),
		})
	}
	return items
}

func stringPointer(value *domain.AssetType) *string {
	if value == nil {
		return nil
	}
	result := string(*value)
	return &result
}

func capabilities(admin bool, grants []domain.PermissionGrant) map[string]resourceCapability {
	if admin {
		return fullCapabilities()
	}

	items := defaultCapabilities()
	for _, grant := range grants {
		capability := items[string(grant.Resource)]
		switch grant.Action {
		case domain.PermissionActionRead:
			capability.Read = true
		case domain.PermissionActionCreate:
			capability.Create = true
		case domain.PermissionActionWrite:
			capability.Write = true
		case domain.PermissionActionDelete:
			capability.Delete = true
		}
		items[string(grant.Resource)] = capability
	}

	return items
}

func defaultCapabilities() map[string]resourceCapability {
	return map[string]resourceCapability{
		"users":     {},
		"groups":    {},
		"locations": {},
		"checkins":  {},
		"assets":    {},
		"api_keys":  {},
	}
}

func fullCapabilities() map[string]resourceCapability {
	items := defaultCapabilities()
	for key := range items {
		items[key] = resourceCapability{Read: true, Create: true, Write: true, Delete: true}
	}
	return items
}
