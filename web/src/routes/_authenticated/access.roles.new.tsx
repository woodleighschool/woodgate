import { createFileRoute } from "@tanstack/react-router";

import { RoleCreatePage } from "@features/authz/role-create";

export const Route = createFileRoute("/_authenticated/access/roles/new")({
  component: RoleCreatePage,
  staticData: { breadcrumb: "Create" },
});
