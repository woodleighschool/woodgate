export const reviewKey = "reviewkey";

const locations = [
  {
    id: "00000000-0000-0000-0000-000000000101",
    name: "Review Room",
    enabled: true,
    notes: true,
    photo: true,
    background_asset_id: null,
    logo_asset_id: null,
    group_ids: ["00000000-0000-0000-0000-000000000401"],
  },
  {
    id: "00000000-0000-0000-0000-000000000102",
    name: "Reception",
    enabled: true,
    notes: false,
    photo: false,
    background_asset_id: null,
    logo_asset_id: null,
    group_ids: ["00000000-0000-0000-0000-000000000401"],
  },
];

const people = [
  {
    id: "00000000-0000-0000-0000-000000000201",
    display_name: "Alex Example",
    upn: "alex@example.invalid",
  },
  {
    id: "00000000-0000-0000-0000-000000000202",
    display_name: "Sam Sample",
    upn: "sam@example.invalid",
  },
  {
    id: "00000000-0000-0000-0000-000000000203",
    display_name: "Taylor Test",
    upn: "taylor@example.invalid",
  },
];

export function reviewPairing(baseURL: string) {
  return { base_url: baseURL, api_key: reviewKey };
}

export async function appRequest(request: Request): Promise<Response | undefined> {
  const url = new URL(request.url);
  if (url.pathname !== "/auth/me" && !url.pathname.startsWith("/api/v1/")) return undefined;
  if (request.headers.get("X-API-Key") !== reviewKey)
    return problem(401, "The API key is invalid.", "unauthorized");

  if (request.method === "GET" && url.pathname === "/auth/me") {
    return Response.json({
      principal: {
        type: "api_key",
        id: "00000000-0000-0000-0000-000000000301",
        name: "App Review",
      },
      admin: false,
      access: [
        { resource: "users", action: "read" },
        { resource: "locations", action: "read" },
        { resource: "assets", action: "read", asset_type: "asset" },
        ...locations.map((location) => ({
          resource: "checkins",
          action: "create",
          location_id: location.id,
        })),
      ],
    });
  }
  if (request.method === "GET" && url.pathname === "/api/v1/locations") {
    const search = url.searchParams.get("search")?.toLowerCase() ?? "";
    return page(
      locations.filter((location) => location.name.toLowerCase().includes(search)),
      url,
    );
  }
  if (request.method === "GET" && url.pathname.startsWith("/api/v1/locations/")) {
    const location = locations.find(
      (entry) => entry.id === url.pathname.slice("/api/v1/locations/".length).toLowerCase(),
    );
    return location ? Response.json(location) : problem(404, "Location not found.", "not_found");
  }
  if (request.method === "GET" && url.pathname === "/api/v1/users") {
    const locationID = url.searchParams.get("location_id");
    if (locationID && !locations.some((location) => location.id === locationID.toLowerCase()))
      return problem(404, "Location not found.", "not_found");
    const search = url.searchParams.get("search")?.toLowerCase() ?? "";
    return page(
      people.filter((person) =>
        `${person.display_name} ${person.upn}`.toLowerCase().includes(search),
      ),
      url,
    );
  }
  if (request.method === "POST" && url.pathname === "/api/v1/checkins")
    return createCheckin(request);
  return problem(404, "Not found.", "not_found");
}

function page<T>(rows: T[], url: URL): Response {
  const limit = Number(url.searchParams.get("limit") ?? 250);
  const offset = Number(url.searchParams.get("offset") ?? 0);
  if (
    !Number.isSafeInteger(limit) ||
    limit < 1 ||
    limit > 250 ||
    !Number.isSafeInteger(offset) ||
    offset < 0
  )
    return problem(400, "Invalid pagination.", "invalid_pagination");
  return Response.json({ rows: rows.slice(offset, offset + limit), total: rows.length });
}

async function createCheckin(request: Request): Promise<Response> {
  if (!request.headers.get("Content-Type")?.startsWith("multipart/form-data;"))
    return problem(415, "Expected a multipart form.", "unsupported_media_type");
  let form: FormData;
  try {
    form = await request.formData();
  } catch {
    return problem(400, "The check-in form is invalid.", "invalid_form");
  }
  const userID = form.get("user_id");
  const location = locations.find((entry) => entry.id === form.get("location_id"));
  if (!location) return problem(400, "Choose a valid location.", "invalid_location");
  if (!people.some((person) => person.id === userID))
    return problem(400, "Choose a valid person.", "invalid_user");
  const direction = form.get("direction");
  if (direction !== "check_in" && direction !== "check_out")
    return problem(400, "Choose check in or check out.", "invalid_direction");
  const notes = form.get("notes");
  if (!location.notes && typeof notes === "string" && notes !== "")
    return problem(400, "Notes are disabled for this location.", "notes_disabled");
  const photo = form.get("photo");
  if (location.photo && (!(photo instanceof File) || photo.size === 0))
    return problem(400, "Add a photo to continue.", "photo_required");
  if (photo instanceof File && photo.size > 0 && !["image/jpeg", "image/png"].includes(photo.type))
    return problem(400, "Choose a PNG or JPEG photo.", "invalid_photo");
  return Response.json(
    { id: crypto.randomUUID(), user_id: userID, location_id: location.id, direction },
    { status: 201 },
  );
}

function problem(status: number, detail: string, code: string): Response {
  return Response.json({ status, detail, code, field_errors: null }, { status });
}
