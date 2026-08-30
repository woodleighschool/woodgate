import { getRouteApi } from "@tanstack/react-router";
import { MonitorSmartphone, Plus } from "lucide-react";

import { BooleanIndicator } from "@components/boolean-indicator";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { useAccount } from "@features/account/queries";
import { canAccess } from "@features/authz/permissions";
import { useStations } from "@features/resources/queries";
import type { Station } from "@lib/api";

const routeApi = getRouteApi("/_authenticated/stations/");

const columns: DataTableColumnDef<Station>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Station",
    cell: ({ row }) => (
      <TextLink to="/stations/$id" params={{ id: String(row.original.id) }} className="font-medium">
        {row.original.name}
      </TextLink>
    ),
    meta: { label: "Station" },
  },
  {
    id: "location",
    accessorKey: "location_id",
    header: "Location",
    cell: ({ row }) => `Location #${row.original.location_id}`,
    meta: { label: "Location" },
  },
  {
    id: "enabled",
    accessorKey: "enabled",
    header: "Enabled",
    cell: ({ row }) => <BooleanIndicator value={row.original.enabled ?? true} />,
    meta: { label: "Enabled" },
  },
  {
    id: "last_seen_at",
    accessorKey: "last_seen_at",
    header: "Last Seen",
    cell: ({ row }) =>
      row.original.last_seen_at ? new Date(row.original.last_seen_at).toLocaleString() : "Never",
    meta: { label: "Last Seen" },
  },
];

export function StationListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const query = useStations(tableSearch);
  const account = useAccount();
  const canEdit = canAccess(account.data, "stations", "edit");
  return (
    <PageShell>
      <PageHeader
        title="Stations"
        description="Manage the dedicated devices used at check-in locations."
        actions={
          canEdit ? (
            <Button size="sm" render={<Link to="/stations/new" />} nativeButton={false}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
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
        icon={<MonitorSmartphone />}
        emptyTitle="Stations"
        emptyDescription="Stations appear here once they are provisioned."
      />
    </PageShell>
  );
}
