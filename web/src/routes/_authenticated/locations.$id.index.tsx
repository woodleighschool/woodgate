import { createFileRoute } from "@tanstack/react-router";

import { LocationDetailPage } from "@features/locations/detail";

export const Route = createFileRoute("/_authenticated/locations/$id/")({
  component: LocationDetailPage,
});
