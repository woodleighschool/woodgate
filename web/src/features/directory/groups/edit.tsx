import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useNavigate, useParams } from "@tanstack/react-router";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryGate } from "@components/query-gate";
import { FieldGroup } from "@components/ui/field";
import { RolePicker } from "@features/authz/role-picker";
import { useGroup, useUpdateGroup } from "@features/directory/groups/queries";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import { parseRouteID } from "@lib/route-params";

export function GroupEditPage() {
  const { id: rawID } = useParams({ from: "/_authenticated/directory/groups/$id/edit" });
  const id = parseRouteID(rawID);
  const query = useGroup(id);
  if (id === null)
    return <QueryGate title="Failed to Load Group" error={{ message: "Invalid group." }} />;
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Group"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  return (
    <GroupRoleForm
      groupID={id}
      name={query.data.display_name}
      roleIDs={query.data.roles.map((role) => role.id)}
    />
  );
}

function GroupRoleForm({
  groupID,
  name,
  roleIDs,
}: {
  groupID: number;
  name: string;
  roleIDs: number[];
}) {
  const navigate = useNavigate();
  const update = useUpdateGroup(groupID);
  const cancel = () =>
    void navigate({ to: "/directory/groups/$id", params: { id: String(groupID) } });
  const form = useForm({
    defaultValues: { role_ids: roleIDs },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    onSubmit: async ({ value, formApi }) => {
      await update.mutateAsync(value);
      formApi.reset(value);
      cancel();
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: cancel });
  return (
    <>
      <PageShell>
        <PageHeader title={`Edit ${name}`} />
        <FieldGroup className="max-w-3xl">
          <form.Field name="role_ids">
            {(field) => (
              <RolePicker
                value={field.state.value}
                onChange={field.handleChange}
                description="Members inherit these roles. A group with no role grants no access."
              />
            )}
          </form.Field>
        </FieldGroup>
        <FormActions form={form} submitLabel="Save" onCancel={exitGuard.requestDiscard} />
      </PageShell>
      {exitGuard.dialog}
    </>
  );
}
