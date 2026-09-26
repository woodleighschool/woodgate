import { useMemo } from "react";

import { SearchCombobox, useSearchCombobox } from "@components/search-combobox";
import type { StationLocation } from "@lib/api";

import { useStationLocations } from "./queries";

const SEARCH_PAGE_SIZE = 20;

interface ResourceComboboxProps {
  id?: string;
  value: StationLocation | null;
  required?: boolean;
  onBlur?: () => void;
  onChange: (value: StationLocation | null) => void;
}

export function LocationCombobox(props: ResourceComboboxProps) {
  const search = useSearchCombobox(props.value?.name ?? "");
  const matches = useStationLocations({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const items = useMemo(
    () => mergeResources(props.value, pending ? [] : (matches.data?.items ?? [])),
    [matches.data?.items, pending, props.value],
  );
  const current = items.find((location) => location.id === props.value?.id) ?? null;
  const error = matches.error;
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
        props.onChange(location ? { id: location.id, name: location.name } : null);
        search.setInputValue(location?.name ?? "");
      }}
    />
  );
}

function mergeResources<T extends { id: number }>(selected: T | null, matches: T[]): T[] {
  const items = new Map<number, T>();
  if (selected) items.set(selected.id, selected);
  for (const match of matches) items.set(match.id, match);
  return [...items.values()];
}
