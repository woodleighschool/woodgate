import { Fragment, useMemo } from "react";

import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
import { useSearchCombobox } from "@components/search-combobox";
import {
  Combobox,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxInput,
  ComboboxItem,
  ComboboxLabel,
  ComboboxList,
  ComboboxSeparator,
} from "@components/ui/combobox";
import {
  DIRECTORY_SOURCES,
  DIRECTORY_SOURCE_VALUES,
  type DirectorySource,
} from "@features/directory/source";
import type { User } from "@lib/api";

import { useUser, useUsers } from "./queries";

const SEARCH_PAGE_SIZE = 20;

export function UserCombobox({
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
  const selectedQuery = useUser(selectedID);
  const search = useSearchCombobox(selectedQuery.data ? userLabel(selectedQuery.data) : "");
  const matches = useUsers({ q: search.q || undefined, per_page: SEARCH_PAGE_SIZE });
  const pending = search.pending || matches.isPending || matches.isPlaceholderData;
  const rows = useMemo(
    () => mergeUsers(selectedQuery.data, pending ? [] : (matches.data?.items ?? [])),
    [matches.data?.items, pending, selectedQuery.data],
  );
  const selected = rows.find((user) => user.id === selectedID) ?? null;
  const groups = userGroups(rows);
  const error = selectedQuery.error ?? matches.error;

  if (error) return <p className="text-sm text-destructive">{error.message}</p>;

  return (
    <Combobox
      items={groups}
      filter={null}
      value={selected}
      inputValue={search.inputValue}
      itemToStringLabel={userLabel}
      itemToStringValue={(user) => String(user.id)}
      isItemEqualToValue={(user, candidate) => user.id === candidate.id}
      onInputValueChange={(next, details) => {
        if (details.reason === "item-press") return;
        search.setInputValue(next);
        if (selected !== null && next !== userLabel(selected)) onChange("");
      }}
      onValueChange={(next) => {
        onChange(next ? String(next.id) : "");
        search.setInputValue(next ? userLabel(next) : "");
      }}
    >
      <ComboboxInput
        id={id}
        className="w-full"
        placeholder="Select a user"
        showClear={search.inputValue !== ""}
        showTrigger={!pending}
        onBlur={onBlur}
      >
        {pending ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      {pending ? null : (
        <ComboboxContent>
          <ComboboxEmpty>No users found.</ComboboxEmpty>
          <ComboboxList>
            {(group: UserGroup, index: number) => (
              <Fragment key={group.source}>
                {index > 0 ? <ComboboxSeparator /> : null}
                <ComboboxGroup items={group.items}>
                  <ComboboxLabel>{DIRECTORY_SOURCES[group.source].name}</ComboboxLabel>
                  <ComboboxCollection>
                    {(item: User) => (
                      <ComboboxItem key={item.id} value={item}>
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate">{userLabel(item)}</span>
                          {item.name ? (
                            <span className="truncate text-xs text-muted-foreground">
                              {item.email}
                            </span>
                          ) : null}
                        </span>
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

function mergeUsers(selected: User | undefined, matches: User[]): User[] {
  const users = new Map<number, User>();
  if (selected) users.set(selected.id, selected);
  for (const user of matches) users.set(user.id, user);
  return [...users.values()];
}

interface UserGroup {
  source: DirectorySource;
  items: User[];
}

function userGroups(items: User[]): UserGroup[] {
  return DIRECTORY_SOURCE_VALUES.flatMap((source) => {
    const matches = items.filter((user) => user.source === source);
    return matches.length > 0 ? [{ source, items: matches }] : [];
  });
}

function userLabel(user: User): string {
  return user.name || user.email;
}

function positiveID(value: string): number | null {
  const id = Number(value);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}
