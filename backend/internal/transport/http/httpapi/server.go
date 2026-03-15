package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/woodleighschool/woodgate/internal/app/authz"
	"github.com/woodleighschool/woodgate/internal/domain"
)

func (handler *Server) ListUsers(writer http.ResponseWriter, request *http.Request, params ListUsersParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	locationID := uuidPointer(params.LocationId)
	principal, err := principalFromContext(request.Context())
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if principal.Kind == authz.PrincipalKindAPIKey {
		if locationID == nil {
			writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
			return
		}

		grantedLocationIDs, locationsErr := handler.authorizer.GrantedLocations(
			request.Context(),
			principal,
			string(domain.PermissionActionCreate),
		)
		if locationsErr != nil {
			writeClassifiedError(writer, locationsErr, apiErrorOptions{})
			return
		}
		if !containsLocationID(grantedLocationIDs, *locationID) {
			writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
			return
		}
	}

	items, total, err := handler.admin.ListUsers(request.Context(), domain.UserListOptions{
		ListOptions: listOptions,
		LocationID:  locationID,
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]User, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapUser(item))
	}

	writeJSON(writer, http.StatusOK, UserListResponse{Rows: rows, Total: total})
}

func (handler *Server) GetUser(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetUser(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "user not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapUser(item))
}

func (handler *Server) PatchUser(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchUserJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	permissions, err := validatePermissionGrants(body.Access, "User access is invalid.")
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateUserAccess(request.Context(), id, body.Admin, permissions)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "user not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapUser(item))
}

func (handler *Server) ListGroups(writer http.ResponseWriter, request *http.Request, params ListGroupsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListGroups(request.Context(), domain.GroupListOptions{ListOptions: listOptions})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]Group, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapGroup(item))
	}

	writeJSON(writer, http.StatusOK, GroupListResponse{Rows: rows, Total: total})
}

func (handler *Server) GetGroup(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetGroup(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "group not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapGroup(item))
}

func (handler *Server) ListGroupMemberships(
	writer http.ResponseWriter,
	request *http.Request,
	params ListGroupMembershipsParams,
) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListGroupMemberships(request.Context(), domain.GroupMembershipListOptions{
		ListOptions: listOptions,
		GroupID:     uuidPointer(params.GroupId),
		UserID:      uuidPointer(params.UserId),
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]GroupMembership, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapGroupMembership(item))
	}

	writeJSON(writer, http.StatusOK, GroupMembershipListResponse{Rows: rows, Total: total})
}

func (handler *Server) GetGroupMembership(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetGroupMembership(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "group membership not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapGroupMembership(item))
}

func (handler *Server) ListAssets(writer http.ResponseWriter, request *http.Request, params ListAssetsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	allowedTypes, bypassScope, err := assetScope(request.Context(), handler.authorizer, domain.PermissionActionRead)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	types := assetTypesFilter(params.Type, allowedTypes, bypassScope)
	if !bypassScope && len(types) == 0 {
		writeJSON(writer, http.StatusOK, AssetListResponse{Rows: []Asset{}, Total: 0})
		return
	}

	items, total, err := handler.admin.ListAssets(request.Context(), domain.AssetListOptions{
		ListOptions: listOptions,
		Types:       types,
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]Asset, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapAsset(item))
	}

	writeJSON(writer, http.StatusOK, AssetListResponse{Rows: rows, Total: total})
}

func (handler *Server) CreateAsset(writer http.ResponseWriter, request *http.Request) {
	allowedTypes, bypassScope, err := assetScope(request.Context(), handler.authorizer, domain.PermissionActionCreate)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if !bypassScope && !slices.Contains(allowedTypes, domain.AssetTypeAsset) {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	name, content, err := parseAssetUploadRequest(writer, request, true)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateAsset(request.Context(), name, content)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapAsset(item))
}

func (handler *Server) GetAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetAsset(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(request.Context(), handler.authorizer, domain.PermissionActionRead, item.Type)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}
	writeJSON(writer, http.StatusOK, mapAsset(item))
}

func (handler *Server) GetAssetContent(writer http.ResponseWriter, request *http.Request, id Id) {
	item, path, err := handler.admin.GetAssetFile(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(request.Context(), handler.authorizer, domain.PermissionActionRead, item.Type)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	_, statErr := os.Stat(path)
	if statErr != nil {
		writeClassifiedError(writer, statErr, apiErrorOptions{})
		return
	}

	writer.Header().Set("Cache-Control", "private, max-age=3600")
	writer.Header().Set("Content-Type", item.ContentType)
	http.ServeFile(writer, request, path)
}

func (handler *Server) PatchAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	current, err := handler.admin.GetAsset(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionWrite,
		current.Type,
	)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	name, content, err := parseAssetUploadRequest(writer, request, false)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateAsset(request.Context(), id, name, content)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapAsset(item))
}

func (handler *Server) DeleteAsset(writer http.ResponseWriter, request *http.Request, id Id) {
	current, err := handler.admin.GetAsset(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	allowed, allowErr := allowAssetType(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionDelete,
		current.Type,
	)
	if allowErr != nil {
		writeClassifiedError(writer, allowErr, apiErrorOptions{})
		return
	}
	if !allowed {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	deleteErr := handler.admin.DeleteAsset(request.Context(), id)
	if deleteErr != nil {
		writeClassifiedError(writer, deleteErr, apiErrorOptions{NotFoundMessage: "asset not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Server) ListLocations(writer http.ResponseWriter, request *http.Request, params ListLocationsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListLocations(request.Context(), domain.LocationListOptions{
		ListOptions: listOptions,
		Enabled:     boolPointer(params.Enabled),
	})
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]Location, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapLocation(item))
	}

	writeJSON(writer, http.StatusOK, LocationListResponse{Rows: rows, Total: total})
}

func (handler *Server) CreateLocation(writer http.ResponseWriter, request *http.Request) {
	var body CreateLocationJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateLocation(
		request.Context(),
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetId),
		uuidPointer(body.LogoAssetId),
		uuidSlice(body.GroupIds),
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapLocation(item))
}

func (handler *Server) GetLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetLocation(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapLocation(item))
}

func (handler *Server) PatchLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchLocationJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Location is invalid."}
	name := requireString("name", body.Name, validationErr)
	description := strings.TrimSpace(body.Description)
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateLocation(
		request.Context(),
		id,
		name,
		description,
		body.Enabled,
		body.Notes,
		body.Photo,
		uuidPointer(body.BackgroundAssetId),
		uuidPointer(body.LogoAssetId),
		uuidSlice(body.GroupIds),
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapLocation(item))
}

func (handler *Server) DeleteLocation(writer http.ResponseWriter, request *http.Request, id Id) {
	if err := handler.admin.DeleteLocation(request.Context(), id); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (handler *Server) ListCheckins(writer http.ResponseWriter, request *http.Request, params ListCheckinsParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	allowedLocationIDs, bootstrap, err := checkinScope(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if bootstrap {
		allowedLocationIDs = nil
	}

	items, total, err := handler.admin.ListCheckins(request.Context(), domain.CheckinListOptions{
		ListOptions: listOptions,
		LocationID:  uuidPointer(params.LocationId),
		UserID:      uuidPointer(params.UserId),
		Direction:   checkinDirectionPointer(params.Direction),
		CreatedFrom: timePointer(params.CreatedFrom),
		CreatedTo:   timePointer(params.CreatedTo),
	}, allowedLocationIDs)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]Checkin, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapCheckin(item))
	}

	writeJSON(writer, http.StatusOK, CheckinListResponse{Rows: rows, Total: total})
}

func (handler *Server) CreateCheckin(writer http.ResponseWriter, request *http.Request) {
	body, err := parseCheckinCreateRequest(writer, request)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	principal, err := principalFromContext(request.Context())
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if principal.Bootstrap {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	subjectKind, subjectID, err := principalSubject(principal)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	locationIDs, err := handler.authorizer.GrantedLocations(
		request.Context(),
		principal,
		string(domain.PermissionActionCreate),
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if !containsLocationID(locationIDs, body.LocationID) {
		writeClassifiedError(writer, domain.ErrPermissionDenied, apiErrorOptions{})
		return
	}

	location, err := handler.admin.GetLocation(request.Context(), body.LocationID)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "location not found"})
		return
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Checkin is invalid."}
	if !location.Enabled {
		validationErr.Add("location_id", "must reference an enabled location", "invalid")
	}
	if location.Photo && len(body.PhotoContent) == 0 {
		validationErr.Add("photo", "is required when the location requires a photo", "required")
	}
	if !location.Notes && body.Notes != "" {
		validationErr.Add("notes", "must be empty when notes are disabled for the location", "invalid")
	}
	if validationErr.HasFieldErrors() {
		writeClassifiedError(writer, validationErr, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateCheckin(
		request.Context(),
		body.UserID,
		body.LocationID,
		body.Direction,
		body.Notes,
		body.PhotoContent,
		subjectKind,
		subjectID,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, mapCheckin(item))
}

func (handler *Server) GetCheckin(writer http.ResponseWriter, request *http.Request, id Id) {
	allowedLocationIDs, bootstrap, err := checkinScope(
		request.Context(),
		handler.authorizer,
		domain.PermissionActionRead,
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}
	if bootstrap {
		allowedLocationIDs = nil
	}

	item, err := handler.admin.GetCheckin(request.Context(), id, allowedLocationIDs)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "checkin not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapCheckin(item))
}

func (handler *Server) ListAPIKeys(writer http.ResponseWriter, request *http.Request, params ListAPIKeysParams) {
	listOptions, err := parseListOptions(params.Limit, params.Offset, params.Search, params.Sort, params.Order)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	items, total, err := handler.admin.ListAPIKeys(
		request.Context(),
		domain.APIKeyListOptions{ListOptions: listOptions},
	)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	rows := make([]APIKey, 0, len(items))
	for _, item := range items {
		rows = append(rows, mapAPIKey(item))
	}

	writeJSON(writer, http.StatusOK, APIKeyListResponse{Rows: rows, Total: total})
}

func (handler *Server) CreateAPIKey(writer http.ResponseWriter, request *http.Request) {
	var body CreateAPIKeyJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeClassifiedError(writer, &domain.ValidationError{
			Code:   "validation_error",
			Detail: "API key is invalid.",
			FieldErrors: []domain.FieldError{{
				Field:   "name",
				Message: "must not be empty",
				Code:    "required",
			}},
		}, apiErrorOptions{})
		return
	}

	secret, prefix, secretHash, err := generateAPISecret()
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.CreateAPIKey(request.Context(), name, prefix, secretHash, body.ExpiresAt)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	writeJSON(writer, http.StatusCreated, CreateAPIKeyData{
		Id:         idFromUUID(item.ID),
		Name:       item.Name,
		KeyPrefix:  item.KeyPrefix,
		LastUsedAt: item.LastUsedAt,
		ExpiresAt:  item.ExpiresAt,
		Admin:      false,
		Access:     []PermissionGrant{},
		CreatedAt:  item.CreatedAt,
		Secret:     secret,
	})
}

func (handler *Server) GetAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	item, err := handler.admin.GetAPIKey(request.Context(), id)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}
	writeJSON(writer, http.StatusOK, mapAPIKey(item))
}

func (handler *Server) PatchAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	var body PatchAPIKeyJSONRequestBody
	if err := decodeJSONBody(request, &body); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	permissions, err := validatePermissionGrants(body.Access, "API key access is invalid.")
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{})
		return
	}

	item, err := handler.admin.UpdateAPIKeyAccess(request.Context(), id, body.Admin, permissions)
	if err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}

	writeJSON(writer, http.StatusOK, mapAPIKey(item))
}

func (handler *Server) DeleteAPIKey(writer http.ResponseWriter, request *http.Request, id Id) {
	if err := handler.admin.DeleteAPIKey(request.Context(), id); err != nil {
		writeClassifiedError(writer, err, apiErrorOptions{NotFoundMessage: "api key not found"})
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func mapUser(item domain.User) User {
	access := make([]PermissionGrant, 0, len(item.Access))
	for _, permission := range item.Access {
		access = append(access, mapPermissionGrant(permission))
	}

	return User{
		Id:          idFromUUID(item.ID),
		Upn:         item.UPN,
		DisplayName: item.DisplayName,
		Department:  item.Department,
		Source:      Source(item.Source),
		Admin:       item.Admin,
		Access:      access,
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
	access := make([]PermissionGrant, 0, len(item.Access))
	for _, permission := range item.Access {
		access = append(access, mapPermissionGrant(permission))
	}

	return APIKey{
		Id:         idFromUUID(item.ID),
		Name:       item.Name,
		KeyPrefix:  item.KeyPrefix,
		LastUsedAt: item.LastUsedAt,
		ExpiresAt:  item.ExpiresAt,
		Admin:      item.Admin,
		Access:     access,
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

func assetTypesFilter(value *AssetType, allowed []domain.AssetType, bypass bool) []domain.AssetType {
	if bypass {
		if value == nil {
			return nil
		}
		return []domain.AssetType{domain.AssetType(*value)}
	}

	if value != nil {
		requested := domain.AssetType(*value)
		if slices.Contains(allowed, requested) {
			return []domain.AssetType{requested}
		}
		return []domain.AssetType{}
	}

	return allowed
}

func allowAssetType(
	ctx context.Context,
	authorizer authz.Authorizer,
	required domain.PermissionAction,
	assetType domain.AssetType,
) (bool, error) {
	allowedTypes, bypassScope, err := assetScope(ctx, authorizer, required)
	if err != nil {
		return false, err
	}
	if bypassScope {
		return true, nil
	}
	return slices.Contains(allowedTypes, assetType), nil
}

func containsLocationID(values []uuid.UUID, target uuid.UUID) bool {
	return slices.Contains(values, target)
}

func assetContentURL(id uuid.UUID) string {
	return "/api/v1/assets/" + id.String() + "/content"
}

func parseAssetUploadRequest(
	writer http.ResponseWriter,
	request *http.Request,
	requireFile bool,
) (*string, []byte, error) {
	parseErr := parseMultipartForm(writer, request)
	if parseErr != nil {
		return nil, nil, parseErr
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Asset is invalid."}
	name := optionalStringPointer(multipartValue(request, "name"))
	content, fileErr := readMultipartFile(request, "file", requireFile, validationErr)
	if fileErr != nil {
		return nil, nil, fileErr
	}

	if validationErr.HasFieldErrors() {
		return nil, nil, validationErr
	}

	return name, content, nil
}

type checkinCreateRequest struct {
	UserID       uuid.UUID
	LocationID   uuid.UUID
	Direction    domain.CheckinDirection
	Notes        string
	PhotoContent []byte
}

func parseCheckinCreateRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (checkinCreateRequest, error) {
	parseErr := parseMultipartForm(writer, request)
	if parseErr != nil {
		return checkinCreateRequest{}, parseErr
	}

	validationErr := &domain.ValidationError{Code: "validation_error", Detail: "Checkin is invalid."}

	userID, userIDErr := uuid.Parse(strings.TrimSpace(multipartValue(request, "user_id")))
	if userIDErr != nil {
		validationErr.Add("user_id", "is invalid", "invalid")
	}

	locationID, locationIDErr := uuid.Parse(strings.TrimSpace(multipartValue(request, "location_id")))
	if locationIDErr != nil {
		validationErr.Add("location_id", "is invalid", "invalid")
	}

	direction, directionErr := domain.ParseCheckinDirection(strings.TrimSpace(multipartValue(request, "direction")))
	if directionErr != nil {
		validationErr.Add("direction", "is invalid", "invalid")
	}

	notes := strings.TrimSpace(multipartValue(request, "notes"))
	photoContent, fileErr := readMultipartFile(request, "photo", false, validationErr)
	if fileErr != nil {
		return checkinCreateRequest{}, fileErr
	}

	if validationErr.HasFieldErrors() {
		return checkinCreateRequest{}, validationErr
	}

	return checkinCreateRequest{
		UserID:       userID,
		LocationID:   locationID,
		Direction:    direction,
		Notes:        notes,
		PhotoContent: photoContent,
	}, nil
}

func parseMultipartForm(writer http.ResponseWriter, request *http.Request) error {
	if !strings.HasPrefix(request.Header.Get("Content-Type"), "multipart/form-data") {
		return badRequestError("request body must be multipart/form-data")
	}

	maxMultipartBodyBytes := int64(maxJSONBodyBytes * multipartBodyMultiple)
	request.Body = http.MaxBytesReader(writer, request.Body, maxMultipartBodyBytes)
	if parseErr := request.ParseMultipartForm(maxMultipartBodyBytes); parseErr != nil {
		return badRequestError("request body is invalid")
	}

	return nil
}

func multipartValue(request *http.Request, field string) string {
	if request.MultipartForm == nil {
		return ""
	}
	values := request.MultipartForm.Value[field]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func optionalStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func readMultipartFile(
	request *http.Request,
	field string,
	requireFile bool,
	validationErr *domain.ValidationError,
) ([]byte, error) {
	file, _, fileErr := request.FormFile(field)
	switch {
	case errors.Is(fileErr, http.ErrMissingFile) && requireFile:
		validationErr.Add(field, "is required", "required")
		return nil, nil
	case fileErr != nil && !errors.Is(fileErr, http.ErrMissingFile):
		return nil, badRequestError("request body is invalid")
	case errors.Is(fileErr, http.ErrMissingFile):
		return nil, nil
	}

	defer file.Close()

	maxMultipartBodyBytes := int64(maxJSONBodyBytes * multipartBodyMultiple)
	content, readErr := io.ReadAll(io.LimitReader(file, maxMultipartBodyBytes+1))
	if readErr != nil {
		return nil, readErr
	}
	if len(content) == 0 {
		validationErr.Add(field, "must not be empty", "required")
	}
	if int64(len(content)) > maxMultipartBodyBytes {
		validationErr.Add(field, "is too large", "invalid")
	}

	return content, nil
}

func validatePermissionGrants(
	access []PermissionGrant,
	detail string,
) ([]domain.PermissionGrant, error) {
	validationErr := &domain.ValidationError{Code: "validation_error", Detail: detail}
	grants := make([]domain.PermissionGrant, 0, len(access))
	for index, permission := range access {
		resource, err := domain.ParsePermissionResource(string(permission.Resource))
		if err != nil {
			return nil, badRequestError("invalid access resource")
		}
		action, err := domain.ParsePermissionAction(string(permission.Action))
		if err != nil {
			return nil, badRequestError("invalid access action")
		}

		locationID := uuidPointer(permission.LocationId)
		assetType := assetTypePointer(permission.AssetType)
		fieldPrefix := fmt.Sprintf("access.%d.location_id", index)
		assetTypeField := fmt.Sprintf("access.%d.asset_type", index)
		if resource == domain.PermissionResourceCheckins && locationID == nil {
			validationErr.Add(fieldPrefix, "is required for checkin permissions", "required")
		}
		if resource != domain.PermissionResourceCheckins && locationID != nil {
			validationErr.Add(fieldPrefix, "must be empty unless resource is checkins", "invalid")
		}
		if resource == domain.PermissionResourceAssets && assetType == nil {
			validationErr.Add(assetTypeField, "is required for asset permissions", "required")
		}
		if resource != domain.PermissionResourceAssets && assetType != nil {
			validationErr.Add(assetTypeField, "must be empty unless resource is assets", "invalid")
		}

		grants = append(grants, domain.PermissionGrant{
			Resource:   resource,
			Action:     action,
			LocationID: locationID,
			AssetType:  assetType,
		})
	}

	if validationErr.HasFieldErrors() {
		return nil, validationErr
	}

	return grants, nil
}
