import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { RoleCreatePage } from "@features/authz/role-create";

export const Route = createFileRoute("/_authenticated/roles/new")({
  component: RoleCreatePage,
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "authz.roles", access: "edit" }, () => {
      throw redirect({ to: "/roles" });
    }),
});
