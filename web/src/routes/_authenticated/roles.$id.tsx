import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { authzRoleQueryOptions } from "@features/resources/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/roles/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(authzRoleQueryOptions, (role) => role.name) },
  loader: (ctx) => loadResource(ctx, authzRoleQueryOptions(ctx.params.id)),
});
