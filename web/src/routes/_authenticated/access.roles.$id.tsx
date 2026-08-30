import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/access/roles/$id")({
  staticData: { breadcrumb: "Details" },
});
