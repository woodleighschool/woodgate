import { queryOptions, useQuery } from "@tanstack/react-query";

import type { PermissionAction } from "@lib/api";
import { ApiError } from "@lib/api";

import { authApi, type AuthPermissions } from "./api";

export const permissionsQuery = queryOptions({
  queryKey: ["auth", "permissions"],
  queryFn: ({ signal }) => authApi.getPermissions(signal),
  staleTime: 30_000,
  retry: (count, error) =>
    !(error instanceof ApiError && (error.status === 401 || error.status === 403)) && count < 2,
});

export function usePermissions() {
  return useQuery(permissionsQuery).data;
}

export function canAccess(
  permissions: AuthPermissions | undefined,
  resource: string,
  action: PermissionAction,
) {
  if (!permissions) return false;
  return (
    permissions.admin ||
    (permissions.capabilities[
      resource === "api-keys" ? "api_keys" : resource === "group-memberships" ? "groups" : resource
    ]?.[action] ??
      false)
  );
}

export function useCanAccess(resource: string, action: PermissionAction) {
  return canAccess(usePermissions(), resource, action);
}
