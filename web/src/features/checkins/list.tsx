import { getRouteApi } from "@tanstack/react-router";
import { addDays, format, isValid, parseISO, startOfDay, subMonths } from "date-fns";
import { ClipboardCheck } from "lucide-react";
import type { DateRange } from "react-day-picker";

import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { DateRangePicker } from "@components/date-range-picker";
import { FacetedFilter } from "@components/faceted-filter";
import { FilterChip } from "@components/filter-controls";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { TextLink } from "@components/link";
import { RelativeTime } from "@components/relative-time";
import { ResourceDataTable } from "@components/resource-data-table";
import { Badge } from "@components/ui/badge";
import { checkinPersonLabel } from "@features/checkins/presentation";
import {
  useCheckins,
  useCheckinDepartments,
  useCheckinLocations,
  useCheckinUser,
} from "@features/resources/queries";
import type { Checkin } from "@lib/api";
import { nonEmpty } from "@lib/utils";

const routeApi = getRouteApi("/_authenticated/checkins/");

const CHECKIN_FILTER_KEYS = [
  { id: "departments", multiple: true },
  { id: "location_id" },
  { id: "direction" },
] as const;
const DIRECTION_OPTIONS = [
  { value: "check_in", label: "Check in" },
  { value: "check_out", label: "Check out" },
] as const;

const columns: DataTableColumnDef<Checkin>[] = [
  {
    id: "user",
    accessorFn: (checkin) => checkinPersonLabel(checkin.person),
    header: "Person",
    cell: ({ row }) => (
      <TextLink to="/checkins" search={{ user_id: row.original.person.id }} className="font-medium">
        {checkinPersonLabel(row.original.person)}
      </TextLink>
    ),
    meta: { label: "Person" },
  },
  {
    id: "department",
    accessorFn: (checkin) => checkin.person.department,
    header: "Department",
    cell: ({ row }) => nonEmpty(row.original.person.department) ?? "-",
    meta: { label: "Department" },
  },
  {
    id: "location",
    accessorFn: (checkin) => checkin.location.name,
    header: "Location",
    cell: ({ row }) => row.original.location.name,
    meta: { label: "Location" },
  },
  {
    id: "direction",
    accessorKey: "direction",
    header: "Direction",
    cell: ({ row }) => (
      <Badge variant="secondary">
        {row.original.direction === "check_in" ? "Check in" : "Check out"}
      </Badge>
    ),
    meta: { label: "Direction" },
  },
  {
    id: "created_at",
    accessorKey: "created_at",
    header: "Time",
    cell: ({ row }) => (
      <TextLink to="/checkins/$id" params={{ id: String(row.original.id) }}>
        <RelativeTime value={row.original.created_at} />
      </TextLink>
    ),
    meta: { label: "Time" },
  },
];

export function CheckinListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: CHECKIN_FILTER_KEYS,
    scopeKeys: ["user_id", "from", "to"],
  });
  const departments = useCheckinDepartments();
  const locations = useCheckinLocations();
  const user = useCheckinUser(search.user_id ?? null);
  const userLabel =
    search.user_id === undefined
      ? undefined
      : user.data
        ? checkinPersonLabel(user.data)
        : "Selected user";
  const bounds = checkinBounds(search.from, search.to);
  const query = useCheckins({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    departments: search.departments,
    location_id: search.location_id,
    user_id: search.user_id,
    direction: search.direction,
    created_from: bounds.createdFrom,
    created_before: bounds.createdBefore,
  });
  const updateFilters = (next: Partial<typeof search>) => {
    void navigate({
      replace: true,
      search: (previous) => ({ ...previous, ...next, page: 1 }),
    });
  };
  return (
    <PageShell>
      <PageHeader
        title="Check-ins"
        description="Review arrival and departure records."
        context={
          userLabel ? (
            <FilterChip
              label="User"
              value={userLabel}
              onRemove={() => tableSearch.clearSearchKeys(["user_id"])}
            />
          ) : null
        }
      />
      <ResourceDataTable
        data={query.data?.items ?? []}
        count={query.data?.count ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isLoading}
        pending={query.isPlaceholderData}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<ClipboardCheck />}
        emptyTitle="Check-ins"
        emptyDescription="Check-in activity appears here."
        filters={
          <>
            <FacetedFilter
              title="Department"
              options={(departments.data ?? []).map((department) => ({
                value: department,
                label: department,
              }))}
              value={search.departments ?? []}
              multiple
              onValueChange={(selected) =>
                updateFilters({ departments: selected.length ? selected : undefined })
              }
            />
            <FacetedFilter
              title="Location"
              options={(locations.data ?? []).map((location) => ({
                value: String(location.id),
                label: location.name,
              }))}
              value={search.location_id ? [String(search.location_id)] : []}
              multiple={false}
              onValueChange={(selected) =>
                updateFilters({
                  location_id: selected.length ? Number(selected.at(-1)) : undefined,
                })
              }
            />
            <FacetedFilter
              title="Direction"
              options={[...DIRECTION_OPTIONS]}
              value={search.direction ? [search.direction] : []}
              multiple={false}
              onValueChange={(selected) =>
                updateFilters({ direction: checkinDirection(selected.at(-1)) })
              }
            />
            <DateRangePicker
              value={parseDateRange(search.from, search.to)}
              defaultMonth={subMonths(new Date(), 1)}
              disabled={{ after: new Date() }}
              onValueChange={(range) =>
                updateFilters({
                  from: range?.from ? format(range.from, "yyyy-MM-dd") : undefined,
                  to: range?.to ? format(range.to, "yyyy-MM-dd") : undefined,
                })
              }
            />
          </>
        }
      />
    </PageShell>
  );
}

function parseDateRange(fromValue: string | undefined, toValue: string | undefined): DateRange {
  return { from: parseDate(fromValue), to: parseDate(toValue) };
}

function parseDate(value: string | undefined): Date | undefined {
  if (!value) return undefined;
  const date = parseISO(value);
  return isValid(date) ? date : undefined;
}

function checkinBounds(
  fromValue: string | undefined,
  toValue: string | undefined,
): { createdFrom?: string; createdBefore?: string } {
  return {
    createdFrom: fromValue ? startOfDay(parseISO(fromValue)).toISOString() : undefined,
    createdBefore: toValue ? addDays(startOfDay(parseISO(toValue)), 1).toISOString() : undefined,
  };
}

function checkinDirection(value: string | undefined): "check_in" | "check_out" | undefined {
  return value === "check_in" || value === "check_out" ? value : undefined;
}
