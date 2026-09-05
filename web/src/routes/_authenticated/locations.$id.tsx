import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/locations/$id")({
  staticData: { breadcrumb: "Details" },
});
