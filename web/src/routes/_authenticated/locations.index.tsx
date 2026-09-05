import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { LocationListPage } from "@features/locations/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "description", "enabled"]);

export const Route = createFileRoute("/_authenticated/locations/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: LocationListPage,
});
