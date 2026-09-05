import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/roles")({
  staticData: { breadcrumb: "Roles" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "authz.roles", access: "view" }, () => {
      throw redirect({ to: "/checkins" });
    }),
});
