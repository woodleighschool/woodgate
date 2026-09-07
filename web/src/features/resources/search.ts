export interface ResourceSearch {
  q?: string;
  page: number;
  per_page: number;
  sort: string;
  [key: string]: unknown;
}

export function parseResourceSearch(
  raw: Record<string, unknown>,
  defaultSort: string,
): ResourceSearch {
  let filter: Record<string, unknown> = {};
  if (raw.filter && typeof raw.filter === "object")
    filter = Object.fromEntries(Object.entries(raw.filter));
  if (typeof raw.filter === "string") {
    try {
      const parsed: unknown = JSON.parse(raw.filter);
      if (parsed && typeof parsed === "object") filter = Object.fromEntries(Object.entries(parsed));
    } catch {
      /* Invalid bookmarks use the default filters. */
    }
  }
  const page = Number(raw.page);
  const perPage = Number(raw.per_page ?? raw.perPage ?? 10);
  const sort = typeof raw.sort === "string" ? raw.sort : defaultSort;
  return {
    ...filter,
    ...raw,
    q:
      typeof raw.q === "string"
        ? raw.q
        : typeof filter.search === "string"
          ? filter.search
          : undefined,
    page: Number.isInteger(page) && page > 0 ? page : 1,
    per_page: Number.isInteger(perPage) && perPage > 0 ? Math.min(perPage, 250) : 10,
    sort: sort.includes(".")
      ? sort
      : `${sort}.${(typeof raw.order === "string" ? raw.order.toLowerCase() : "asc") === "desc" ? "desc" : "asc"}`,
  };
}
