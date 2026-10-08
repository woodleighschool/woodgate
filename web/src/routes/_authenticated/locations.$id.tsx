import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { locationQueryOptions } from "@features/resources/queries";

export const Route = createFileRoute("/_authenticated/locations/$id")({
  staticData: { breadcrumb: resourceName(locationQueryOptions, (location) => location.name) },
});
