import { addDays, parseISO, startOfDay } from "date-fns";
import type { DateRange } from "react-day-picker";

export function checkinDateRange(
  search: { from?: string; to?: string; period?: "all" },
  now = new Date(),
): DateRange | undefined {
  if (search.from) {
    return { from: parseISO(search.from), to: search.to ? parseISO(search.to) : undefined };
  }
  if (search.period === "all") return undefined;
  const today = startOfDay(now);
  return { from: today, to: today };
}

export function checkinBounds(range: DateRange | undefined): {
  createdFrom?: string;
  createdBefore?: string;
} {
  return {
    createdFrom: range?.from ? startOfDay(range.from).toISOString() : undefined,
    createdBefore: range?.to ? addDays(startOfDay(range.to), 1).toISOString() : undefined,
  };
}
