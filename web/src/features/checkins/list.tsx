import { getRouteApi } from "@tanstack/react-router";
import { addDays, format, isValid, parseISO, startOfDay, subMonths } from "date-fns";
import { ClipboardCheck, X } from "lucide-react";
import type { DateRange } from "react-day-picker";

import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { DateRangePicker } from "@components/date-range-picker";
import { FacetedFilter } from "@components/faceted-filter";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { TextLink } from "@components/link";
import { ResourceDataTable } from "@components/resource-data-table";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import { useCheckins } from "@features/resources/queries";
import type { Checkin } from "@lib/api";

const routeApi = getRouteApi("/_authenticated/checkins/");

const CHECKIN_FILTER_KEYS = [{ id: "direction" }] as const;
const DIRECTION_OPTIONS = [
  { value: "check_in", label: "Check in" },
  { value: "check_out", label: "Check out" },
] as const;

const columns: DataTableColumnDef<Checkin>[] = [
  {
    id: "user",
    accessorFn: (row) => row.user_name || row.user_id,
    header: "Person",
    cell: ({ row }) => (
      <TextLink to="/checkins/$id" params={{ id: String(row.original.id) }} className="font-medium">
        {row.original.user_name || `User #${row.original.user_id}`}
      </TextLink>
    ),
    meta: { label: "Person" },
  },
  {
    id: "location",
    accessorFn: (row) => row.location_name || row.location_id,
    header: "Location",
    cell: ({ row }) => row.original.location_name || "-",
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
    cell: ({ row }) => new Date(row.original.created_at).toLocaleString(),
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
    scopeKeys: ["from", "to"],
  });
  const bounds = checkinBounds(search.from, search.to);
  const query = useCheckins({
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
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
      <PageHeader title="Check-ins" description="Review arrival and departure records." />
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
            {tableSearch.isFiltered ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() =>
                  void navigate({
                    replace: true,
                    search: { page: 1, per_page: search.per_page, sort: search.sort },
                  })
                }
              >
                <X data-icon="inline-start" />
                Reset
              </Button>
            ) : null}
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
