import { createFileRoute } from "@tanstack/react-router";

import { LocationCreatePage } from "@features/locations/create";

export const Route = createFileRoute("/_authenticated/locations/new")({
  component: LocationCreatePage,
  staticData: { breadcrumb: "Create" },
});
