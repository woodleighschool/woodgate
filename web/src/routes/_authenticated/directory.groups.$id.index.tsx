import { createFileRoute } from "@tanstack/react-router";

import { GroupDetailPage } from "@features/directory/groups/detail";

export const Route = createFileRoute("/_authenticated/directory/groups/$id/")({
  component: GroupDetailPage,
});
