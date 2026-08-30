import { createFileRoute } from "@tanstack/react-router";

import { AssignmentListPage } from "@features/authz/assignments";

export const Route = createFileRoute("/_authenticated/access/assignments/")({
  component: AssignmentListPage,
});
