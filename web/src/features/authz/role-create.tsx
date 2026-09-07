import { useNavigate } from "@tanstack/react-router";

import { RoleForm } from "@features/authz/role-form";
import { useCreateAuthzRole } from "@features/resources/queries";

export function RoleCreatePage() {
  const navigate = useNavigate();
  const create = useCreateAuthzRole();
  return (
    <RoleForm
      title="Create Role"
      pending={create.isPending}
      onSubmit={(body) => {
        void create
          .mutateAsync(body)
          .then((role) => navigate({ to: "/roles/$id", params: { id: String(role.id) } }));
      }}
      onCancel={() => void navigate({ to: "/roles" })}
    />
  );
}
