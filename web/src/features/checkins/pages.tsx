import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ClipboardList } from "lucide-react";

import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
import { ChoiceSelect, DateValue, exportRows, ReferenceName } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import {
  getCheckin,
  listCheckinDepartments,
  listCheckins,
  unwrap,
  type Checkin,
  type ListCheckinsData,
} from "@lib/api";

const directions = { check_in: "Check In", check_out: "Check Out" };
const columns: DataTableColumnDef<Checkin>[] = [
  {
    accessorKey: "photo_url",
    header: "Photo",
    size: 90,
    enableSorting: false,
    cell: ({ row }) =>
      row.original.photo_url ? (
        <img
          className="size-14 rounded-sm object-cover"
          src={row.original.photo_url}
          alt={row.original.user_display_name}
        />
      ) : (
        "-"
      ),
  },
  {
    accessorKey: "user_display_name",
    header: "User",
    cell: ({ row }) => (
      <Link to="/checkins/$id/show" params={{ id: row.original.id }} data-text-link>
        {row.original.user_display_name}
      </Link>
    ),
  },
  { accessorKey: "department", header: "Department" },
  { accessorKey: "location_name", header: "Location" },
  {
    accessorKey: "direction",
    header: "Direction",
    cell: ({ row }) => directions[row.original.direction],
  },
  { accessorKey: "notes", header: "Notes" },
  {
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => <DateValue value={row.original.created_at} />,
  },
];

export function CheckinsTable({ userId }: { userId?: string }) {
  const { tableSearch, params, search, setFilter } = useResourceTable("created_at.desc");
  const filters: NonNullable<ListCheckinsData["query"]> = {
    ...params,
    user_id: userId ?? (typeof search.user_id === "string" ? search.user_id : undefined),
    location_id: typeof search.location_id === "string" ? search.location_id : undefined,
    department: typeof search.department === "string" ? search.department : undefined,
    direction:
      search.direction === "check_in" || search.direction === "check_out"
        ? search.direction
        : undefined,
    created_from: typeof search.created_from === "string" ? search.created_from : undefined,
    created_to: typeof search.created_to === "string" ? search.created_to : undefined,
  };
  const query = useQuery({
    queryKey: ["checkins", "list", filters],
    queryFn: ({ signal }) => unwrap(listCheckins({ query: filters, signal })),
    placeholderData: keepPreviousData,
  });
  const departments = useQuery({
    queryKey: ["checkins", "departments"],
    queryFn: ({ signal }) => unwrap(listCheckinDepartments({ signal })),
  });
  return (
    <>
      <QueryError
        title="Failed to load departments"
        error={departments.error}
        onRetry={() => void departments.refetch()}
      />
      <ResourceDataTable<Checkin>
        data={query.data?.rows ?? []}
        count={query.data?.total ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isPending}
        pending={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<ClipboardList />}
        emptyTitle="Check-ins"
        emptyDescription="No check-ins have been recorded."
        filters={
          <>
            <ChoiceSelect
              label="Department"
              value={typeof search.department === "string" ? search.department : "all"}
              onChange={(value) => setFilter("department", value === "all" ? undefined : value)}
              options={[
                { value: "all", label: "All departments" },
                ...(departments.data?.rows ?? []).map((row) => ({
                  value: row.id,
                  label: row.name,
                })),
              ]}
            />
            <ChoiceSelect
              label="Direction"
              value={typeof search.direction === "string" ? search.direction : "all"}
              onChange={(value) => setFilter("direction", value === "all" ? undefined : value)}
              options={[
                { value: "all", label: "All directions" },
                { value: "check_in", label: "Check In" },
                { value: "check_out", label: "Check Out" },
              ]}
            />
          </>
        }
        exportOptions={{
          filename: "checkins",
          columns: (
            [
              "id",
              "user_id",
              "user_display_name",
              "department",
              "location_id",
              "location_name",
              "direction",
              "notes",
              "asset_id",
              "photo_url",
              "created_by_kind",
              "created_by_id",
              "created_at",
            ] as const
          ).map((key) => ({ header: key, value: (row) => row[key] })),
          loadRows: () =>
            exportRows((offset) =>
              unwrap(listCheckins({ query: { ...filters, offset, limit: 250 } })),
            ),
        }}
      />
    </>
  );
}

export function CheckinsList() {
  return (
    <PageShell>
      <PageHeader title="Check-ins" icon={<ClipboardList />} />
      <CheckinsTable />
    </PageShell>
  );
}

export function CheckinShow({ id }: { id: string }) {
  const query = useQuery({
    queryKey: ["checkins", id],
    queryFn: ({ signal }) => unwrap(getCheckin({ path: { id }, signal })),
  });
  if (query.isPending)
    return (
      <PageShell>
        <Skeleton className="h-80 w-full" />
      </PageShell>
    );
  if (query.isError)
    return (
      <PageShell>
        <QueryError error={query.error} onRetry={() => void query.refetch()} />
      </PageShell>
    );
  const record = query.data;
  return (
    <PageShell>
      <PageHeader
        title="Check-in"
        icon={<ClipboardList />}
        actions={
          <Button nativeButton={false} variant="outline" render={<Link to="/checkins" />}>
            List
          </Button>
        }
      />
      <KeyValueSection title="Overview">
        <KeyValueRow
          label="User"
          value={
            <ReferenceName
              resource="users"
              id={record.user_id}
              fallback={record.user_display_name}
            />
          }
        />
        <KeyValueRow label="Department" value={record.department} />
        <KeyValueRow
          label="Location"
          value={
            <ReferenceName
              resource="locations"
              id={record.location_id}
              fallback={record.location_name}
            />
          }
        />
        <KeyValueRow label="Direction" value={directions[record.direction]} />
        <KeyValueRow label="Notes" value={record.notes} />
        <KeyValueRow
          label="Photo"
          value={
            record.photo_url ? (
              <a href={record.photo_url} target="_blank" rel="noreferrer">
                <img
                  src={record.photo_url}
                  alt={record.user_display_name}
                  className="max-h-96 max-w-full rounded-sm object-contain"
                />
              </a>
            ) : undefined
          }
        />
        <KeyValueRow
          label={record.created_by_kind === "user" ? "Created By User" : "Created By API Key"}
          value={
            <ReferenceName
              resource={record.created_by_kind === "user" ? "users" : "api-keys"}
              id={record.created_by_id}
            />
          }
        />
        <KeyValueRow label="Created" value={<DateValue value={record.created_at} />} />
      </KeyValueSection>
    </PageShell>
  );
}
