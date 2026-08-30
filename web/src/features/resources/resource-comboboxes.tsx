import { useMemo } from "react";

import { SearchCombobox, useSearchCombobox } from "@components/search-combobox";

import { useLocation, useLocations } from "./queries";

const SEARCH_PAGE_SIZE = 20;

interface ResourceComboboxProps {
  id?: string;
  value: string;
  required?: boolean;
  onBlur?: () => void;
  onChange: (value: string) => void;
}

export function LocationCombobox(props: ResourceComboboxProps) {
  const selectedID = positiveID(props.value);
  const selected = useLocation(selectedID);
  const search = useSearchCombobox(selected.data?.name ?? "");
  const matches = useLocations({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const items = useMemo(
    () => mergeResources(selected.data, pending ? [] : (matches.data?.items ?? [])),
    [matches.data?.items, pending, selected.data],
  );
  const current = items.find((location) => location.id === selectedID) ?? null;
  const error = selected.error ?? matches.error;
  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <SearchCombobox
      id={props.id}
      items={items}
      value={current}
      inputValue={search.inputValue}
      loading={pending}
      placeholder="Select a location"
      emptyMessage="No locations found."
      itemKey={(location) => String(location.id)}
      itemLabel={(location) => location.name}
      required={props.required}
      onBlur={props.onBlur}
      onInputValueChange={search.setInputValue}
      onValueChange={(location) => {
        props.onChange(location ? String(location.id) : "");
        search.setInputValue(location?.name ?? "");
      }}
    />
  );
}

function mergeResources<T extends { id: number }>(selected: T | undefined, matches: T[]): T[] {
  const items = new Map<number, T>();
  if (selected) items.set(selected.id, selected);
  for (const match of matches) items.set(match.id, match);
  return [...items.values()];
}

function positiveID(value: string): number | null {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}
