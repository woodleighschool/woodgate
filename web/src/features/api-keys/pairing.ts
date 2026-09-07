import type { PermissionGrant } from "@lib/api-client/types.gen";

export function appKeyGrants(locationId: string): PermissionGrant[] {
  return [
    { resource: "locations", action: "read" },
    { resource: "users", action: "read" },
    { resource: "assets", action: "read", asset_type: "asset" },
    { resource: "checkins", action: "create", location_id: locationId },
  ];
}

export function pairingPayload(secret: string, baseUrl: string): string {
  return JSON.stringify({ api_key: secret, base_url: baseUrl });
}
