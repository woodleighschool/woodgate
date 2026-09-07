import assert from "node:assert/strict";
import test from "node:test";

import { appKeyGrants, pairingPayload } from "./pairing.ts";

void test("pairing grants scope check-in creation to the selected location and exclude photos", () => {
  assert.deepEqual(appKeyGrants("location-a"), [
    { resource: "locations", action: "read" },
    { resource: "users", action: "read" },
    { resource: "assets", action: "read", asset_type: "asset" },
    { resource: "checkins", action: "create", location_id: "location-a" },
  ]);
});

void test("pairing preserves the companion app's api_key and base_url contract", () => {
  assert.deepEqual(JSON.parse(pairingPayload("synthetic-secret", "https://example.invalid")), {
    api_key: "synthetic-secret",
    base_url: "https://example.invalid",
  });
});
