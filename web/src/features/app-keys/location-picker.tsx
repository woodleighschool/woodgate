import { useMemo } from "react";

import { useSearchCombobox } from "@components/search-combobox";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from "@components/ui/combobox";
import { Spinner } from "@components/ui/spinner";
import type { AppKeyLocationSummary } from "@lib/api";

import { useAppKeyLocations } from "./queries";

const SEARCH_PAGE_SIZE = 50;

export function LocationPicker({
  id,
  value,
  onChange,
}: {
  id?: string;
  value: AppKeyLocationSummary[];
  onChange: (value: AppKeyLocationSummary[]) => void;
}) {
  const search = useSearchCombobox("");
  const matches = useAppKeyLocations({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const rows = useMemo(
    () => mergeLocations(value, pending ? [] : (matches.data?.items ?? [])),
    [matches.data?.items, pending, value],
  );
  const anchorRef = useComboboxAnchor();
  const error = matches.error;

  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <Combobox
      multiple
      items={rows}
      filter={null}
      value={value}
      inputValue={search.inputValue}
      itemToStringLabel={(location) => location.name}
      itemToStringValue={(location) => String(location.id)}
      isItemEqualToValue={(location, candidate) => location.id === candidate.id}
      onInputValueChange={(next, details) => {
        if (details.reason !== "item-press") search.setInputValue(next);
      }}
      onValueChange={(next) => {
        onChange(next);
        search.setInputValue("");
      }}
    >
      <ComboboxChips ref={anchorRef} className="h-auto min-h-9 pr-2">
        <ComboboxValue>
          {(current: AppKeyLocationSummary[]) => (
            <>
              {current.map((location) => (
                <ComboboxChip key={location.id}>{location.name}</ComboboxChip>
              ))}
              <ComboboxChipsInput
                id={id}
                className="h-[calc(--spacing(5.5))] min-w-32 flex-1 p-0 text-sm"
                placeholder="Add location"
              />
            </>
          )}
        </ComboboxValue>
        {pending ? <Spinner className="size-3.5" /> : null}
      </ComboboxChips>
      {pending ? null : (
        <ComboboxContent anchor={anchorRef}>
          <ComboboxEmpty>No locations found.</ComboboxEmpty>
          <ComboboxList>
            {(item: AppKeyLocationSummary) => (
              <ComboboxItem key={item.id} value={item}>
                {item.name}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

function mergeLocations<T extends AppKeyLocationSummary>(...sets: T[][]): T[] {
  const locations = new Map<number, T>();
  for (const set of sets) {
    for (const location of set) locations.set(location.id, location);
  }
  return [...locations.values()];
}
