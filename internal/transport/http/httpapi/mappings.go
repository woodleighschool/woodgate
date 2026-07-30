package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func mapUser(item domain.User) User {
	return User{
		Id:          idFromUUID(item.ID),
		Upn:         item.UPN,
		DisplayName: item.DisplayName,
		Department:  item.Department,
		Source:      Source(item.Source),
		Admin:       item.Admin,
		Access:      mapSliceValue(item.Access, mapPermissionGrant),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapGroup(item domain.Group) Group {
	return Group{
		Id:          idFromUUID(item.ID),
		Name:        item.Name,
		Description: item.Description,
		MemberCount: item.MemberCount,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapGroupMembership(item domain.GroupMembership) GroupMembership {
	return GroupMembership{
		Id:        idFromUUID(item.ID),
		GroupId:   idFromUUID(item.GroupID),
		UserId:    idFromUUID(item.UserID),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func mapAsset(item domain.Asset) Asset {
	return Asset{
		Id:        idFromUUID(item.ID),
		Name:      item.Name,
		Type:      AssetType(item.Type),
		Url:       assetContentURL(item.ID),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func mapLocation(item domain.Location) Location {
	return Location{
		Id:                idFromUUID(item.ID),
		Name:              item.Name,
		Description:       item.Description,
		Enabled:           item.Enabled,
		Notes:             item.Notes,
		Photo:             item.Photo,
		BackgroundAssetId: idPointer(item.BackgroundAssetID),
		LogoAssetId:       idPointer(item.LogoAssetID),
		GroupIds:          idSlice(item.GroupIDs),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func mapCheckin(item domain.Checkin) Checkin {
	return Checkin{
		Id:            idFromUUID(item.ID),
		UserId:        idFromUUID(item.UserID),
		LocationId:    idFromUUID(item.LocationID),
		Direction:     CheckinDirection(item.Direction),
		Notes:         item.Notes,
		AssetId:       idPointer(item.AssetID),
		CreatedByKind: PermissionSubjectKind(item.CreatedByKind),
		CreatedById:   idFromUUID(item.CreatedByID),
		CreatedAt:     item.CreatedAt,
	}
}

func mapAPIKey(item domain.APIKey) APIKey {
	return APIKey{
		Id:         idFromUUID(item.ID),
		Name:       item.Name,
		KeyPrefix:  item.KeyPrefix,
		LastUsedAt: item.LastUsedAt,
		ExpiresAt:  item.ExpiresAt,
		Admin:      item.Admin,
		Access:     mapSliceValue(item.Access, mapPermissionGrant),
		CreatedAt:  item.CreatedAt,
	}
}

func mapPermissionGrant(item domain.PermissionGrant) PermissionGrant {
	return PermissionGrant{
		Resource:   PermissionResource(item.Resource),
		Action:     PermissionAction(item.Action),
		LocationId: idPointer(item.LocationID),
		AssetType:  assetTypeOpenAPIPointer(item.AssetType),
	}
}

func idFromUUID(value uuid.UUID) Id {
	return value
}

func idPointer(value *uuid.UUID) *Id {
	if value == nil {
		return nil
	}
	id := idFromUUID(*value)
	return &id
}

func idSlice(values []uuid.UUID) []Id {
	if len(values) == 0 {
		return []Id{}
	}

	ids := make([]Id, 0, len(values))
	for _, value := range values {
		ids = append(ids, idFromUUID(value))
	}
	return ids
}

func uuidPointer(value *Id) *uuid.UUID {
	if value == nil {
		return nil
	}
	id := *value
	return &id
}

func uuidSlice(values []Id) []uuid.UUID {
	if len(values) == 0 {
		return []uuid.UUID{}
	}

	ids := make([]uuid.UUID, 0, len(values))
	ids = append(ids, values...)
	return ids
}

func boolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func timePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func checkinDirectionPointer(value *CheckinDirection) *domain.CheckinDirection {
	if value == nil {
		return nil
	}
	direction := domain.CheckinDirection(*value)
	return &direction
}

func assetTypePointer(value *AssetType) *domain.AssetType {
	if value == nil {
		return nil
	}
	assetType := domain.AssetType(*value)
	return &assetType
}

func assetTypeOpenAPIPointer(value *domain.AssetType) *AssetType {
	if value == nil {
		return nil
	}
	item := AssetType(*value)
	return &item
}

func assetContentURL(id uuid.UUID) string {
	return "/api/v1/assets/" + id.String() + "/content"
}
