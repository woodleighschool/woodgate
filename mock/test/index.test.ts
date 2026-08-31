import { exports } from "cloudflare:workers";
import { describe, expect, it } from "vitest";

const origin = "https://mock.example.invalid";
const authorization = "Bearer testing123";

describe("WoodGate development mock", () => {
  it("publishes its development credential", async () => {
    const response = await exports.default.fetch(`${origin}/`);

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      service: "WoodGate development mock",
      station_secret: "testing123",
    });
  });

  it("requires the Station secret", async () => {
    const response = await exports.default.fetch(`${origin}/api/station/v1/configuration`);

    expect(response.status).toBe(401);
  });

  it("returns the static Station configuration", async () => {
    const response = await stationFetch("/api/station/v1/configuration");

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      station_id: 1,
      station_name: "Development Station",
      location: {
        id: 1,
        name: "Testing Room",
        enabled: true,
        notes: true,
        photo: true,
        background_object_id: null,
        logo_object_id: null,
      },
    });
  });

  it("returns synthetic people", async () => {
    const response = await stationFetch("/api/station/v1/people");
    const body = await response.json<{ items: Array<{ email: string }>; count: number }>();

    expect(response.status).toBe(200);
    expect(body.count).toBe(3);
    expect(body.items.every((person) => person.email.endsWith(".invalid"))).toBe(true);
  });

  it("accepts and discards a complete check-in", async () => {
    const form = new FormData();
    form.set("person_id", "2");
    form.set("direction", "check_in");
    form.set("notes", "Development check-in");
    form.set(
      "photo",
      new File([new Uint8Array([0xff, 0xd8, 0xff, 0xd9])], "selfie.jpg", { type: "image/jpeg" }),
    );

    const response = await stationFetch("/api/station/v1/checkins", {
      method: "POST",
      body: form,
    });

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toEqual({
      id: 1,
      person_id: 2,
      location_id: 1,
      direction: "check_in",
    });
  });

  it("rejects invalid people", async () => {
    const form = new FormData();
    form.set("person_id", "99");
    form.set("direction", "check_out");
    form.set("notes", "Development check-out");
    form.set("photo", new File([new Uint8Array([0xff])], "selfie.jpg", { type: "image/jpeg" }));

    const response = await stationFetch("/api/station/v1/checkins", {
      method: "POST",
      body: form,
    });

    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toMatchObject({ code: "invalid_person" });
  });

  it("upgrades the Station control connection", async () => {
    const response = await stationFetch("/api/station/v1/connect", {
      headers: {
        Upgrade: "websocket",
        "Sec-WebSocket-Protocol": "woodgate-station.v1",
      },
    });

    expect(response.status).toBe(101);
    expect(response.headers.get("Sec-WebSocket-Protocol")).toBe("woodgate-station.v1");
    expect(response.webSocket).not.toBeNull();
  });
});

function stationFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("Authorization", authorization);

  return exports.default.fetch(
    new Request(`${origin}${path}`, {
      ...init,
      headers,
    }),
  );
}
