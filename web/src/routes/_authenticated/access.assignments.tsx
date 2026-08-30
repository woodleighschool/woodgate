import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/access/assignments")({
  staticData: { breadcrumb: "Assignments" },
});
