import type { components } from "@/api/openapi";

export type Source = components["schemas"]["Source"];
export type PermissionSubjectKind = components["schemas"]["PermissionSubjectKind"];
export type PermissionResource = components["schemas"]["PermissionResource"];
export type PermissionAction = components["schemas"]["PermissionAction"];
export type AssetType = components["schemas"]["AssetType"];
export type CheckinDirection = components["schemas"]["CheckinDirection"];

export type User = components["schemas"]["User"];
export type UserListResponse = components["schemas"]["UserListResponse"];

export type Group = components["schemas"]["Group"];
export type GroupListResponse = components["schemas"]["GroupListResponse"];

export type GroupMembership = components["schemas"]["GroupMembership"];
export type GroupMembershipListResponse = components["schemas"]["GroupMembershipListResponse"];

export type Asset = components["schemas"]["Asset"];
export type AssetListResponse = components["schemas"]["AssetListResponse"];
export type AssetCreateRequest = components["schemas"]["AssetCreateRequest"];
export type AssetUpdateRequest = components["schemas"]["AssetUpdateRequest"];

export type Location = components["schemas"]["Location"];
export type LocationListResponse = components["schemas"]["LocationListResponse"];
export type LocationWriteRequest = components["schemas"]["LocationWriteRequest"];

export type Checkin = components["schemas"]["Checkin"];
export type CheckinListResponse = components["schemas"]["CheckinListResponse"];
export type CheckinCreateRequest = components["schemas"]["CheckinCreateRequest"];

export type PermissionGrant = components["schemas"]["PermissionGrant"];
export type UserAccessWriteRequest = components["schemas"]["UserAccessWriteRequest"];

export type APIKey = components["schemas"]["APIKey"];
export type APIKeyListResponse = components["schemas"]["APIKeyListResponse"];
export type APIKeyAccessWriteRequest = components["schemas"]["APIKeyAccessWriteRequest"];
export type APIKeyCreateRequest = components["schemas"]["APIKeyCreateRequest"];
export type CreateAPIKeyData = components["schemas"]["CreateAPIKeyData"];
