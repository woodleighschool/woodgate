import { createFileRoute } from "@tanstack/react-router";

import { AppKeyCreatePage } from "@features/app-keys/pages";
import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/app-keys/new")({
  component: AppKeyCreatePage,
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "app_keys", access: "edit" }),
});
