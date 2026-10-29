import { useNavigate } from "@tanstack/react-router";

import { RoleForm } from "@features/authz/role-form";
import { useCreateAuthzRole } from "@features/resources/queries";

export function RoleCreatePage() {
  const navigate = useNavigate();
  const create = useCreateAuthzRole();
  return (
    <RoleForm
      title="Create Role"
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => void navigate({ to: "/roles/$id", params: { id } })}
      onCancel={() => void navigate({ to: "/roles" })}
    />
  );
}
