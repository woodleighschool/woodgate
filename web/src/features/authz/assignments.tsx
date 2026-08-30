import { UserRoundCog } from "lucide-react";
import { useState } from "react";

import { AsyncButton } from "@components/async-button";
import { DataTableEmpty } from "@components/data-table/data-table-empty";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Badge } from "@components/ui/badge";
import { Checkbox } from "@components/ui/checkbox";
import { Field, FieldLabel } from "@components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { useAccount } from "@features/account/queries";
import { canAccess } from "@features/authz/permissions";
import { GroupCombobox } from "@features/directory/groups/group-picker";
import { UserCombobox } from "@features/directory/users/user-combobox";
import {
  useAuthzAssignments,
  useAuthzRoles,
  useReplaceAuthzAssignments,
} from "@features/resources/queries";
import type { AuthzAssignment } from "@lib/api";

const columns: DataTableColumnDef<AuthzAssignment>[] = [
  {
    id: "subject_id",
    accessorKey: "subject_id",
    header: "Subject",
    cell: ({ row }) => `${row.original.subject_kind} #${row.original.subject_id}`,
    meta: { label: "Subject" },
  },
  {
    id: "subject_kind",
    accessorKey: "subject_kind",
    header: "Type",
    cell: ({ row }) => (
      <Badge variant="outline">{row.original.subject_kind === "user" ? "User" : "Group"}</Badge>
    ),
    meta: { label: "Type" },
  },
  {
    id: "role_id",
    accessorKey: "role_id",
    header: "Role",
    cell: ({ row }) => `Role #${row.original.role_id}`,
    meta: { label: "Role" },
  },
];

export function AssignmentListPage() {
  const query = useAuthzAssignments();
  const account = useAccount();
  const roles = useAuthzRoles();
  const replace = useReplaceAuthzAssignments();
  const [kind, setKind] = useState<"user" | "group">("user");
  const [subjectID, setSubjectID] = useState("");
  const [roleIDs, setRoleIDs] = useState<number[]>([]);
  const canEdit = canAccess(account.data, "authz.assignments", "edit");

  const chooseSubject = (value: string) => {
    setSubjectID(value);
    const id = Number(value);
    setRoleIDs(
      (query.data ?? [])
        .filter((assignment) => assignment.subject_kind === kind && assignment.subject_id === id)
        .map((assignment) => assignment.role_id),
    );
  };
  return (
    <PageShell>
      <PageHeader
        title="Role Assignments"
        description="Bind type-wide roles to users and directory groups."
      />
      {canEdit ? (
        <section className="grid max-w-3xl gap-4 rounded-xl border p-4">
          <h2 className="font-semibold">Edit Assignments</h2>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="assignment-kind">Subject type</FieldLabel>
              <Select
                value={kind}
                onValueChange={(value) => {
                  setKind(subjectKind(value ?? "user"));
                  setSubjectID("");
                  setRoleIDs([]);
                }}
              >
                <SelectTrigger id="assignment-kind" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="user">User</SelectItem>
                    <SelectItem value="group">Group</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel htmlFor="assignment-subject">Subject</FieldLabel>
              {kind === "user" ? (
                <UserCombobox id="assignment-subject" value={subjectID} onChange={chooseSubject} />
              ) : (
                <GroupCombobox id="assignment-subject" value={subjectID} onChange={chooseSubject} />
              )}
            </Field>
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            {(roles.data ?? []).map((role) => (
              <FieldLabel key={role.id}>
                <Checkbox
                  checked={roleIDs.includes(role.id)}
                  onCheckedChange={(checked) =>
                    setRoleIDs((current) =>
                      checked
                        ? [...current, role.id]
                        : current.filter((roleID) => roleID !== role.id),
                    )
                  }
                />
                {role.name}
              </FieldLabel>
            ))}
          </div>
          <AsyncButton
            className="w-fit"
            size="sm"
            isPending={replace.isPending}
            disabled={!subjectID}
            onClick={() => {
              void replace.mutateAsync({
                subject_kind: kind,
                subject_id: Number(subjectID),
                role_ids: roleIDs,
              });
            }}
          >
            Save Assignments
          </AsyncButton>
        </section>
      ) : null}
      {query.error ? (
        <QueryError
          title="Failed to Load Assignments"
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
              icon={<UserRoundCog />}
              title="No Assignments"
              description="Role assignments appear here."
              filteredDescription=""
            />
          }
        />
      )}
    </PageShell>
  );
}

function subjectKind(value: string): "user" | "group" {
  return value === "group" ? "group" : "user";
}
