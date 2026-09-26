import { linkOptions, type ActiveOptions } from "@tanstack/react-router";
import {
  ClipboardCheck,
  MapPin,
  MonitorSmartphone,
  type LucideIcon,
  UsersRound,
} from "lucide-react";

import type { PermissionRequirement } from "@features/authz/permissions";
import type { Account } from "@lib/api";

import { filterNavigation, firstNavigationTarget } from "./navigation";

export interface NavItem {
  label: string;
  to?: string;
  activeOptions?: ActiveOptions;
  icon?: LucideIcon;
  disabled?: boolean;
  permission?: PermissionRequirement;
  items?: readonly NavItem[];
}

export interface NavMenu {
  label: string;
  items: readonly NavItem[];
}

const navSections: NavMenu[] = [
  {
    label: "Operations",
    items: linkOptions([
      {
        label: "Check-ins",
        to: "/checkins",
        icon: ClipboardCheck,
        permission: { resource: "checkins", access: "view" },
      },
      {
        label: "Locations",
        to: "/locations",
        icon: MapPin,
        permission: { resource: "locations", access: "view" },
      },
      {
        label: "Stations",
        to: "/stations",
        icon: MonitorSmartphone,
        permission: { resource: "stations", access: "view" },
      },
    ]),
  },
  {
    label: "System",
    items: [
      {
        label: "Directory",
        to: "/directory",
        activeOptions: { exact: true },
        icon: UsersRound,
        items: linkOptions([
          {
            label: "Overview",
            to: "/directory",
            activeOptions: { exact: true },
            permission: { resource: "directory", access: "view" },
          },
          {
            label: "Users",
            to: "/directory/users",
            permission: { resource: "users", access: "view" },
          },
          {
            label: "Groups",
            to: "/directory/groups",
            permission: { resource: "groups", access: "view" },
          },
        ]),
      },
    ],
  },
];

export function visibleNavSections(account: Account | undefined): NavMenu[] {
  return navSections
    .map((section) => ({
      ...section,
      items: filterNavigation(section.items, account?.effective_permissions),
    }))
    .filter((section) => section.items.length > 0);
}

export function firstAccessiblePath(account: Account | undefined): string | undefined {
  return firstNavigationTarget(visibleNavSections(account).flatMap((section) => section.items));
}
