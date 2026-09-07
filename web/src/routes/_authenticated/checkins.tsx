import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/checkins")({
  staticData: { breadcrumb: "Check-ins" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "checkins", access: "view" }),
});
