import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { StationEditPage } from "@features/stations/edit";

export const Route = createFileRoute("/_authenticated/stations/$id/edit")({
  component: StationEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "stations", access: "edit" }),
});
