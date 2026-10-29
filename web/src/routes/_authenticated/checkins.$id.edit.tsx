import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";
import { CheckinEditPage } from "@features/checkins/edit";

export const Route = createFileRoute("/_authenticated/checkins/$id/edit")({
  component: CheckinEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermission(context.queryClient, { resource: "checkins", access: "edit" }, () => {
      throw redirect({ to: "/checkins/$id", params: { id: params.id } });
    }),
});
