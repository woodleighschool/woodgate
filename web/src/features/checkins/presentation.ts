import type { Checkin } from "@lib/api";
import { enumOptions } from "@lib/enum-metadata";
import { nonEmpty } from "@lib/utils";

export function checkinPersonLabel(person: { name: string; email: string }): string {
  return nonEmpty(person.name) ?? person.email;
}

const CHECKIN_DIRECTIONS = {
  check_in: { name: "Check In", variant: "success" },
  check_out: { name: "Check Out", variant: "info" },
} as const;

export const CHECKIN_DIRECTION_OPTIONS = enumOptions(CHECKIN_DIRECTIONS, ["check_in", "check_out"]);

export function checkinDirectionMetadata(direction: Checkin["direction"]) {
  return CHECKIN_DIRECTIONS[direction === "check_in" ? "check_in" : "check_out"];
}
