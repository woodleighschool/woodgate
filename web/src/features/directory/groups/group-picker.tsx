import { Fragment, useMemo } from "react";

import { useSearchCombobox } from "@components/search-combobox";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
  ComboboxSeparator,
  ComboboxValue,
  useComboboxAnchor,
} from "@components/ui/combobox";
import { Spinner } from "@components/ui/spinner";
import {
  DIRECTORY_SOURCES,
  DIRECTORY_SOURCE_VALUES,
  type DirectorySource,
} from "@features/directory/source";
import { useLocationGroups } from "@features/resources/queries";
import type { GroupSummary } from "@lib/api";

const SEARCH_PAGE_SIZE = 50;

export function GroupPicker({
  id,
  value,
  onChange,
}: {
  id?: string;
  value: GroupSummary[];
  onChange: (value: GroupSummary[]) => void;
}) {
  const search = useSearchCombobox("");
  const matches = useLocationGroups({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const rows = useMemo(
    () => mergeGroups(value, pending ? [] : (matches.data?.items ?? [])),
    [matches.data?.items, pending, value],
  );
  const groups = groupOptions(rows);
  const anchorRef = useComboboxAnchor();
  const error = matches.error;

  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <Combobox
      multiple
      items={groups}
      filter={null}
      value={value}
      inputValue={search.inputValue}
      itemToStringLabel={(group) => group.display_name}
      itemToStringValue={(group) => String(group.id)}
      isItemEqualToValue={(group, candidate) => group.id === candidate.id}
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
          {(current: GroupSummary[]) => (
            <>
              {current.map((group) => (
                <ComboboxChip key={group.id}>{group.display_name}</ComboboxChip>
              ))}
              <ComboboxChipsInput
                id={id}
                className="h-[calc(--spacing(5.5))] min-w-32 flex-1 p-0 text-sm"
                placeholder="Add directory group"
              />
            </>
          )}
        </ComboboxValue>
        {pending ? <Spinner className="size-3.5" /> : null}
      </ComboboxChips>
      {pending ? null : (
        <ComboboxContent anchor={anchorRef}>
          <ComboboxEmpty>No directory groups found.</ComboboxEmpty>
          <ComboboxList>
            {(group: GroupOption, index: number) => (
              <Fragment key={group.source}>
                {index > 0 ? <ComboboxSeparator /> : null}
                <ComboboxGroup items={group.items}>
                  <ComboboxLabel>{DIRECTORY_SOURCES[group.source].name}</ComboboxLabel>
                  <ComboboxCollection>
                    {(item: GroupSummary) => (
                      <ComboboxItem key={item.id} value={item}>
                        <span className="min-w-0 flex-1 truncate">{item.display_name}</span>
                        {item.mail_nickname ? (
                          <span className="truncate text-xs text-muted-foreground">
                            {item.mail_nickname}
                          </span>
                        ) : null}
                      </ComboboxItem>
                    )}
                  </ComboboxCollection>
                </ComboboxGroup>
              </Fragment>
            )}
          </ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

function mergeGroups<T extends GroupSummary>(...sets: T[][]): T[] {
  const groups = new Map<number, T>();
  for (const set of sets) {
    for (const group of set) groups.set(group.id, group);
  }
  return [...groups.values()];
}

interface GroupOption {
  source: DirectorySource;
  items: GroupSummary[];
}

function groupOptions(items: GroupSummary[]): GroupOption[] {
  return DIRECTORY_SOURCE_VALUES.flatMap((source) => {
    const matches = items.filter((group) => group.source === source);
    return matches.length > 0 ? [{ source, items: matches }] : [];
  });
}
