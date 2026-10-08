import { createFileRoute } from "@tanstack/react-router";

import { resourceName } from "@components/layout/app-breadcrumbs";
import { groupQueryOptions } from "@features/directory/groups/queries";

export const Route = createFileRoute("/_authenticated/directory/groups/$id")({
  staticData: { breadcrumb: resourceName(groupQueryOptions, (group) => group.display_name) },
});
