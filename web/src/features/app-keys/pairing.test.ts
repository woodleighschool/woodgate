import assert from "node:assert/strict";
import { test } from "node:test";

import { pairingPayload } from "./pairing.ts";

void test("pairing retains the current native API key payload", () => {
  assert.deepEqual(JSON.parse(pairingPayload("https://app.example.invalid/", "reviewkey")), {
    base_url: "https://app.example.invalid",
    api_key: "reviewkey",
  });
});
