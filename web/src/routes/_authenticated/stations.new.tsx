import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { StationCreatePage } from "@features/stations/create";

export const Route = createFileRoute("/_authenticated/stations/new")({
  component: StationCreatePage,
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "stations", access: "edit" }),
});
