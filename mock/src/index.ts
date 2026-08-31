const stationSecret = "testing123";
const stationSubprotocol = "woodgate-station.v1";

const configuration = {
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
};

const people = [
  { id: 1, name: "Alex Example", email: "alex@example.invalid" },
  { id: 2, name: "Sam Sample", email: "sam@example.invalid" },
  { id: 3, name: "Taylor Test", email: "taylor@example.invalid" },
];

export default {
  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);

    if (request.method === "GET" && url.pathname === "/") {
      return Response.json({
        service: "WoodGate development mock",
        station_secret: stationSecret,
      });
    }

    if (!url.pathname.startsWith("/api/station/v1/")) {
      return problem(404, "Not found.", "not_found");
    }

    if (request.headers.get("Authorization") !== `Bearer ${stationSecret}`) {
      return problem(401, "The Station secret is invalid.", "unauthorized");
    }

    if (request.method === "GET" && url.pathname === "/api/station/v1/configuration") {
      return Response.json(configuration);
    }

    if (request.method === "GET" && url.pathname === "/api/station/v1/people") {
      return Response.json({ items: people, count: people.length });
    }

    if (request.method === "POST" && url.pathname === "/api/station/v1/checkins") {
      return createCheckin(request);
    }

    if (request.method === "GET" && url.pathname === "/api/station/v1/connect") {
      return connect(request);
    }

    return problem(404, "Not found.", "not_found");
  },
} satisfies ExportedHandler;

async function createCheckin(request: Request): Promise<Response> {
  if (!request.headers.get("Content-Type")?.startsWith("multipart/form-data;")) {
    return problem(415, "Expected a multipart form.", "unsupported_media_type");
  }

  let form: FormData;
  try {
    form = await request.formData();
  } catch {
    return problem(400, "The check-in form is invalid.", "invalid_form");
  }

  const personID = numberField(form, "person_id");
  if (personID === undefined || !people.some((person) => person.id === personID)) {
    return problem(400, "Choose a valid person.", "invalid_person");
  }

  const direction = stringField(form, "direction");
  if (direction !== "check_in" && direction !== "check_out") {
    return problem(400, "Choose check in or check out.", "invalid_direction");
  }

  const notes = stringField(form, "notes")?.trim();
  if (!notes) {
    return problem(400, "Add notes to continue.", "notes_required");
  }

  const photo = form.get("photo");
  if (!(photo instanceof File) || photo.size === 0 || photo.type !== "image/jpeg") {
    return problem(400, "Add a JPEG photo to continue.", "photo_required");
  }

  return Response.json({
    id: 1,
    person_id: personID,
    location_id: configuration.location.id,
    direction,
  });
}

function connect(request: Request): Response {
  if (request.headers.get("Upgrade")?.toLowerCase() !== "websocket") {
    return problem(426, "Expected a WebSocket connection.", "upgrade_required");
  }

  const protocols = request.headers
    .get("Sec-WebSocket-Protocol")
    ?.split(",")
    .map((protocol) => protocol.trim());
  if (!protocols?.includes(stationSubprotocol)) {
    return problem(400, "The Station WebSocket protocol is required.", "invalid_protocol");
  }

  const [client, server] = Object.values(new WebSocketPair());
  server.accept();
  server.addEventListener("message", (event) => {
    if (typeof event.data !== "string") {
      server.close(1003, "Text messages only");
    }
  });
  server.send(JSON.stringify({ type: "hello" }));

  return new Response(null, {
    status: 101,
    headers: { "Sec-WebSocket-Protocol": stationSubprotocol },
    webSocket: client,
  });
}

function numberField(form: FormData, name: string): number | undefined {
  const value = stringField(form, name);
  if (value === undefined) {
    return undefined;
  }

  const number = Number(value);
  return Number.isSafeInteger(number) ? number : undefined;
}

function stringField(form: FormData, name: string): string | undefined {
  const value = form.get(name);
  return typeof value === "string" ? value : undefined;
}

function problem(status: number, detail: string, code: string): Response {
  return Response.json({ status, detail, code, field_errors: null }, { status });
}
