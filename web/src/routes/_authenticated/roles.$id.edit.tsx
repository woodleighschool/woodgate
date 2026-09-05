import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { RoleEditPage } from "@features/authz/role-edit";

export const Route = createFileRoute("/_authenticated/roles/$id/edit")({
  component: RoleEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "authz.roles", access: "edit" }, () => {
      throw redirect({ to: "/roles/$id", params: { id: params.id } });
    }),
});
