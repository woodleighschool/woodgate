import { useState, useMemo, type ReactNode } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTable } from "@components/data-table/data-table";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableColumnDef, DataTableRowData } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import type { DataTableQuery } from "@components/data-table/use-data-table-search";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Checkbox } from "@components/ui/checkbox";
import { toast } from "@components/ui/toast";
import { DEFAULT_PAGE_SIZE } from "@lib/pagination";

function selectionColumn<T extends DataTableRowData>(): DataTableColumnDef<T> {
  return {
    id: "select",
    size: 40,
    enableSorting: false,
    enableResizing: false,
    header: ({ table }) => (
      <Checkbox
        aria-label="Select all on this page"
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(checked) => table.toggleAllPageRowsSelected(checked)}
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        aria-label="Select record"
        checked={row.getIsSelected()}
        onCheckedChange={(checked) => row.toggleSelected(checked)}
      />
    ),
  };
}

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
  onDeleteRows,
  getRowId,
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
  onDeleteRows?: (rows: T[]) => Promise<void>;
  getRowId?: (row: T) => string;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const selection = useMemo(() => selectionColumn<T>(), []);
  const pageCount = loading ? -1 : Math.ceil(count / tableSearch.per_page);
  const table = useDataTable({
    tableState: tableSearch,
    data,
    columns: onDeleteRows ? [selection, ...columns] : columns,
    enableRowSelection: Boolean(onDeleteRows),
    getRowId,
    pageCount,
    rowCount: count,
    initialState: { pagination: { pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE } },
  });

  if (error) {
    return <QueryError title={`Failed to Load ${emptyTitle}`} error={error} onRetry={onRetry} />;
  }
  if (loading) return <DataTableSkeleton columnCount={columns.length} />;

  const selected = table.getSelectedRowModel().rows.map((row) => row.original);
  return (
    <>
      <DataTable
        table={table}
        pending={pending}
        exportOptions={exportOptions}
        toolbarActions={
          onDeleteRows && selected.length > 0 ? (
            <Button variant="destructive" size="sm" onClick={() => setConfirmDelete(true)}>
              Delete {selected.length} selected
            </Button>
          ) : undefined
        }
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
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title={`Delete ${selected.length} records?`}
        description="This cannot be undone."
        confirmLabel="Delete"
        variant="destructive"
        pending={deleting}
        onConfirm={() => {
          if (!onDeleteRows) return;
          setDeleting(true);
          void onDeleteRows(selected)
            .then(() => {
              table.resetRowSelection();
              setConfirmDelete(false);
              return undefined;
            })
            .catch((cause: unknown) =>
              toast.add({
                type: "error",
                title: "Failed to delete",
                description: cause instanceof Error ? cause.message : undefined,
              }),
            )
            .finally(() => setDeleting(false));
        }}
      />
    </>
  );
}
