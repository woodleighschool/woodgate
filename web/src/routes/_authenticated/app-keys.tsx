import { createFileRoute } from "@tanstack/react-router";

import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/app-keys")({
  staticData: { breadcrumb: "App Keys" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "app_keys", access: "view" }),
});
