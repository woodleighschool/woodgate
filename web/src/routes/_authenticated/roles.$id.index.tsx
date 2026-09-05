import { createFileRoute } from "@tanstack/react-router";

import { RoleDetailPage } from "@features/authz/role-detail";

export const Route = createFileRoute("/_authenticated/roles/$id/")({
  component: RoleDetailPage,
});
