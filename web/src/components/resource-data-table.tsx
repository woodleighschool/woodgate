import type { ReactNode } from "react";

import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef, DataTableRowData } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import type { DataTableQuery } from "@components/data-table/use-data-table-search";
import { QueryError } from "@components/query-error";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

export function ResourceDataTable<T extends DataTableRowData>({
  data,
  count,
  columns,
  tableSearch,
  loading,
  pending,
  error,
  onRetry,
  icon,
  emptyTitle,
  emptyDescription,
  filters,
  exportOptions,
  onRowClick,
}: {
  data: T[];
  count: number;
  columns: DataTableColumnDef<T>[];
  tableSearch: DataTableQuery;
  loading: boolean;
  pending: boolean;
  error: { message?: string } | null;
  onRetry: () => void;
  icon: ReactNode;
  emptyTitle: string;
  emptyDescription: string;
  filters?: ReactNode;
  exportOptions?: DataTableExportOptions<T>;
  onRowClick?: (row: T) => void;
}) {
  const pageCount = loading ? -1 : Math.ceil(count / tableSearch.per_page);
  const table = useDataTable({
    tableState: tableSearch,
    data,
    columns,
    pageCount,
    rowCount: count,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
  });

  if (error) {
    return <QueryError title={`Failed to Load ${emptyTitle}`} error={error} onRetry={onRetry} />;
  }
  if (loading)
    return <DataTableSkeleton columnCount={columns.length} withExport={!!exportOptions} />;

  return (
    <DataTable
      table={table}
      pending={pending}
      exportOptions={exportOptions}
      onRowClick={onRowClick}
      empty={
        <DataTableEmpty
          icon={icon}
          filtered={tableSearch.isFiltered}
          title={`No ${emptyTitle}`}
          description={emptyDescription}
          filteredDescription="No records matched the current search."
        />
      }
    >
      <DataTableSearchInput
        loading={pending}
        value={tableSearch.q ?? ""}
        onValueChange={tableSearch.onQueryChange}
      />
      {filters}
    </DataTable>
  );
}
