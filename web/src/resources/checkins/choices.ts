import type { CheckinDirection } from "@/api/types";

export const CHECKIN_DIRECTION_LABELS = {
  check_in: "Check in",
  check_out: "Check out",
} satisfies Record<CheckinDirection, string>;

export const CHECKIN_DIRECTION_CHOICES = [
  { id: "check_in", name: CHECKIN_DIRECTION_LABELS.check_in },
  { id: "check_out", name: CHECKIN_DIRECTION_LABELS.check_out },
] satisfies { id: CheckinDirection; name: string }[];
