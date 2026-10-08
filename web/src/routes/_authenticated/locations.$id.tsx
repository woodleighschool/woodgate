import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { locationQueryOptions } from "@features/resources/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/locations/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(locationQueryOptions, (location) => location.name) },
  loader: (ctx) => loadResource(ctx, locationQueryOptions(ctx.params.id)),
});
