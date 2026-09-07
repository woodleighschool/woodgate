import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { LocationCreatePage } from "@features/locations/create";

export const Route = createFileRoute("/_authenticated/locations/new")({
  component: LocationCreatePage,
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "locations", access: "edit" }),
});
