import { Fragment, useMemo } from "react";

import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
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
  ComboboxInput,
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
import type { Group } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";

import { useGroup, useGroups } from "./queries";

const SEARCH_PAGE_SIZE = 50;

export function GroupPicker({
  id,
  value,
  onChange,
}: {
  id?: string;
  value: number[];
  onChange: (value: number[]) => void;
}) {
  const search = useSearchCombobox("");
  const matches = useGroups({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const selectedQuery = useGroups(
    { values: value.map(String), per_page: MAX_PAGE_SIZE },
    { enabled: value.length > 0 },
  );
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const rows = useMemo(
    () =>
      mergeGroups(
        value.length > 0 ? (selectedQuery.data?.items ?? []) : [],
        pending ? [] : (matches.data?.items ?? []),
      ),
    [matches.data?.items, pending, selectedQuery.data?.items, value.length],
  );
  const selected = value.flatMap((groupID) => {
    const group = rows.find((row) => row.id === groupID);
    return group ? [group] : [];
  });
  const groups = groupOptions(rows);
  const anchorRef = useComboboxAnchor();
  const error = selectedQuery.error ?? matches.error;

  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <Combobox
      multiple
      items={groups}
      filter={null}
      value={selected}
      inputValue={search.inputValue}
      itemToStringLabel={(group) => group.display_name}
      itemToStringValue={(group) => String(group.id)}
      isItemEqualToValue={(group, candidate) => group.id === candidate.id}
      onInputValueChange={(next, details) => {
        if (details.reason !== "item-press") search.setInputValue(next);
      }}
      onValueChange={(next) => {
        onChange(next.map((group) => group.id));
        search.setInputValue("");
      }}
    >
      <ComboboxChips ref={anchorRef} className="h-auto min-h-9 pr-2">
        <ComboboxValue>
          {(current: Group[]) => (
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
                    {(item: Group) => (
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

export function GroupCombobox({
  id,
  value,
  onBlur,
  onChange,
}: {
  id?: string;
  value: string;
  onBlur?: () => void;
  onChange: (value: string) => void;
}) {
  const selectedID = positiveID(value);
  const selectedQuery = useGroup(selectedID);
  const search = useSearchCombobox(selectedQuery.data?.display_name ?? "");
  const matches = useGroups({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const rows = useMemo(
    () =>
      mergeGroups(
        selectedQuery.data ? [selectedQuery.data] : [],
        pending ? [] : (matches.data?.items ?? []),
      ),
    [matches.data?.items, pending, selectedQuery.data],
  );
  const selected = rows.find((group) => group.id === selectedID) ?? null;
  const groups = groupOptions(rows);
  const error = selectedQuery.error ?? matches.error;

  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <Combobox
      items={groups}
      filter={null}
      value={selected}
      inputValue={search.inputValue}
      itemToStringLabel={(group) => group.display_name}
      itemToStringValue={(group) => String(group.id)}
      isItemEqualToValue={(group, candidate) => group.id === candidate.id}
      onInputValueChange={(next, details) => {
        if (details.reason === "item-press") return;
        search.setInputValue(next);
        if (selected !== null && next !== selected.display_name) onChange("");
      }}
      onValueChange={(next) => {
        onChange(next ? String(next.id) : "");
        search.setInputValue(next?.display_name ?? "");
      }}
    >
      <ComboboxInput
        id={id}
        className="w-full"
        placeholder="Select a directory group"
        showClear={search.inputValue !== ""}
        showTrigger={!pending}
        onBlur={onBlur}
      >
        {pending ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      {pending ? null : (
        <ComboboxContent>
          <ComboboxEmpty>No directory groups found.</ComboboxEmpty>
          <ComboboxList>
            {(group: GroupOption, index: number) => (
              <Fragment key={group.source}>
                {index > 0 ? <ComboboxSeparator /> : null}
                <ComboboxGroup items={group.items}>
                  <ComboboxLabel>{DIRECTORY_SOURCES[group.source].name}</ComboboxLabel>
                  <ComboboxCollection>
                    {(item: Group) => (
                      <ComboboxItem key={item.id} value={item}>
                        {item.display_name}
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

function mergeGroups(...sets: Group[][]): Group[] {
  const groups = new Map<number, Group>();
  for (const set of sets) {
    for (const group of set) groups.set(group.id, group);
  }
  return [...groups.values()];
}

interface GroupOption {
  source: DirectorySource;
  items: Group[];
}

function groupOptions(items: Group[]): GroupOption[] {
  return DIRECTORY_SOURCE_VALUES.flatMap((source) => {
    const matches = items.filter((group) => group.source === source);
    return matches.length > 0 ? [{ source, items: matches }] : [];
  });
}

function positiveID(value: string): number | null {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}
