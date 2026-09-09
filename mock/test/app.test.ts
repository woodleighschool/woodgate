import { exports } from "cloudflare:workers";
import { describe, expect, it } from "vitest";

const origin = "https://mock.example.invalid";
const reviewRoom = "00000000-0000-0000-0000-000000000101";
const secondLocation = "00000000-0000-0000-0000-000000000102";
const person = "00000000-0000-0000-0000-000000000201";
const uuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

describe("current companion contract", () => {
  it("requires the current API key separately from Station authentication", async () => {
    const response = await exports.default.fetch(`${origin}/auth/me`, {
      headers: { Authorization: "Bearer testing123" },
    });
    expect(response.status).toBe(401);
    await expect(response.json()).resolves.toMatchObject({ status: 401, code: "unauthorized" });
  });
  it("provides explicit UUID grants required by native pairing", async () => {
    const response = await appFetch("/auth/me");
    const auth = await response.json<{
      principal: { type: string; id: string };
      admin: boolean;
      access: { resource: string; action: string; location_id?: string }[];
    }>();
    expect(auth.principal.type).toBe("api_key");
    expect(auth.principal.id).toMatch(uuid);
    expect(auth.admin).toBe(false);
    const grants = auth.access.filter(
      (grant) => grant.resource === "checkins" && grant.action === "create",
    );
    expect(grants.map((grant) => grant.location_id)).toEqual([reviewRoom, secondLocation]);
    const location = await appFetch(`/api/v1/locations/${reviewRoom}`);
    await expect(location.json()).resolves.toEqual({
      id: reviewRoom,
      name: "Review Room",
      enabled: true,
      notes: true,
      photo: true,
      background_asset_id: null,
      logo_asset_id: null,
      group_ids: ["00000000-0000-0000-0000-000000000401"],
    });
  });
  it("paginates locations and people with the original rows/total fields", async () => {
    const locations = await appFetch("/api/v1/locations?limit=1&offset=1");
    await expect(locations.json()).resolves.toMatchObject({
      rows: [{ id: secondLocation }],
      total: 2,
    });
    const users = await appFetch(`/api/v1/users?location_id=${reviewRoom}&limit=2&offset=1`);
    const body = await users.json<{
      rows: { id: string; display_name: string; upn: string }[];
      total: number;
    }>();
    expect(body.total).toBe(3);
    expect(body.rows).toHaveLength(2);
    expect(
      body.rows.every(
        (row) => uuid.test(row.id) && row.upn.endsWith(".invalid") && row.display_name,
      ),
    ).toBe(true);
    const filtered = await appFetch(`/api/v1/users?location_id=${reviewRoom}&search=alex`);
    await expect(filtered.json()).resolves.toMatchObject({ rows: [{ id: person }], total: 1 });
    expect((await appFetch("/api/v1/users?location_id=invalid")).status).toBe(404);
    expect((await appFetch("/api/v1/users?limit=-1")).status).toBe(400);
  });
  it("accepts and discards original multipart check-ins and returns UUID DTOs", async () => {
    const form = checkinForm(reviewRoom, "check_in");
    form.set(
      "photo",
      new File([new Uint8Array([0xff, 0xd8, 0xff, 0xd9])], "selfie.jpg", { type: "image/jpeg" }),
    );
    const response = await appFetch("/api/v1/checkins", { method: "POST", body: form });
    const body = await response.json<{
      id: string;
      user_id: string;
      location_id: string;
      direction: string;
    }>();
    expect(response.status).toBe(201);
    expect(body.id).toMatch(uuid);
    expect(body).toMatchObject({ user_id: person, location_id: reviewRoom, direction: "check_in" });
    const checkout = await appFetch("/api/v1/checkins", {
      method: "POST",
      body: checkinForm(secondLocation, "check_out"),
    });
    expect(checkout.status).toBe(201);
    await expect(checkout.json()).resolves.toMatchObject({
      user_id: person,
      location_id: secondLocation,
      direction: "check_out",
    });
  });
  it("requires configured photos and rejects notes when disabled", async () => {
    const form = checkinForm(reviewRoom, "check_in");
    const missingPhoto = await appFetch("/api/v1/checkins", { method: "POST", body: form });
    await expect(missingPhoto.json()).resolves.toMatchObject({ code: "photo_required" });
    const disabledNotes = checkinForm(secondLocation, "check_in");
    disabledNotes.set("notes", "Review check-in");
    const rejectedNotes = await appFetch("/api/v1/checkins", {
      method: "POST",
      body: disabledNotes,
    });
    await expect(rejectedNotes.json()).resolves.toMatchObject({ code: "notes_disabled" });
    form.set("user_id", "1");
    const invalidPerson = await appFetch("/api/v1/checkins", { method: "POST", body: form });
    await expect(invalidPerson.json()).resolves.toMatchObject({ code: "invalid_user" });
  });
});

function checkinForm(location: string, direction: string) {
  const form = new FormData();
  form.set("user_id", person);
  form.set("location_id", location);
  form.set("direction", direction);
  return form;
}
function appFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("X-API-Key", "reviewkey");
  return exports.default.fetch(new Request(`${origin}${path}`, { ...init, headers }));
}
