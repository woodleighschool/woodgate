import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { authzRoleQueryOptions } from "@features/resources/queries";

export const Route = createFileRoute("/_authenticated/roles/$id")({
  staticData: { breadcrumb: resourceName(authzRoleQueryOptions, (role) => role.name) },
});
