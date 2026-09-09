import assert from "node:assert/strict";
import { test } from "node:test";

import { createSessionTransport } from "./session.ts";

void test("authentication failure expires both queries and mutations while preserving responses", async () => {
  let expired = 0;
  const session = createSessionTransport(async () => new Response(null, { status: 401 }));
  session.onExpired(() => {
    expired++;
  });
  const read = await session.fetch("/api/users");
  assert.equal(read.status, 401);
  assert.equal(expired, 1);
  session.renew();
  const write = await session.fetch("/api/locations/example", { method: "PATCH" });
  assert.equal(write.status, 401);
  assert.equal(expired, 2);
});
void test("permission and validation failures preserve the current session", async () => {
  let expired = false;
  const session = createSessionTransport(async () => new Response(null, { status: 403 }));
  session.onExpired(() => {
    expired = true;
  });
  assert.equal((await session.fetch("/api/users")).status, 403);
  assert.equal(expired, false);
});
void test("responses from an earlier session cannot expire a new sign-in", async () => {
  let respond: ((response: Response) => void) | undefined;
  let expired = false;
  const session = createSessionTransport(
    () =>
      new Promise<Response>((resolve) => {
        respond = resolve;
      }),
  );
  session.onExpired(() => {
    expired = true;
  });
  const pending = session.fetch("/api/users");
  session.renew();
  assert.ok(respond);
  respond(new Response(null, { status: 401 }));
  await pending;
  assert.equal(expired, false);
});
void test("concurrent failures expire a session only once", async () => {
  let expired = 0;
  const session = createSessionTransport(async () => new Response(null, { status: 401 }));
  session.onExpired(() => {
    expired++;
  });
  await Promise.all([session.fetch("/auth/user"), session.fetch("/api/users")]);
  assert.equal(expired, 1);
});

void test("failed login preserves its inline response without expiring a session", async () => {
  let expired = false;
  const session = createSessionTransport(async () => new Response(null, { status: 401 }));
  session.onExpired(() => {
    expired = true;
  });
  assert.equal(
    (await session.fetch("https://app.example.invalid/api/session", { method: "POST" })).status,
    401,
  );
  assert.equal(expired, false);
});
