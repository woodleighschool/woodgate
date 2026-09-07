import type { PermissionGrant } from "@lib/api-client/types.gen";

export const permissionActions = ["read", "create", "write", "delete"] as const;

export function normalizeAccess(grants: PermissionGrant[] = []): PermissionGrant[] {
  return grants.toSorted((left, right) => grantKey(left).localeCompare(grantKey(right)));
}

function grantKey(grant: PermissionGrant): string {
  return `${grant.resource}:${grant.location_id ?? ""}:${grant.asset_type ?? ""}:${grant.action}`;
}

export function accessSignature(admin: boolean, access: PermissionGrant[] = []): string {
  return JSON.stringify({ admin, access: normalizeAccess(access) });
}

export function hasGrant(grants: PermissionGrant[], target: PermissionGrant): boolean {
  return grants.some((grant) => grantKey(grant) === grantKey(target));
}

export function toggleGrant(
  grants: PermissionGrant[],
  target: PermissionGrant,
  enabled: boolean,
): PermissionGrant[] {
  const next = grants.filter((grant) => grantKey(grant) !== grantKey(target));
  return normalizeAccess(enabled ? [...next, target] : next);
}

export function locationsForAccess(
  grants: PermissionGrant[],
  locations: { id: string; name: string }[],
): { id: string; name: string }[] {
  const records = new Map(locations.map((location) => [location.id, location]));
  for (const grant of grants) {
    if (grant.resource === "checkins" && grant.location_id && !records.has(grant.location_id)) {
      records.set(grant.location_id, { id: grant.location_id, name: grant.location_id });
    }
  }
  return [...records.values()].toSorted((left, right) => left.name.localeCompare(right.name));
}
