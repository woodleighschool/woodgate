import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { RoleForm } from "@features/authz/role-form";
import { useAuthzRole, useUpdateAuthzRole } from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";

export function RoleEditPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/roles/$id/edit" });
  const id = parseRouteID(rawID);
  const query = useAuthzRole(id);
  const update = useUpdateAuthzRole(id ?? 0);
  if (id === null || query.error || !query.data || query.data.builtin) {
    return (
      <QueryGate
        title="Failed to Load Role"
        error={query.error ?? { message: "This role cannot be edited." }}
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
          .then((role) => navigate({ to: "/roles/$id", params: { id: String(role.id) } }));
      }}
      onCancel={() => void navigate({ to: "/roles/$id", params: { id: String(id) } })}
    />
  );
}
