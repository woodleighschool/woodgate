import { Plus, ShieldCheck } from "lucide-react";

import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { useAuthzRoles } from "@features/resources/queries";
import type { AuthzRole } from "@lib/api";
import { nonEmpty } from "@lib/utils";

const columns: DataTableColumnDef<AuthzRole>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Role",
    cell: ({ row }) => (
      <TextLink to="/roles/$id" params={{ id: String(row.original.id) }} className="font-medium">
        {row.original.name}
      </TextLink>
    ),
    meta: { label: "Role" },
  },
  {
    id: "description",
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => nonEmpty(row.original.description) ?? "-",
    meta: { label: "Description" },
  },
  {
    id: "key",
    accessorKey: "key",
    header: "Key",
    cell: ({ row }) => <span className="font-mono text-xs">{row.original.key}</span>,
    meta: { label: "Key" },
  },
  {
    id: "builtin",
    accessorKey: "builtin",
    header: "Type",
    cell: ({ row }) => (row.original.builtin ? "Built-in" : "Custom"),
    meta: { label: "Type" },
  },
];

export function RoleListPage() {
  const query = useAuthzRoles();
  const canEditRoles = useCan({ resource: "authz.roles", access: "edit" });
  const canEdit = canEditRoles;
  return (
    <PageShell>
      <PageHeader
        title="Roles"
        description="Define type-wide access with none, view, or edit for each resource."
        actions={
          canEdit ? (
            <Button size="sm" render={<Link to="/roles/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      {query.error ? (
        <QueryError
          title="Failed to Load Roles"
          error={query.error}
          onRetry={() => void query.refetch()}
        />
      ) : query.isLoading ? (
        <DataTableSkeleton columnCount={columns.length} />
      ) : (
        <DataTableStatic
          data={query.data ?? []}
          columns={columns}
          empty={
            <DataTableEmpty
              icon={<ShieldCheck />}
              title="No Roles"
              description="Roles appear here once access control is configured."
              filteredDescription=""
            />
          }
        />
      )}
    </PageShell>
  );
}
