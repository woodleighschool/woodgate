import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "@tanstack/react-router";
import { Users } from "lucide-react";

import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { Skeleton } from "@components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import { useCanAccess } from "@features/auth/access";
import { MembershipsTable } from "@features/resources/memberships";
import { DateValue, exportRows } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import { getGroup, listGroups, unwrap, type Group } from "@lib/api";

const columns: DataTableColumnDef<Group>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <Link to="/groups/$id/show" params={{ id: row.original.id }} data-text-link>
        {row.original.name}
      </Link>
    ),
  },
  { accessorKey: "description", header: "Description" },
  { accessorKey: "member_count", header: "Members" },
];
export function GroupsList() {
  const { tableSearch, params } = useResourceTable("name.asc");
  const query = useQuery({
    queryKey: ["groups", "list", params],
    queryFn: ({ signal }) => unwrap(listGroups({ query: params, signal })),
    placeholderData: keepPreviousData,
  });
  return (
    <PageShell>
      <PageHeader title="Groups" icon={<Users />} />
      <ResourceDataTable<Group>
        data={query.data?.rows ?? []}
        count={query.data?.total ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isPending}
        pending={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<Users />}
        emptyTitle="Groups"
        emptyDescription="No groups are available."
        exportOptions={{
          filename: "groups",
          columns: (
            ["id", "name", "description", "member_count", "created_at", "updated_at"] as const
          ).map((key) => ({ header: key, value: (row) => row[key] })),
          loadRows: () =>
            exportRows((offset) =>
              unwrap(listGroups({ query: { ...params, offset, limit: 250 } })),
            ),
        }}
      />
    </PageShell>
  );
}
export function GroupShow({ id }: { id: string }) {
  const query = useQuery({
    queryKey: ["groups", id],
    queryFn: ({ signal }) => unwrap(getGroup({ path: { id }, signal })),
  });
  const canGroups = useCanAccess("groups", "read");
  const navigate = useNavigate();
  const params = useParams({ strict: false }) as { tab?: string };
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
        title={record.name}
        icon={<Users />}
        actions={
          <Button nativeButton={false} variant="outline" render={<Link to="/groups" />}>
            List
          </Button>
        }
      />
      <Tabs
        value={canGroups && params.tab === "1" ? "users" : "overview"}
        onValueChange={(value) => {
          void navigate({
            to: value === "users" ? "/groups/$id/show/$tab" : "/groups/$id/show",
            params: { id, tab: "1" },
            search: {},
          });
        }}
      >
        <TabsList variant="line">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          {canGroups ? <TabsTrigger value="users">Users</TabsTrigger> : null}
        </TabsList>
        <TabsContent value="overview">
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={record.name} />
            <KeyValueRow label="Description" value={record.description} />
            <KeyValueRow label="Members" value={record.member_count} />
            <KeyValueRow label="Created" value={<DateValue value={record.created_at} />} />
            <KeyValueRow label="Updated" value={<DateValue value={record.updated_at} />} />
          </KeyValueSection>
        </TabsContent>
        {canGroups ? (
          <TabsContent value="users">
            <MembershipsTable groupId={id} />
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}
