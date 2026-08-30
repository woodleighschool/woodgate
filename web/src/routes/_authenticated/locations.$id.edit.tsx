import { createFileRoute } from "@tanstack/react-router";

import { LocationEditPage } from "@features/locations/edit";

export const Route = createFileRoute("/_authenticated/locations/$id/edit")({
  component: LocationEditPage,
  staticData: { breadcrumb: "Edit" },
});
