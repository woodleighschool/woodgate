import { createFileRoute } from "@tanstack/react-router";

import { RoleEditPage } from "@features/authz/role-edit";

export const Route = createFileRoute("/_authenticated/access/roles/$id/edit")({
  component: RoleEditPage,
  staticData: { breadcrumb: "Edit" },
});
