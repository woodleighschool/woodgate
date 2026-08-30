import { createFileRoute } from "@tanstack/react-router";

import { RoleDetailPage } from "@features/authz/role-detail";

export const Route = createFileRoute("/_authenticated/access/roles/$id/")({
  component: RoleDetailPage,
});
