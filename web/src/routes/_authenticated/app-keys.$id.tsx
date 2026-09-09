import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/app-keys/$id")({
  staticData: { breadcrumb: "Details" },
});
