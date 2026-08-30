import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/access")({
  staticData: { breadcrumb: "Access Control" },
});
