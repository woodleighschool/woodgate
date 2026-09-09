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
  items,
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
  items: TItem[];
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
      onValueChange={onValueChange}
    >
      <ComboboxInput
        id={id}
        className="w-full"
        placeholder={placeholder}
        required={required}
        showClear={inputValue !== ""}
        showTrigger={!loading}
        onBlur={onBlur}
      >
        {loading ? <InputGroupLoadingAddon /> : null}
      </ComboboxInput>
      {loading ? null : (
        <ComboboxContent>
          <ComboboxEmpty>{emptyMessage}</ComboboxEmpty>
          <ComboboxList>
            {(item) => (
              <ComboboxItem key={itemKey(item)} value={item}>
                {renderItem?.(item) ?? itemLabel(item)}
              </ComboboxItem>
            )}
          </ComboboxList>
        </ComboboxContent>
      )}
    </Combobox>
  );
}

export function useSearchCombobox(selectedLabel: string, debounceMs = 200) {
  const [inputValue, setInputValue] = useState(selectedLabel);
  const [q, setQ] = useState(selectedLabel.trim());
  const [previousLabel, setPreviousLabel] = useState(selectedLabel);
  const { run: updateQuery, cancel: cancelQuery } = useDebouncedCallback(
    (next: string) => setQ(next),
    debounceMs,
  );

  if (selectedLabel !== previousLabel) {
    setPreviousLabel(selectedLabel);
    setInputValue(selectedLabel);
    setQ(selectedLabel.trim());
  }

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
