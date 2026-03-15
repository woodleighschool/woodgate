import type { CheckinDirection } from "@/api/types";

export const CHECKIN_DIRECTION_CHOICES = [
  { id: "check_in", name: "Check in" },
  { id: "check_out", name: "Check out" },
] satisfies { id: CheckinDirection; name: string }[];
