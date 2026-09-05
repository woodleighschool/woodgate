import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { stationQueryOptions } from "@features/resources/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/stations/$id")({
  params: idParams,
  staticData: { breadcrumb: resourceName(stationQueryOptions, (station) => station.name) },
  loader: (ctx) => loadResource(ctx, stationQueryOptions(ctx.params.id)),
});
