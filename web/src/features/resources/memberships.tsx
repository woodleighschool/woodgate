import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Users } from "lucide-react";

import type { DataTableColumnDef } from "@components/data-table/types";
import { ResourceDataTable } from "@components/resource-data-table";
import { ReferenceName } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import { listGroupMemberships, unwrap, type GroupMembership } from "@lib/api";

const groupColumns: DataTableColumnDef<GroupMembership>[] = [
  {
    id: "group",
    header: "Group",
    enableSorting: false,
    cell: ({ row }) => <ReferenceName resource="groups" id={row.original.group_id} />,
  },
];
const userColumns: DataTableColumnDef<GroupMembership>[] = [
  {
    id: "user",
    header: "User",
    enableSorting: false,
    cell: ({ row }) => <ReferenceName resource="users" id={row.original.user_id} />,
  },
  {
    id: "department",
    header: "Department",
    enableSorting: false,
    cell: ({ row }) => (
      <ReferenceName resource="users" id={row.original.user_id} field="department" />
    ),
  },
];
export function MembershipsTable({ userId, groupId }: { userId?: string; groupId?: string }) {
  const { tableSearch, params } = useResourceTable("created_at.asc");
  const filters = { ...params, user_id: userId, group_id: groupId };
  const query = useQuery({
    queryKey: ["group-memberships", filters],
    queryFn: ({ signal }) => unwrap(listGroupMemberships({ query: filters, signal })),
    placeholderData: keepPreviousData,
  });
  return (
    <ResourceDataTable<GroupMembership>
      data={query.data?.rows ?? []}
      count={query.data?.total ?? 0}
      columns={userId ? groupColumns : userColumns}
      tableSearch={tableSearch}
      loading={query.isPending}
      pending={query.isFetching}
      error={query.error}
      onRetry={() => void query.refetch()}
      icon={<Users />}
      emptyTitle={userId ? "Groups" : "Users"}
      emptyDescription="There are no group memberships."
    />
  );
}
