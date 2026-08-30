import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/checkins")({
  staticData: { breadcrumb: "Check-ins" },
});
