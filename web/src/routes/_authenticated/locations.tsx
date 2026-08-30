import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/locations")({
  staticData: { breadcrumb: "Locations" },
});
