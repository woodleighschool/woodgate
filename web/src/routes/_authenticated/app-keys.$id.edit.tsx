import { createFileRoute } from "@tanstack/react-router";

import { AppKeyEditPage } from "@features/app-keys/pages";
import { requirePermission } from "@features/authn/guards";

export const Route = createFileRoute("/_authenticated/app-keys/$id/edit")({
  component: AppKeyEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context }) =>
    requirePermission(context.queryClient, { resource: "app_keys", access: "edit" }),
});
