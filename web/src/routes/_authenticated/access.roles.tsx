import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/access/roles")({
  staticData: { breadcrumb: "Roles" },
});
