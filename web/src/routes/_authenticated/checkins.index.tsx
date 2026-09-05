import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { z } from "zod";

import { CheckinListPage } from "@features/checkins/list";
import { createTableSearchSchema, TABLE_SEARCH_DEFAULTS } from "@lib/table-search";

const searchSchema = createTableSearchSchema(["user", "location", "direction", "created_at"])
  .extend({
    direction: z.enum(["check_in", "check_out"]).optional().catch(undefined),
    from: z.iso.date().optional().catch(undefined),
    to: z.iso.date().optional().catch(undefined),
  })
  .transform((search) => {
    if (search.to && !search.from) return { ...search, to: undefined };
    if (search.from && search.to && search.from > search.to) {
      return { ...search, from: search.to, to: search.from };
    }
    return search;
  });

export const Route = createFileRoute("/_authenticated/checkins/")({
  validateSearch: searchSchema,
  search: { middlewares: [stripSearchParams(TABLE_SEARCH_DEFAULTS)] },
  component: CheckinListPage,
});
