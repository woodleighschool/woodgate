import { useNavigate, useParams } from "@tanstack/react-router";
import type { Access } from "@woodleighschool/authz";
import { permissionLevel } from "@woodleighschool/authz";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { PermissionLevelBadge, permissionLabel } from "@features/authz/permission-level";
import { useAuthzResources, useAuthzRole, useDeleteAuthzRole } from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";

interface PermissionRow {
  name: string;
  label: string;
  level: Access;
}

const columns: DataTableColumnDef<PermissionRow>[] = [
  { id: "label", accessorKey: "label", header: "Resource", meta: { label: "Resource" } },
  {
    id: "level",
    accessorKey: "level",
    header: "Access",
    cell: ({ row }) => <PermissionLevelBadge level={row.original.level} />,
    meta: { label: "Access" },
  },
];

export function RoleDetailPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/roles/$id" });
  const id = parseRouteID(rawID);
  const role = useAuthzRole(id);
  const resources = useAuthzResources();
  const canEditRoles = useCan({ resource: "authz.roles", access: "edit" });
  const remove = useDeleteAuthzRole();
  const [confirmOpen, setConfirmOpen] = useState(false);
  if (id === null)
    return <QueryGate title="Failed to Load Role" error={{ message: "Invalid role." }} />;
  if (role.error || !role.data) {
    return (
      <QueryGate
        title="Failed to Load Role"
        error={role.error}
        onRetry={() => void role.refetch()}
      />
    );
  }

  const names =
    resources.data?.map((resource) => resource.resource) ?? Object.keys(role.data.permissions);
  const rows = names.map((name) => ({
    name,
    label:
      resources.data?.find((resource) => resource.resource === name)?.display_name ??
      permissionLabel(name),
    level: permissionLevel(role.data?.permissions[name]),
  }));

  return (
    <PageShell>
      <PageHeader
        title={role.data.name}
        description={role.data.description}
        actions={
          !role.data.builtin && canEditRoles ? (
            <>
              <Button
                size="sm"
                variant="outline"
                render={<Link to="/roles/$id/edit" params={{ id: String(role.data.id) }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button size="sm" variant="destructive" onClick={() => setConfirmOpen(true)}>
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />
      <DataTableStatic columns={columns} data={rows} heading="Resource Access" />
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Delete Role?"
        description="The role will be removed from every user and group."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => {
          void remove.mutateAsync(role.data.id).then(() => navigate({ to: "/roles" }));
        }}
      />
    </PageShell>
  );
}
