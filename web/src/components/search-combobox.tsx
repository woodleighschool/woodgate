import { useCallback, useState, type ReactNode } from "react";

import { InputGroupLoadingAddon } from "@components/input-group-loading-addon";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@components/ui/combobox";
import { useDebouncedCallback } from "@hooks/use-debounced-callback";

export function SearchCombobox<TItem>({
  id,
  "aria-invalid": invalid,
  items,
  count,
  value,
  inputValue,
  loading,
  placeholder,
  emptyMessage,
  itemKey,
  itemLabel,
  renderItem,
  required,
  onBlur,
  onInputValueChange,
  onValueChange,
}: {
  id?: string;
  "aria-invalid"?: true;
  items: TItem[];
  count?: number;
  value: TItem | null;
  inputValue: string;
  loading: boolean;
  placeholder: string;
  emptyMessage: string;
  itemKey: (item: TItem) => string;
  itemLabel: (item: TItem) => string;
  renderItem?: (item: TItem) => ReactNode;
  required?: boolean;
  onBlur?: () => void;
  onInputValueChange: (value: string) => void;
  onValueChange: (value: TItem | null) => void;
}) {
  return (
    <Combobox
      items={items}
      filter={null}
      value={value}
      inputValue={inputValue}
      itemToStringLabel={itemLabel}
      itemToStringValue={itemKey}
      isItemEqualToValue={(item, selected) => itemKey(item) === itemKey(selected)}
      onInputValueChange={(next, details) => {
        if (details.reason === "item-press") return;
        onInputValueChange(next);
        if (value !== null && next !== itemLabel(value)) onValueChange(null);
      }}
      onValueChange={(next) => {
        onInputValueChange(next ? itemLabel(next) : "");
        onValueChange(next);
      }}
    >
      <ComboboxInput
        id={id}
        aria-invalid={invalid}
        className="w-full"
        placeholder={placeholder}
        required={required}
        showClear={inputValue !== ""}
        showTrigger={!loading}
        onBlur={onBlur}
      >
        {loading ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      <ComboboxContent>
        <ComboboxEmpty>{loading ? "Loading..." : emptyMessage}</ComboboxEmpty>
        <ComboboxList>
          {(item) => (
            <ComboboxItem key={itemKey(item)} value={item}>
              {renderItem?.(item) ?? itemLabel(item)}
            </ComboboxItem>
          )}
        </ComboboxList>
        {!loading && count !== undefined && count > items.length ? (
          <output className="block border-t p-2 text-xs text-muted-foreground">
            Showing {items.length} of {count} results. Type to narrow your search.
          </output>
        ) : null}
      </ComboboxContent>
    </Combobox>
  );
}

export function useSearchCombobox(initialValue: string, debounceMs = 200) {
  const [inputValue, setInputValue] = useState(initialValue);
  const [q, setQ] = useState(initialValue.trim());
  const { run: updateQuery, cancel: cancelQuery } = useDebouncedCallback(
    (next: string) => setQ(next),
    debounceMs,
  );

  const updateInput = useCallback(
    (nextValue: string) => {
      setInputValue(nextValue);
      const nextQuery = nextValue.trim();
      if (nextQuery === "") {
        cancelQuery();
        setQ("");
      } else {
        updateQuery(nextQuery);
      }
    },
    [cancelQuery, updateQuery],
  );

  return { inputValue, q, pending: inputValue.trim() !== q, setInputValue: updateInput };
}
