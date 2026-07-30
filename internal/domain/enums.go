package domain

import "fmt"

func ParsePrincipalSource(value string) (PrincipalSource, error) {
	switch PrincipalSource(value) {
	case PrincipalSourceLocal:
		return PrincipalSourceLocal, nil
	case PrincipalSourceEntra:
		return PrincipalSourceEntra, nil
	default:
		return "", fmt.Errorf("unsupported source %q", value)
	}
}

func ParsePermissionSubjectKind(value string) (PermissionSubjectKind, error) {
	switch PermissionSubjectKind(value) {
	case PermissionSubjectKindUser:
		return PermissionSubjectKindUser, nil
	case PermissionSubjectKindAPIKey:
		return PermissionSubjectKindAPIKey, nil
	default:
		return "", fmt.Errorf("unsupported permission subject kind %q", value)
	}
}

func ParsePermissionResource(value string) (PermissionResource, error) {
	switch PermissionResource(value) {
	case PermissionResourceUsers:
		return PermissionResourceUsers, nil
	case PermissionResourceGroups:
		return PermissionResourceGroups, nil
	case PermissionResourceLocations:
		return PermissionResourceLocations, nil
	case PermissionResourceCheckins:
		return PermissionResourceCheckins, nil
	case PermissionResourceAssets:
		return PermissionResourceAssets, nil
	case PermissionResourceAPIKeys:
		return PermissionResourceAPIKeys, nil
	default:
		return "", fmt.Errorf("unsupported permission resource %q", value)
	}
}

func ParsePermissionAction(value string) (PermissionAction, error) {
	switch PermissionAction(value) {
	case PermissionActionRead:
		return PermissionActionRead, nil
	case PermissionActionCreate:
		return PermissionActionCreate, nil
	case PermissionActionWrite:
		return PermissionActionWrite, nil
	case PermissionActionDelete:
		return PermissionActionDelete, nil
	default:
		return "", fmt.Errorf("unsupported permission action %q", value)
	}
}

func ParseCheckinDirection(value string) (CheckinDirection, error) {
	switch CheckinDirection(value) {
	case CheckinDirectionIn:
		return CheckinDirectionIn, nil
	case CheckinDirectionOut:
		return CheckinDirectionOut, nil
	default:
		return "", fmt.Errorf("unsupported checkin direction %q", value)
	}
}

func ParseAssetType(value string) (AssetType, error) {
	switch AssetType(value) {
	case AssetTypeAsset:
		return AssetTypeAsset, nil
	case AssetTypePhoto:
		return AssetTypePhoto, nil
	default:
		return "", fmt.Errorf("unsupported asset type %q", value)
	}
}

func (action PermissionAction) Allows(required PermissionAction) bool {
	return action == required
}
