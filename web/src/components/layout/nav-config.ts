import { linkOptions, type ActiveOptions } from "@tanstack/react-router";
import {
  ClipboardCheck,
  MapPin,
  MonitorSmartphone,
  ShieldCheck,
  type LucideIcon,
  UsersRound,
} from "lucide-react";

export interface NavItem {
  label: string;
  to?: string;
  activeOptions?: ActiveOptions;
  icon?: LucideIcon;
  disabled?: boolean;
  items?: readonly NavItem[];
}

export interface NavMenu {
  label: string;
  items: readonly NavItem[];
}

export const navSections: NavMenu[] = [
  {
    label: "Operations",
    items: linkOptions([
      { label: "Check-ins", to: "/checkins", icon: ClipboardCheck },
      { label: "Locations", to: "/locations", icon: MapPin },
      { label: "Stations", to: "/stations", icon: MonitorSmartphone },
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
          { label: "Overview", to: "/directory", activeOptions: { exact: true } },
          { label: "Users", to: "/directory/users" },
          { label: "Groups", to: "/directory/groups" },
        ]),
      },
      {
        label: "Access Control",
        to: "/access",
        activeOptions: { exact: true },
        icon: ShieldCheck,
        items: linkOptions([
          { label: "Roles", to: "/access/roles" },
          { label: "Assignments", to: "/access/assignments" },
        ]),
      },
    ],
  },
];
