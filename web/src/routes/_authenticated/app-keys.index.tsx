import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { AppKeyListPage } from "@features/app-keys/pages";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "created_at", "last_used_at", "expires_at"]);

export const Route = createFileRoute("/_authenticated/app-keys/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: AppKeyListPage,
});
