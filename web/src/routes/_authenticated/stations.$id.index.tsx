import { createFileRoute } from "@tanstack/react-router";

import { StationDetailPage } from "@features/stations/detail";

export const Route = createFileRoute("/_authenticated/stations/$id/")({
  component: StationDetailPage,
});
