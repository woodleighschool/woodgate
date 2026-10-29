import { QueryError } from "@components/query-error";
import { SearchCombobox, useSearchCombobox } from "@components/search-combobox";
import { useCheckinAvailableLocations, useCheckinPeople } from "@features/resources/queries";
import type { CheckinLocationSummary, CheckinPersonSummary } from "@lib/api";

interface ChoiceProps<T> {
  id?: string;
  "aria-invalid"?: true;
  value: T | null;
  onBlur: () => void;
  onChange: (value: T | null) => void;
}

export function CheckinLocationCombobox(props: ChoiceProps<CheckinLocationSummary>) {
  const search = useSearchCombobox(props.value?.name ?? "");
  const query = useCheckinAvailableLocations({ q: search.q || undefined, per_page: 20 });
  const pending = search.pending || query.isPending || query.isPlaceholderData;
  return (
    <>
      <SearchCombobox
        {...props}
        items={pending ? [] : (query.data?.items ?? [])}
        inputValue={search.inputValue}
        loading={pending}
        placeholder="Select a location"
        emptyMessage="No available locations."
        itemKey={(location) => String(location.id)}
        itemLabel={(location) => location.name}
        required
        onInputValueChange={search.setInputValue}
        onValueChange={props.onChange}
      />
      <QueryError error={query.error} onRetry={() => void query.refetch()} />
    </>
  );
}

export function CheckinPersonCombobox({
  locationID,
  ...props
}: ChoiceProps<CheckinPersonSummary> & { locationID: number }) {
  const search = useSearchCombobox(props.value?.name || props.value?.email || "");
  const query = useCheckinPeople({
    location_id: locationID,
    q: search.q || undefined,
    per_page: 20,
  });
  const pending = search.pending || query.isPending;
  return (
    <>
      <SearchCombobox
        {...props}
        items={pending ? [] : (query.data?.items ?? [])}
        inputValue={search.inputValue}
        loading={pending}
        placeholder="Search people"
        emptyMessage="No eligible people found."
        itemKey={(person) => String(person.id)}
        itemLabel={(person) => person.name || person.email}
        renderItem={(person) => (
          <div className="flex flex-col">
            <span>{person.name || person.email}</span>
            <span className="text-sm text-muted-foreground">
              {[person.department, person.email].filter(Boolean).join(" · ")}
            </span>
          </div>
        )}
        required
        onInputValueChange={search.setInputValue}
        onValueChange={props.onChange}
      />
      <QueryError error={query.error} onRetry={() => void query.refetch()} />
    </>
  );
}
