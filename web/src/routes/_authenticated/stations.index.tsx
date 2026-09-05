import { createFileRoute, stripSearchParams } from "@tanstack/react-router";

import { StationListPage } from "@features/stations/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["name", "location.name", "enabled", "version"]);

export const Route = createFileRoute("/_authenticated/stations/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: StationListPage,
});
