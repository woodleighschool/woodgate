import { createFileRoute, redirect } from "@tanstack/react-router";

import { requirePermissions } from "@features/authn/guards";
import { UserCreatePage } from "@features/directory/users/create";

export const Route = createFileRoute("/_authenticated/directory/users/new")({
  staticData: { breadcrumb: "Create" },
  beforeLoad: ({ context }) =>
    requirePermissions(
      context.queryClient,
      [
        { resource: "users", access: "edit" },
        { resource: "authz.roles", access: "edit" },
      ],
      () => {
        throw redirect({ to: "/directory/users" });
      },
    ),
  component: UserCreatePage,
});
