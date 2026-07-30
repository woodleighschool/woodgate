import type { AuthPermissions, ResourceCapability } from "@/api/authClient";

const resourceKeyByName: Record<string, string> = {
  users: "users",
  groups: "groups",
  "group-memberships": "groups",
  locations: "locations",
  checkins: "checkins",
  assets: "assets",
  "api-keys": "api_keys",
};

const emptyCapability: ResourceCapability = {
  read: false,
  create: false,
  write: false,
  delete: false,
};

export const capabilityFor = (permissions: AuthPermissions | undefined, resourceName: string): ResourceCapability => {
  if (!permissions) {
    return emptyCapability;
  }
  if (permissions.admin) {
    return { read: true, create: true, write: true, delete: true };
  }

  const resourceKey = resourceKeyByName[resourceName] ?? resourceName;
  return permissions.capabilities[resourceKey] ?? emptyCapability;
};

const capabilityKeyForAction = (action: string): keyof ResourceCapability | undefined => {
  switch (action) {
    case "list":
    case "show":
    case "read": {
      return "read";
    }
    case "create": {
      return "create";
    }
    case "edit":
    case "write": {
      return "write";
    }
    case "delete": {
      return "delete";
    }
    default: {
      return undefined;
    }
  }
};

export const canAccess = (permissions: AuthPermissions | undefined, resourceName: string, action: string): boolean => {
  const capabilityKey = capabilityKeyForAction(action);
  if (!capabilityKey) {
    return false;
  }

  return capabilityFor(permissions, resourceName)[capabilityKey];
};
