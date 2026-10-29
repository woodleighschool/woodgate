import { createFileRoute } from "@tanstack/react-router";

import { checkinQueryOptions } from "@features/resources/queries";
import { loadResource } from "@lib/resource-loader";
import { idParams } from "@lib/route-params";

export const Route = createFileRoute("/_authenticated/checkins/$id")({
  params: idParams,
  staticData: { breadcrumb: "Check-in" },
  loader: (ctx) => loadResource(ctx, checkinQueryOptions(ctx.params.id)),
});
