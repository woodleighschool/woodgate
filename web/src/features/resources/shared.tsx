import { useQuery } from "@tanstack/react-query";
import { X } from "lucide-react";
import { useState, type ReactNode } from "react";

import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { SearchCombobox, useSearchCombobox } from "@components/search-combobox";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { useCanAccess } from "@features/auth/access";
import {
  getAsset,
  getGroup,
  getUser,
  getLocation,
  getApiKey,
  listAssets,
  listGroups,
  unwrap,
} from "@lib/api";

export function DateValue({ value }: { value?: string | null }) {
  return value ? <time dateTime={value}>{new Date(value).toLocaleString()}</time> : <>-</>;
}

export function ChoiceSelect({
  value,
  onChange,
  label,
  options,
  id,
}: {
  value: string;
  onChange: (value: string) => void;
  label: string;
  options: { value: string; label: string }[];
  id?: string;
}) {
  return (
    <Select value={value} onValueChange={(next) => onChange(next ?? "")} items={options}>
      <SelectTrigger id={id} aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
}

type ReferenceResource = "users" | "groups" | "assets" | "locations" | "api-keys";
async function loadReference(resource: ReferenceResource, id: string, signal: AbortSignal) {
  switch (resource) {
    case "users":
      return unwrap(getUser({ path: { id }, signal }));
    case "groups":
      return unwrap(getGroup({ path: { id }, signal }));
    case "assets":
      return unwrap(getAsset({ path: { id }, signal }));
    case "locations":
      return unwrap(getLocation({ path: { id }, signal }));
  }
  return unwrap(getApiKey({ path: { id }, signal }));
}

export function ReferenceName({
  resource,
  id,
  field,
  fallback,
}: {
  resource: ReferenceResource;
  id?: string | null;
  field?: "department";
  fallback?: ReactNode;
}) {
  const allowed = useCanAccess(resource, "read");
  const query = useQuery({
    queryKey: [resource, id],
    queryFn: ({ signal }) => loadReference(resource, id!, signal),
    enabled: allowed && Boolean(id),
    staleTime: 30_000,
  });
  if (!id) return <>-</>;
  const record = query.data;
  const label = record
    ? field === "department" && "department" in record
      ? record.department
      : "display_name" in record
        ? record.display_name
        : record.name || id
    : fallback || id;
  if (!allowed || query.isError || field) return <>{label}</>;
  return (
    <Link
      to={
        resource === "locations"
          ? "/locations/$id"
          : resource === "users"
            ? "/users/$id/show"
            : resource === "groups"
              ? "/groups/$id/show"
              : resource === "assets"
                ? "/assets/$id/show"
                : "/api-keys/$id/show"
      }
      params={{ id }}
      data-text-link
    >
      {label}
    </Link>
  );
}

interface ReferenceOption {
  id: string;
  name?: string | null;
}
export function ReferencePicker({
  resource,
  value,
  onChange,
  id,
}: {
  resource: "assets" | "groups";
  value: string | null;
  onChange: (value: string | null) => void;
  id?: string;
}) {
  const allowed = useCanAccess(resource, "read");
  const selected = useQuery({
    queryKey: [resource, value],
    queryFn: ({ signal }) => loadReference(resource, value!, signal),
    enabled: allowed && Boolean(value),
  });
  const selectedName =
    selected.data && "name" in selected.data ? selected.data.name || value || "" : value || "";
  const search = useSearchCombobox(selectedName);
  const query = useQuery<{ rows: ReferenceOption[]; total: number }>({
    queryKey: [resource, "picker", search.q],
    queryFn: async ({ signal }) =>
      resource === "assets"
        ? unwrap(
            listAssets({
              query: {
                search: search.q || undefined,
                type: "asset",
                limit: 100,
                sort: "name",
                order: "asc",
              },
              signal,
            }),
          )
        : unwrap(
            listGroups({
              query: { search: search.q || undefined, limit: 100, sort: "name", order: "asc" },
              signal,
            }),
          ),
    enabled: allowed,
  });
  if (!allowed) return <p className="text-sm text-muted-foreground">{value || "None"}</p>;
  const items: ReferenceOption[] = query.data?.rows ?? [];
  const selectedOption = value ? { id: value, name: selectedName } : null;
  return (
    <div className="flex flex-col gap-2">
      <SearchCombobox
        id={id}
        items={items}
        value={selectedOption}
        inputValue={search.inputValue}
        loading={query.isFetching}
        placeholder={`Search ${resource}…`}
        emptyMessage={`No ${resource} found`}
        itemKey={(item) => item.id}
        itemLabel={(item) => item.name || item.id}
        onInputValueChange={search.setInputValue}
        onValueChange={(item) => onChange(item?.id ?? null)}
      />
      <QueryError error={query.error} onRetry={() => void query.refetch()} />
    </div>
  );
}

export function GroupPicker({
  value,
  onChange,
}: {
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const [pickerKey, setPickerKey] = useState(0);
  const allowed = useCanAccess("groups", "read");
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        {value.map((id) => (
          <Badge key={id} variant="secondary">
            <ReferenceName resource="groups" id={id} />
            {allowed ? (
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                aria-label="Remove group"
                onClick={() => onChange(value.filter((entry) => entry !== id))}
              >
                <X />
              </Button>
            ) : null}
          </Badge>
        ))}
      </div>
      {allowed ? (
        <ReferencePicker
          key={pickerKey}
          resource="groups"
          value={null}
          onChange={(id) => {
            if (id && !value.includes(id)) onChange([...value, id]);
            setPickerKey((previous) => previous + 1);
          }}
        />
      ) : null}
    </div>
  );
}

export async function exportRows<T>(
  load: (offset: number) => Promise<{ rows: T[]; total: number }>,
): Promise<T[]> {
  const rows: T[] = [];
  for (;;) {
    const result = await load(rows.length);
    rows.push(...result.rows);
    if (rows.length >= result.total || result.rows.length === 0) return rows;
  }
}
