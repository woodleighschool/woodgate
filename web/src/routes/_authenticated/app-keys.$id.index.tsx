import { createFileRoute } from "@tanstack/react-router";

import { AppKeyDetailPage } from "@features/app-keys/pages";

export const Route = createFileRoute("/_authenticated/app-keys/$id/")({
  component: AppKeyDetailPage,
});
