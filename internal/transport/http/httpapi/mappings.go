package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/domain"
)

func mapUser(item domain.User) User {
	return User{
		ID:          idFromUUID(item.ID),
		UPN:         item.UPN,
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
		ID:          idFromUUID(item.ID),
		Name:        item.Name,
		Description: item.Description,
		MemberCount: item.MemberCount,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapGroupMembership(item domain.GroupMembership) GroupMembership {
	return GroupMembership{
		ID:        idFromUUID(item.ID),
		GroupID:   idFromUUID(item.GroupID),
		UserID:    idFromUUID(item.UserID),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func mapAsset(item domain.Asset) Asset {
	return Asset{
		ID:        idFromUUID(item.ID),
		Name:      item.Name,
		Type:      AssetType(item.Type),
		URL:       assetContentURL(item.ID),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func mapLocation(item domain.Location) Location {
	return Location{
		ID:                idFromUUID(item.ID),
		Name:              item.Name,
		Description:       item.Description,
		Enabled:           item.Enabled,
		Notes:             item.Notes,
		Photo:             item.Photo,
		BackgroundAssetID: idPointer(item.BackgroundAssetID),
		LogoAssetID:       idPointer(item.LogoAssetID),
		GroupIDs:          idSlice(item.GroupIDs),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func mapCheckin(item domain.Checkin) Checkin {
	return Checkin{
		ID:              idFromUUID(item.ID),
		UserID:          idFromUUID(item.UserID),
		UserDisplayName: item.UserDisplayName,
		Department:      item.Department,
		LocationID:      idFromUUID(item.LocationID),
		LocationName:    item.LocationName,
		Direction:       CheckinDirection(item.Direction),
		Notes:           item.Notes,
		AssetID:         idPointer(item.AssetID),
		PhotoURL:        assetContentURLPointer(item.AssetID),
		CreatedByKind:   PermissionSubjectKind(item.CreatedByKind),
		CreatedByID:     idFromUUID(item.CreatedByID),
		CreatedAt:       item.CreatedAt,
	}
}

func mapDepartmentOption(item string) DepartmentOption {
	return DepartmentOption{ID: item, Name: item}
}

func mapAPIKey(item domain.APIKey) APIKey {
	return APIKey{
		ID:         idFromUUID(item.ID),
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
		LocationID: idPointer(item.LocationID),
		AssetType:  assetTypeOpenAPIPointer(item.AssetType),
	}
}

func idFromUUID(value uuid.UUID) uuid.UUID {
	return value
}

func idPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	id := idFromUUID(*value)
	return &id
}

func idSlice(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return []uuid.UUID{}
	}

	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		ids = append(ids, idFromUUID(value))
	}
	return ids
}

func uuidPointer(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	id := *value
	return &id
}

func uuidSlice(values []uuid.UUID) []uuid.UUID {
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

func assetContentURLPointer(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	url := assetContentURL(*id)
	return &url
}
