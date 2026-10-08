import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { groupQueryOptions } from "@features/directory/groups/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/directory/groups/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(groupQueryOptions, (group) => group.display_name) },
  loader: (ctx) => loadResource(ctx, groupQueryOptions(ctx.params.id)),
});
