import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { appKeyQueryOptions } from "@features/app-keys/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/app-keys/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(appKeyQueryOptions, (key) => key.name) },
  loader: (ctx) => loadResource(ctx, appKeyQueryOptions(ctx.params.id)),
});
