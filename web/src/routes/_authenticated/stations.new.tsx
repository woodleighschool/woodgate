import { createFileRoute } from "@tanstack/react-router";

import { StationCreatePage } from "@features/stations/create";

export const Route = createFileRoute("/_authenticated/stations/new")({
  component: StationCreatePage,
  staticData: { breadcrumb: "Create" },
});
