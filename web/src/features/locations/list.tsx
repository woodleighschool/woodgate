import { getRouteApi } from "@tanstack/react-router";
import { MapPin, Plus } from "lucide-react";

import { BooleanIndicator } from "@components/boolean-indicator";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { useAccount } from "@features/account/queries";
import { canAccess } from "@features/authz/permissions";
import { useLocations } from "@features/resources/queries";
import type { Location } from "@lib/api";
import { nonEmpty } from "@lib/utils";

const routeApi = getRouteApi("/_authenticated/locations/");

const columns: DataTableColumnDef<Location>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink
        to="/locations/$id"
        params={{ id: String(row.original.id) }}
        className="font-medium"
      >
        {row.original.name}
      </TextLink>
    ),
    meta: { label: "Name" },
  },
  {
    id: "description",
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => nonEmpty(row.original.description) ?? "-",
    meta: { label: "Description" },
  },
  {
    id: "enabled",
    accessorKey: "enabled",
    header: "Enabled",
    cell: ({ row }) => <BooleanIndicator value={row.original.enabled ?? true} />,
    meta: { label: "Enabled" },
  },
];

export function LocationListPage() {
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const query = useLocations(tableSearch);
  const account = useAccount();
  const canEdit = canAccess(account.data, "locations", "edit");
  return (
    <PageShell>
      <PageHeader
        title="Locations"
        description="Manage the places where people check in."
        actions={
          canEdit ? (
            <Button size="sm" render={<Link to="/locations/new" />} nativeButton={false}>
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
        icon={<MapPin />}
        emptyTitle="Locations"
        emptyDescription="Locations appear here once they are configured."
      />
    </PageShell>
  );
}
