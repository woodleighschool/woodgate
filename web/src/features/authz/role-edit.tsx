import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { RoleForm } from "@features/authz/role-form";
import { useAuthzRole, useUpdateAuthzRole } from "@features/resources/queries";

export function RoleEditPage() {
  const navigate = useNavigate();
  const { id } = useParams({ from: "/_authenticated/roles/$id/edit" });
  const query = useAuthzRole(id);
  const update = useUpdateAuthzRole(id);
  if (!query.data || query.data.builtin) {
    return (
      <QueryGate
        title="Failed to Load Role"
        error={query.data ? { message: "This role cannot be edited." } : query.error}
      />
    );
  }
  return (
    <RoleForm
      title="Edit Role"
      initial={query.data}
      pending={update.isPending}
      onSubmit={(body) => {
        void update
          .mutateAsync(body)
          .then((role) => navigate({ to: "/roles/$id", params: { id: role.id } }));
      }}
      onCancel={() => void navigate({ to: "/roles/$id", params: { id } })}
    />
  );
}
