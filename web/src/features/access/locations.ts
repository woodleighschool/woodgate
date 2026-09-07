import { listLocations, unwrap, type Location } from "@lib/api";

export async function loadAccessLocations(
  signal: AbortSignal,
  enabled?: boolean,
): Promise<Location[]> {
  const locations: Location[] = [];
  for (;;) {
    const result = await unwrap(
      listLocations({
        query: { limit: 250, offset: locations.length, sort: "name", order: "asc", enabled },
        signal,
      }),
    );
    locations.push(...result.rows);
    if (result.rows.length === 0 || locations.length >= result.total) return locations;
  }
}
