import { useForm } from "@tanstack/react-form";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { MapPin, Plus } from "lucide-react";
import { useState } from "react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { ConfirmDialog } from "@components/confirm-dialog";
import type { DataTableColumnDef } from "@components/data-table/types";
import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button } from "@components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Skeleton } from "@components/ui/skeleton";
import { Switch } from "@components/ui/switch";
import { Textarea } from "@components/ui/textarea";
import { toast } from "@components/ui/toast";
import { ValidatedFormField } from "@components/validated-form-field";
import { useCanAccess } from "@features/auth/access";
import {
  ChoiceSelect,
  DateValue,
  exportRows,
  GroupPicker,
  ReferenceName,
  ReferencePicker,
} from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import {
  createLocation,
  deleteLocation,
  getLocation,
  listLocations,
  patchLocation,
  unwrap,
  type Location,
  type LocationWriteRequest,
} from "@lib/api";
import { requiredString } from "@lib/form-validation";

const columns: DataTableColumnDef<Location>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <Link data-text-link to="/locations/$id" params={{ id: row.original.id }}>
        {row.original.name}
      </Link>
    ),
  },
  ...(["enabled", "notes", "photo"] as const).map((key) => ({
    accessorKey: key,
    header: key[0].toUpperCase() + key.slice(1),
    size: 95,
    cell: ({ row }: { row: { original: Location } }) => (
      <BooleanIndicator value={row.original[key]} />
    ),
  })),
  {
    accessorKey: "background_asset_id",
    header: "Background Asset",
    cell: ({ row }) => <ReferenceName resource="assets" id={row.original.background_asset_id} />,
  },
  {
    accessorKey: "logo_asset_id",
    header: "Logo Asset",
    cell: ({ row }) => <ReferenceName resource="assets" id={row.original.logo_asset_id} />,
  },
  {
    id: "groups",
    header: "Groups",
    enableSorting: false,
    cell: ({ row }) => (
      <div className="flex flex-wrap gap-2">
        {row.original.group_ids.map((id) => (
          <ReferenceName key={id} resource="groups" id={id} />
        ))}
      </div>
    ),
  },
];
export function LocationsList() {
  const { tableSearch, params, search, setFilter } = useResourceTable("name.asc");
  const filters = {
    ...params,
    enabled:
      search.enabled === "true" || search.enabled === true
        ? true
        : search.enabled === "false" || search.enabled === false
          ? false
          : undefined,
  };
  const query = useQuery({
    queryKey: ["locations", "list", filters],
    queryFn: ({ signal }) => unwrap(listLocations({ query: filters, signal })),
    placeholderData: keepPreviousData,
  });
  const canCreate = useCanAccess("locations", "create");
  const canDelete = useCanAccess("locations", "delete");
  const queryClient = useQueryClient();
  return (
    <PageShell>
      <PageHeader
        title="Locations"
        icon={<MapPin />}
        actions={
          canCreate ? (
            <Button nativeButton={false} render={<Link to="/locations/create" />}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : undefined
        }
      />
      <ResourceDataTable<Location>
        data={query.data?.rows ?? []}
        count={query.data?.total ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isPending}
        pending={query.isFetching}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<MapPin />}
        emptyTitle="Locations"
        emptyDescription="No locations have been created."
        getRowId={(row) => row.id}
        filters={
          <ChoiceSelect
            label="Enabled"
            value={
              search.enabled === true || search.enabled === "true"
                ? "true"
                : search.enabled === false || search.enabled === "false"
                  ? "false"
                  : "all"
            }
            onChange={(value) => setFilter("enabled", value === "all" ? undefined : value)}
            options={[
              { value: "all", label: "All locations" },
              { value: "true", label: "Enabled" },
              { value: "false", label: "Disabled" },
            ]}
          />
        }
        exportOptions={{
          filename: "locations",
          columns: (
            [
              "id",
              "name",
              "description",
              "enabled",
              "notes",
              "photo",
              "background_asset_id",
              "logo_asset_id",
              "group_ids",
              "created_at",
              "updated_at",
            ] as const
          ).map((key) => ({
            header: key,
            value: (row) =>
              Array.isArray(row[key]) ? JSON.stringify(row[key]) : String(row[key] ?? ""),
          })),
          loadRows: () =>
            exportRows((offset) =>
              unwrap(listLocations({ query: { ...filters, offset, limit: 250 } })),
            ),
        }}
        onDeleteRows={
          canDelete
            ? async (rows) => {
                try {
                  for (const row of rows) await unwrap(deleteLocation({ path: { id: row.id } }));
                } finally {
                  await queryClient.invalidateQueries({ queryKey: ["locations"] });
                }
              }
            : undefined
        }
      />
    </PageShell>
  );
}

export function LocationCreate() {
  return <LocationForm />;
}
export function LocationEdit({ id }: { id: string }) {
  const query = useQuery({
    queryKey: ["locations", id],
    queryFn: ({ signal }) => unwrap(getLocation({ path: { id }, signal })),
  });
  if (query.isPending)
    return (
      <PageShell>
        <Skeleton className="h-80 w-full" />
      </PageShell>
    );
  if (query.isError)
    return (
      <PageShell>
        <QueryError error={query.error} onRetry={() => void query.refetch()} />
      </PageShell>
    );
  return <LocationForm key={id} record={query.data} />;
}

function LocationForm({ record }: { record?: Location }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const canWrite = useCanAccess("locations", record ? "write" : "create");
  const canDelete = useCanAccess("locations", "delete");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const form = useForm({
    defaultValues: {
      name: record?.name ?? "",
      description: record?.description ?? "",
      enabled: record?.enabled ?? false,
      notes: record?.notes ?? false,
      photo: record?.photo ?? false,
      background_asset_id: record?.background_asset_id ?? null,
      logo_asset_id: record?.logo_asset_id ?? null,
      group_ids: record?.group_ids ?? [],
    } satisfies LocationWriteRequest,
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        const body = { ...value, name: value.name.trim() };
        const saved = await unwrap(
          record ? patchLocation({ path: { id: record.id }, body }) : createLocation({ body }),
        );
        form.reset(value);
        await queryClient.invalidateQueries({ queryKey: ["locations"] });
        toast.add({ type: "success", title: "Location saved" });
        await exit.runWithoutPrompt(() =>
          navigate({ to: "/locations/$id", params: { id: saved.id } }),
        );
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to save location"));
      }
    },
  });
  const exit = usePageFormExitGuard({
    form,
    onDiscard: () => navigate({ to: "/locations" }),
    enabled: canWrite,
  });
  const deletion = useMutation({
    mutationFn: () => unwrap(deleteLocation({ path: { id: record!.id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["locations"] });
      await exit.runWithoutPrompt(() => navigate({ to: "/locations" }));
    },
  });
  return (
    <PageShell>
      <PageHeader
        title={record?.name ?? "Create Location"}
        icon={<MapPin />}
        actions={
          <>
            <Button nativeButton={false} variant="outline" render={<Link to="/locations" />}>
              List
            </Button>
            {record && canDelete ? (
              <Button variant="destructive" onClick={() => setConfirmDelete(true)}>
                Delete
              </Button>
            ) : null}
          </>
        }
      />
      <form
        className="max-w-3xl"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <FieldGroup>
          <fieldset disabled={!canWrite} className="flex min-w-0 flex-col gap-6">
            <form.Field
              name="name"
              validators={{ onBlur: requiredString("Name"), onSubmit: requiredString("Name") }}
            >
              {(field) => (
                <ValidatedFormField field={field} label="Name" htmlFor="location-name" required>
                  {(control) => (
                    <Input
                      {...control}
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                      onBlur={field.handleBlur}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
            <form.Field name="description">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Description"
                  htmlFor="location-description"
                >
                  {(control) => (
                    <Textarea
                      {...control}
                      rows={3}
                      value={field.state.value}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
            {(["enabled", "notes", "photo"] as const).map((key) => (
              <form.Field key={key} name={key}>
                {(field) => (
                  <Field orientation="horizontal">
                    <Switch
                      id={`location-${key}`}
                      checked={field.state.value}
                      onCheckedChange={(value) => field.handleChange(value)}
                    />
                    <FieldLabel htmlFor={`location-${key}`}>
                      {key[0].toUpperCase() + key.slice(1)}
                    </FieldLabel>
                  </Field>
                )}
              </form.Field>
            ))}
            {(["background_asset_id", "logo_asset_id"] as const).map((key) => (
              <form.Field key={key} name={key}>
                {(field) => (
                  <Field>
                    <FieldLabel htmlFor={`location-${key}`}>
                      {key === "background_asset_id" ? "Background Asset" : "Logo Asset"}
                    </FieldLabel>
                    <ReferencePicker
                      id={`location-${key}`}
                      resource="assets"
                      value={field.state.value}
                      onChange={(value) => field.handleChange(value)}
                    />
                  </Field>
                )}
              </form.Field>
            ))}
            <form.Field name="group_ids">
              {(field) => (
                <Field>
                  <FieldLabel>Groups</FieldLabel>
                  <GroupPicker
                    value={field.state.value}
                    onChange={(value) => field.handleChange(value)}
                  />
                </Field>
              )}
            </form.Field>
          </fieldset>
          {record ? (
            <div className="text-sm text-muted-foreground">
              Created <DateValue value={record.created_at} /> · Updated{" "}
              <DateValue value={record.updated_at} />
            </div>
          ) : null}
          <QueryError title="Unable to save" error={error} />
          {canWrite ? (
            <FormActions
              nativeSubmit
              submitLabel="Save"
              form={form}
              onCancel={exit.requestDiscard}
            />
          ) : null}
        </FieldGroup>
      </form>
      <QueryError error={deletion.error} />
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete Location?"
        description="This cannot be undone."
        confirmLabel="Delete"
        variant="destructive"
        pending={deletion.isPending}
        onConfirm={() => deletion.mutate()}
      />
      {exit.dialog}
    </PageShell>
  );
}
