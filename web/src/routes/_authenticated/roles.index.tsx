import { createFileRoute } from "@tanstack/react-router";

import { RoleListPage } from "@features/authz/roles";

export const Route = createFileRoute("/_authenticated/roles/")({
  component: RoleListPage,
});
