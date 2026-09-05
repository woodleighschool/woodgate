import assert from "node:assert/strict";
import { test } from "node:test";

import type { NavItem } from "./nav-config";
import { filterNavigation, firstNavigationTarget } from "./navigation.ts";

void test("navigation prunes inaccessible branches and redirects a group to an allowed descendant", () => {
  const items: NavItem[] = [
    {
      label: "Locations",
      to: "/locations",
      items: [
        {
          label: "Edit",
          to: "/locations/new",
          permission: { resource: "locations", access: "edit" },
        },
        {
          label: "Read",
          to: "/locations/list",
          permission: { resource: "locations", access: "view" },
        },
      ],
    },
    { label: "Hidden", to: "/disabled", disabled: true },
  ];
  const original = structuredClone(items);
  const filtered = filterNavigation(items, { locations: "view" });
  assert.equal(filtered[0]?.label, "Locations");
  assert.equal(filtered[0]?.to, "/locations/list");
  assert.equal(filtered[0]?.items?.length, 1);
  assert.equal(firstNavigationTarget(filtered), "/locations/list");
  assert.deepEqual(filterNavigation(items, undefined), []);
  assert.deepEqual(items, original);
});
