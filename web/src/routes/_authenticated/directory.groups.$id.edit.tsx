import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { GroupEditPage } from "@features/directory/groups/edit";

export const Route = createFileRoute("/_authenticated/directory/groups/$id/edit")({
  component: GroupEditPage,
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermissions(
      context.queryClient,
      [
        { resource: "groups", access: "edit" },
        { resource: "authz.roles", access: "edit" },
      ],
      () => {
        throw redirect({
          to: "/directory/groups/$id",
          params: { id: params.id },
        });
      },
    ),
});
