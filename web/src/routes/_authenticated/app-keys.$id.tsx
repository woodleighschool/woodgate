import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { appKeyQueryOptions } from "@features/app-keys/queries";

export const Route = createFileRoute("/_authenticated/app-keys/$id")({
  staticData: { breadcrumb: resourceName(appKeyQueryOptions, (key) => key.name) },
});
