import { Badge } from "@components/ui/badge";
import type { PermissionLevel } from "@lib/api";

const labels: Record<PermissionLevel, string> = {
  none: "None",
  view: "View",
  edit: "Edit",
};

export function PermissionLevelBadge({ level }: { level: PermissionLevel }) {
  return <Badge variant={level === "none" ? "outline" : "secondary"}>{labels[level]}</Badge>;
}

export function permissionLabel(value: string): string {
  return value
    .replaceAll(".", " ")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}
