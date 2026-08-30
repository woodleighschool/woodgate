import type { Account, PermissionLevel } from "@lib/api";

const levels: Record<PermissionLevel, number> = {
  none: 0,
  view: 1,
  edit: 2,
};

export function canAccess(
  account: Account | undefined,
  resource: string,
  required: PermissionLevel,
): boolean {
  const granted = permissionLevel(account?.effective_permissions?.[resource]);
  return levels[granted] >= levels[required];
}

export function permissionLevel(value: string | undefined): PermissionLevel {
  if (value === "view" || value === "edit") return value;
  return "none";
}
