import { getRouteApi } from "@tanstack/react-router";
import { MonitorSmartphone, Plus } from "lucide-react";

import { BooleanIndicator } from "@components/boolean-indicator";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { useStations } from "@features/resources/queries";
import { STATION_STATES } from "@features/stations/model";
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
    accessorKey: "location.name",
    header: "Location",
    cell: ({ row }) => row.original.location.name,
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
    id: "state",
    accessorKey: "state",
    header: "State",
    cell: ({ row }) => (
      <EnumStatusIndicator value={row.original.state} metadata={STATION_STATES} showIndicator />
    ),
    enableSorting: false,
    meta: { label: "State" },
  },
  {
    id: "version",
    accessorKey: "version",
    header: "Version",
    cell: ({ row }) => row.original.version || <span className="text-muted-foreground">-</span>,
    meta: { label: "Version" },
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
  const canEditStations = useCan({ resource: "stations", access: "edit" });
  const canEdit = canEditStations;
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
