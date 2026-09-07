import assert from "node:assert/strict";
import test from "node:test";

import type { PermissionGrant } from "../../lib/api-client/types.gen.ts";
import { accessSignature, hasGrant, locationsForAccess, toggleGrant } from "./grants.ts";

void test("changing a global grant preserves narrower grants and independent actions", () => {
  const scoped: PermissionGrant[] = [
    { resource: "checkins", action: "create", location_id: "location-a" },
    { resource: "checkins", action: "read", location_id: "location-b" },
    { resource: "assets", action: "read", asset_type: "photo" },
    { resource: "assets", action: "create", asset_type: "asset" },
  ];
  const global: PermissionGrant = { resource: "checkins", action: "create" };
  const enabled = toggleGrant(scoped, global, true);
  assert.equal(hasGrant(enabled, global), true);
  for (const grant of scoped) assert.equal(hasGrant(enabled, grant), true);
  const disabled = toggleGrant(enabled, global, false);
  assert.equal(hasGrant(disabled, global), false);
  assert.equal(accessSignature(false, disabled), accessSignature(false, scoped));
});

void test("unreturned locations stay available by ID without losing existing access", () => {
  const grants: PermissionGrant[] = [
    { resource: "checkins", action: "create", location_id: "hidden-location" },
    { resource: "checkins", action: "read", location_id: "hidden-location" },
    { resource: "checkins", action: "read", location_id: "visible-location" },
  ];
  assert.deepEqual(locationsForAccess(grants, [{ id: "visible-location", name: "Reception" }]), [
    { id: "hidden-location", name: "hidden-location" },
    { id: "visible-location", name: "Reception" },
  ]);
});

void test("nullable scopes match global grants and repeated toggles do not duplicate them", () => {
  const grants: PermissionGrant[] = [
    { resource: "users", action: "read", location_id: null, asset_type: null },
  ];
  const target: PermissionGrant = { resource: "users", action: "read" };
  assert.equal(hasGrant(grants, target), true);
  assert.deepEqual(toggleGrant(grants, target, false), []);
  assert.deepEqual(toggleGrant(toggleGrant(grants, target, true), target, true), [target]);
  assert.notEqual(accessSignature(true, grants), accessSignature(false, grants));
});
