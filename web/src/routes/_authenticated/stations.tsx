import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/stations")({
  staticData: { breadcrumb: "Stations" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "stations", access: "view" }),
});
