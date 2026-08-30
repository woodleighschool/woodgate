import { createFileRoute } from "@tanstack/react-router";

import { CheckinDetailPage } from "@features/checkins/detail";

export const Route = createFileRoute("/_authenticated/checkins/$id")({
  staticData: { breadcrumb: "Details" },
  component: CheckinDetailPage,
});
