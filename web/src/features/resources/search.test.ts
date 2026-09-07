import assert from "node:assert/strict";
import { test } from "node:test";

import { parseResourceSearch } from "./search.ts";

void test("existing list bookmarks preserve UUID scopes, search, paging, and sort", () => {
  const search = parseResourceSearch(
    {
      page: "3",
      perPage: "25",
      sort: "created_at",
      order: "DESC",
      filter: JSON.stringify({
        search: "Example",
        user_id: "00000000-0000-0000-0000-000000000001",
        enabled: false,
      }),
    },
    "name.asc",
  );
  assert.equal(search.page, 3);
  assert.equal(search.per_page, 25);
  assert.equal(search.sort, "created_at.desc");
  assert.equal(search.q, "Example");
  assert.equal(search.enabled, false);
  assert.equal(search.user_id, "00000000-0000-0000-0000-000000000001");
});
void test("malformed bookmarks use supported list defaults", () => {
  const search = parseResourceSearch(
    { page: -2, perPage: "invalid", filter: "{" },
    "created_at.desc",
  );
  assert.equal(search.page, 1);
  assert.equal(search.per_page, 10);
  assert.equal(search.sort, "created_at.desc");
  assert.equal(search.q, undefined);
});
void test("new table state takes precedence over old bookmark fields", () => {
  const search = parseResourceSearch(
    {
      q: "new",
      per_page: 50,
      perPage: 25,
      sort: "name.asc",
      order: "DESC",
      filter: { search: "old" },
    },
    "created_at.desc",
  );
  assert.equal(search.q, "new");
  assert.equal(search.per_page, 50);
  assert.equal(search.sort, "name.asc");
});
