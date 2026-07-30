package authz

import (
	"errors"
	"fmt"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func mapAction(action string) (string, error) {
	switch action {
	case "read":
		return string(domain.PermissionActionRead), nil
	case "create":
		return string(domain.PermissionActionCreate), nil
	case "write":
		return string(domain.PermissionActionWrite), nil
	case "delete":
		return string(domain.PermissionActionDelete), nil
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}
}

func mapSubjectKind(kind PrincipalKind) (domain.PermissionSubjectKind, error) {
	switch kind {
	case PrincipalKindUser:
		return domain.PermissionSubjectKindUser, nil
	case PrincipalKindAPIKey:
		return domain.PermissionSubjectKindAPIKey, nil
	default:
		return "", errors.New("unsupported principal kind")
	}
}

func mapResource(resource string) (string, error) {
	switch resource {
	case "users":
		return string(domain.PermissionResourceUsers), nil
	case "groups", "group-memberships":
		return string(domain.PermissionResourceGroups), nil
	case "locations":
		return string(domain.PermissionResourceLocations), nil
	case "checkins":
		return string(domain.PermissionResourceCheckins), nil
	case "assets":
		return string(domain.PermissionResourceAssets), nil
	case "api-keys":
		return string(domain.PermissionResourceAPIKeys), nil
	default:
		return "", fmt.Errorf("unsupported resource %q", resource)
	}
}
