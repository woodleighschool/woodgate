package pgutil

import "github.com/woodleighschool/woodgate/internal/domain"

func ToSource(value string) (domain.PrincipalSource, error) {
	return domain.ParsePrincipalSource(value)
}

func ToPermissionSubjectKind(value string) (domain.PermissionSubjectKind, error) {
	return domain.ParsePermissionSubjectKind(value)
}

func ToPermissionResource(value string) (domain.PermissionResource, error) {
	return domain.ParsePermissionResource(value)
}

func ToPermissionAction(value string) (domain.PermissionAction, error) {
	return domain.ParsePermissionAction(value)
}

func ToCheckinDirection(value string) (domain.CheckinDirection, error) {
	return domain.ParseCheckinDirection(value)
}

func ToAssetType(value string) (domain.AssetType, error) {
	return domain.ParseAssetType(value)
}
