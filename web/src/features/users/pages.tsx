import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "@tanstack/react-router";
import { UserRound } from "lucide-react";

import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import { AccessEditor } from "@features/access/access-editor";
import { useCanAccess } from "@features/auth/access";
import { CheckinsTable } from "@features/checkins/pages";
import { MembershipsTable } from "@features/resources/memberships";
import { DateValue, exportRows } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import { getUser, listUsers, unwrap, type User } from "@lib/api";

const columns: DataTableColumnDef<User>[] = [
  {
    accessorKey: "display_name",
    header: "Name",
    cell: ({ row }) => (
      <Link to="/users/$id/show" params={{ id: row.original.id }} data-text-link>
        {row.original.display_name}
      </Link>
    ),
  },
  { accessorKey: "upn", header: "UPN" },
  { accessorKey: "department", header: "Department" },
  {
    accessorKey: "source",
    header: "Source",
    cell: ({ row }) => (row.original.source === "entra" ? "Entra ID" : "Local"),
  },
  {
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => <DateValue value={row.original.updated_at} />,
  },
];
export function UsersList() {
  const { tableSearch, params, search } = useResourceTable("display_name.asc");
  const filters = {
    ...params,
    location_id: typeof search.location_id === "string" ? search.location_id : undefined,
  };
  const query = useQuery({
    queryKey: ["users", "list", filters],
    queryFn: ({ signal }) => unwrap(listUsers({ query: filters, signal })),
    placeholderData: keepPreviousData,
  });
  return (
    <PageShell>
      <PageHeader title="Users" icon={<UserRound />} />
      <ResourceDataTable<User>
        data={query.data?.rows ?? []}
        count={query.data?.total ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isPending}
        pending={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<UserRound />}
        emptyTitle="Users"
        emptyDescription="No users are available."
        exportOptions={{
          filename: "users",
          columns: (
            [
              "id",
              "display_name",
              "upn",
              "department",
              "source",
              "admin",
              "access",
              "created_at",
              "updated_at",
            ] as const
          ).map((key) => ({
            header: key,
            value: (row) =>
              Array.isArray(row[key]) ? JSON.stringify(row[key]) : String(row[key] ?? ""),
          })),
          loadRows: () =>
            exportRows((offset) =>
              unwrap(listUsers({ query: { ...filters, offset, limit: 250 } })),
            ),
        }}
      />
    </PageShell>
  );
}

export function UserShow({ id }: { id: string }) {
  const query = useQuery({
    queryKey: ["users", id],
    queryFn: ({ signal }) => unwrap(getUser({ path: { id }, signal })),
  });
  const canGroups = useCanAccess("groups", "read");
  const canCheckins = useCanAccess("checkins", "read");
  const canWrite = useCanAccess("users", "write");
  const navigate = useNavigate();
  const params = useParams({ strict: false }) as { tab?: string };
  const tabs = [
    "overview",
    ...(canGroups ? ["groups"] : []),
    ...(canCheckins ? ["checkins"] : []),
    ...(canWrite ? ["access"] : []),
  ];
  const tab = tabs[Number(params.tab ?? 0)] ?? "overview";
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
        title={record.display_name}
        icon={<UserRound />}
        actions={
          <Button nativeButton={false} variant="outline" render={<Link to="/users" />}>
            List
          </Button>
        }
      />
      <Tabs
        value={tab}
        onValueChange={(value) => {
          const index = tabs.indexOf(String(value));
          void navigate({
            to: index > 0 ? "/users/$id/show/$tab" : "/users/$id/show",
            params: { id, tab: String(index) },
            search: {},
          });
        }}
      >
        <TabsList variant="line">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          {canGroups ? <TabsTrigger value="groups">Groups</TabsTrigger> : null}
          {canCheckins ? <TabsTrigger value="checkins">Check-ins</TabsTrigger> : null}
          {canWrite ? <TabsTrigger value="access">Access</TabsTrigger> : null}
        </TabsList>
        <TabsContent value="overview">
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={record.display_name} />
            <KeyValueRow label="UPN" value={record.upn} />
            <KeyValueRow label="Department" value={record.department} />
            <KeyValueRow label="Source" value={record.source === "entra" ? "Entra ID" : "Local"} />
            <KeyValueRow label="Created" value={<DateValue value={record.created_at} />} />
            <KeyValueRow label="Updated" value={<DateValue value={record.updated_at} />} />
          </KeyValueSection>
        </TabsContent>
        {canGroups ? (
          <TabsContent value="groups">
            <MembershipsTable userId={id} />
          </TabsContent>
        ) : null}
        {canCheckins ? (
          <TabsContent value="checkins">
            <CheckinsTable userId={id} />
          </TabsContent>
        ) : null}
        {canWrite ? (
          <TabsContent value="access">
            <AccessEditor resource="users" record={record} />
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}
