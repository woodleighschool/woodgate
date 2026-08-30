import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/access/")({
  beforeLoad: () => {
    throw redirect({ to: "/access/roles" });
  },
});
