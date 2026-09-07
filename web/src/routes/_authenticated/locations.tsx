import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/locations")({
  staticData: { breadcrumb: "Locations" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "locations", access: "view" }),
});
