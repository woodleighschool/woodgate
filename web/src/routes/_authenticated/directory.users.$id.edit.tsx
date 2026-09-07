import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { UserEditPage } from "@features/directory/users/edit";

export const Route = createFileRoute("/_authenticated/directory/users/$id/edit")({
  staticData: { breadcrumb: "Edit" },
  beforeLoad: ({ context, params }) =>
    requirePermissions(
      context.queryClient,
      [
        { resource: "users", access: "edit" },
        { resource: "authz.roles", access: "edit" },
      ],
      () => {
        throw redirect({
          to: "/directory/users/$id",
          params: { id: params.id },
        });
      },
    ),
  component: UserEditPage,
});
