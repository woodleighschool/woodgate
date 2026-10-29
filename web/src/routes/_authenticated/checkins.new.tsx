import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { CheckinCreatePage } from "@features/checkins/create";

export const Route = createFileRoute("/_authenticated/checkins/new")({
  component: CheckinCreatePage,
  staticData: { breadcrumb: "Create Check-in" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "checkins", access: "edit" }),
});
