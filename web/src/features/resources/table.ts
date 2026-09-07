import { useNavigate, useSearch } from "@tanstack/react-router";

import { useDataTableSearch } from "@components/data-table/use-data-table-search";

import { parseResourceSearch, type ResourceSearch } from "./search";

export function useResourceTable(defaultSort: string) {
  const raw = useSearch({ strict: false }) as Record<string, unknown>;
  const navigate = useNavigate();
  const search = parseResourceSearch(raw, defaultSort);
  const update = (updater: (previous: ResourceSearch) => ResourceSearch) => {
    void navigate({
      to: ".",
      search: (previous: Record<string, unknown>) => {
        const next = updater(parseResourceSearch(previous, defaultSort));
        const { filter: _filter, perPage: _perPage, order: _order, ...normalized } = next;
        return normalized;
      },
      replace: true,
    });
  };
  const tableSearch = useDataTableSearch({ search, onSearchChange: update });
  const [sort, direction] = search.sort.split(".");
  const params = {
    limit: search.per_page,
    offset: (search.page - 1) * search.per_page,
    search: search.q || undefined,
    sort,
    order: direction === "desc" ? ("desc" as const) : ("asc" as const),
  };
  return {
    tableSearch,
    params,
    search,
    setFilter: (key: string, value: unknown) =>
      update((previous) => ({ ...previous, [key]: value || undefined, page: 1 })),
  };
}
