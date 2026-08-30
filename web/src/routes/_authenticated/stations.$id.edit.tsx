import { createFileRoute } from "@tanstack/react-router";

import { StationEditPage } from "@features/stations/edit";

export const Route = createFileRoute("/_authenticated/stations/$id/edit")({
  component: StationEditPage,
  staticData: { breadcrumb: "Edit" },
});
