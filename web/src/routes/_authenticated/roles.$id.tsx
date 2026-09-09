import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/roles/$id")({
  staticData: { breadcrumb: "Details" },
});
