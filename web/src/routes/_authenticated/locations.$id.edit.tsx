import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { LocationEditPage } from "@features/locations/edit";

export const Route = createFileRoute("/_authenticated/locations/$id/edit")({
  component: LocationEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "locations", access: "edit" }),
});
