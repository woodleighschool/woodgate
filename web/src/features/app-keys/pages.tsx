import { useForm } from "@tanstack/react-form";
import { getRouteApi, useNavigate, useParams } from "@tanstack/react-router";
import { Copy, KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { DataTableColumnDef } from "@components/data-table/types";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { FormActions } from "@components/form-actions";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { RelativeTime } from "@components/relative-time";
import { ResourceDataTable } from "@components/resource-data-table";
import { TokenList } from "@components/token-list";
import { Button } from "@components/ui/button";
import { Checkbox } from "@components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@components/ui/dialog";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Skeleton } from "@components/ui/skeleton";
import { toast } from "@components/ui/toast";
import { ValidatedFormField } from "@components/validated-form-field";
import { useCan } from "@features/authz/access";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { AppKeyKey, AppKeyMutation, AppKeyCreatedKey } from "@lib/api";
import { requiredString } from "@lib/form-validation";
import { parseRouteID } from "@lib/route-params";
import { runtime } from "@lib/runtime";

import { LocationPicker } from "./location-picker";
import { pairingPayload } from "./pairing";
import {
  useAppKey,
  useAppKeys,
  useCreateAppKey,
  useDeleteAppKey,
  useUpdateAppKey,
} from "./queries";

const listRoute = getRouteApi("/_authenticated/app-keys/");
const columns: DataTableColumnDef<AppKeyKey>[] = [
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink to="/app-keys/$id" params={{ id: String(row.original.id) }}>
        {row.original.name}
      </TextLink>
    ),
  },
  { accessorKey: "key_prefix", header: "Prefix", enableSorting: false },
  {
    id: "locations",
    header: "Locations",
    enableSorting: false,
    cell: ({ row }) =>
      row.original.all_locations ? (
        "All locations"
      ) : (
        <TokenList values={row.original.locations.map((location) => location.name)} />
      ),
  },
  {
    accessorKey: "expires_at",
    header: "Expires",
    cell: ({ row }) =>
      row.original.expires_at ? <RelativeTime value={row.original.expires_at} /> : "Never",
  },
  {
    accessorKey: "last_used_at",
    header: "Last Used",
    cell: ({ row }) =>
      row.original.last_used_at ? <RelativeTime value={row.original.last_used_at} /> : "Never",
  },
];

export function AppKeyListPage() {
  const search = listRoute.useSearch();
  const navigate = listRoute.useNavigate();
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
  });
  const query = useAppKeys(tableSearch);
  const canEdit = useCan({ resource: "app_keys", access: "edit" });
  return (
    <PageShell>
      <PageHeader
        title="App Keys"
        description="Pair companion devices and choose the locations they can use."
        actions={
          canEdit ? (
            <Button size="sm" nativeButton={false} render={<Link to="/app-keys/new" />}>
              <Plus data-icon="inline-start" />
              Create
            </Button>
          ) : null
        }
      />
      <ResourceDataTable
        data={query.data?.items ?? []}
        count={query.data?.count ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={query.isLoading}
        pending={query.isPlaceholderData}
        error={query.error}
        onRetry={() => void query.refetch()}
        icon={<KeyRound />}
        emptyTitle="App Keys"
        emptyDescription="Create an app key to pair a companion device."
      />
    </PageShell>
  );
}

export function AppKeyCreatePage() {
  const navigate = useNavigate();
  const [created, setCreated] = useState<AppKeyCreatedKey | null>(null);
  const create = useCreateAppKey(setCreated);
  return (
    <>
      <AppKeyForm
        onSave={async (body) => {
          const key = await create.mutateAsync(body);
          return key.id;
        }}
        onCancel={() => navigate({ to: "/app-keys" })}
      />
      {created ? (
        <CreatedKeyDialog
          created={created}
          onClose={() => {
            const id = created.id;
            setCreated(null);
            void navigate({ to: "/app-keys/$id", params: { id: String(id) } });
          }}
        />
      ) : null}
    </>
  );
}

export function AppKeyEditPage() {
  const { id: rawID } = useParams({ from: "/_authenticated/app-keys/$id/edit" });
  const id = parseRouteID(rawID);
  const query = useAppKey(id);
  const update = useUpdateAppKey(id ?? 0);
  const navigate = useNavigate();
  if (id === null)
    return (
      <PageShell>
        <QueryError error={{ message: "Invalid app key." }} />
      </PageShell>
    );
  if (query.isPending)
    return (
      <PageShell>
        <Skeleton className="h-80 w-full" />
      </PageShell>
    );
  if (!query.data)
    return (
      <PageShell>
        <QueryError error={query.error ?? { message: "Invalid app key." }} />
      </PageShell>
    );
  return (
    <AppKeyForm
      key={id}
      initial={query.data}
      onSave={async (body) => (await update.mutateAsync(body)).id}
      onSaved={(savedID) => navigate({ to: "/app-keys/$id", params: { id: String(savedID) } })}
      onCancel={() => navigate({ to: "/app-keys/$id", params: { id: String(id) } })}
    />
  );
}

function AppKeyForm({
  initial,
  onSave,
  onSaved,
  onCancel,
}: {
  initial?: AppKeyKey;
  onSave: (body: AppKeyMutation) => Promise<number>;
  onSaved?: (id: number) => Promise<void>;
  onCancel: () => Promise<void>;
}) {
  const [error, setError] = useState<Error | null>(null);
  const form = useForm({
    defaultValues: {
      name: initial?.name ?? "",
      all_locations: initial?.all_locations ?? false,
      locations: initial?.locations ?? [],
      expires_at: initial?.expires_at ? localDateTime(initial.expires_at) : "",
    },
    onSubmit: async ({ value, formApi }) => {
      setError(null);
      try {
        const id = await onSave({
          name: value.name.trim(),
          all_locations: value.all_locations,
          location_ids: value.all_locations ? [] : value.locations.map((location) => location.id),
          expires_at: value.expires_at ? new Date(value.expires_at).toISOString() : undefined,
        });
        formApi.reset(value);
        if (onSaved) await exit.runWithoutPrompt(() => onSaved(id));
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to save app key"));
      }
    },
  });
  const exit = usePageFormExitGuard({ form, onDiscard: onCancel });
  return (
    <PageShell>
      <PageHeader title={initial ? "Edit App Key" : "Create App Key"} />
      <FieldGroup className="max-w-3xl">
        <form.Field
          name="name"
          validators={{ onBlur: requiredString("Name"), onSubmit: requiredString("Name") }}
        >
          {(field) => (
            <ValidatedFormField field={field} label="Name" htmlFor="app-key-name" required>
              {(control) => (
                <Input
                  {...control}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              )}
            </ValidatedFormField>
          )}
        </form.Field>
        <form.Field name="expires_at">
          {(field) => (
            <Field>
              <FieldLabel htmlFor="app-key-expires">Expires</FieldLabel>
              <Input
                id="app-key-expires"
                type="datetime-local"
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
              />
              <FieldDescription>Leave blank for a key that does not expire.</FieldDescription>
            </Field>
          )}
        </form.Field>
        <form.Field name="all_locations">
          {(field) => (
            <Field orientation="horizontal">
              <Checkbox
                id="all-locations"
                checked={field.state.value}
                onCheckedChange={field.handleChange}
              />
              <FieldLabel htmlFor="all-locations">
                All locations, including locations created later
              </FieldLabel>
            </Field>
          )}
        </form.Field>
        <form.Subscribe selector={(state) => state.values.all_locations}>
          {(allLocations) =>
            allLocations ? null : (
              <form.Field name="locations">
                {(field) => (
                  <Field>
                    <FieldLabel htmlFor="app-key-locations">Locations</FieldLabel>
                    <LocationPicker
                      id="app-key-locations"
                      value={field.state.value}
                      onChange={field.handleChange}
                    />
                    <FieldDescription>
                      Devices can select any enabled location assigned to this key.
                    </FieldDescription>
                  </Field>
                )}
              </form.Field>
            )
          }
        </form.Subscribe>
        <QueryError error={error} />
        <FormActions
          form={form}
          submitLabel={initial ? "Save" : "Create"}
          onCancel={exit.requestDiscard}
        />
      </FieldGroup>
      {exit.dialog}
    </PageShell>
  );
}

export function AppKeyDetailPage() {
  const { id: rawID } = useParams({ from: "/_authenticated/app-keys/$id/" });
  const id = parseRouteID(rawID);
  return <AppKeyDetail key={rawID} id={id} />;
}
function AppKeyDetail({ id }: { id: number | null }) {
  const query = useAppKey(id);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const deletion = useDeleteAppKey();
  const navigate = useNavigate();
  const canEdit = useCan({ resource: "app_keys", access: "edit" });
  if (id === null)
    return (
      <PageShell>
        <QueryError error={{ message: "Invalid app key." }} />
      </PageShell>
    );
  if (query.isPending)
    return (
      <PageShell>
        <Skeleton className="h-80 w-full" />
      </PageShell>
    );
  if (!query.data)
    return (
      <PageShell>
        <QueryError error={query.error ?? { message: "Invalid app key." }} />
      </PageShell>
    );
  const key = query.data;
  return (
    <PageShell>
      <PageHeader
        title={key.name}
        actions={
          <>
            {canEdit ? (
              <>
                <Button
                  size="sm"
                  nativeButton={false}
                  variant="outline"
                  render={<Link to="/app-keys/$id/edit" params={{ id: String(key.id) }} />}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                <Button size="sm" variant="destructive" onClick={() => setConfirmDelete(true)}>
                  <Trash2 data-icon="inline-start" />
                  Delete
                </Button>
              </>
            ) : null}
          </>
        }
      />
      <KeyValueSection title="Overview">
        <KeyValueRow label="Prefix" value={key.key_prefix} />
        <KeyValueRow
          label="Locations"
          value={
            key.all_locations ? (
              "All locations"
            ) : (
              <TokenList values={key.locations.map((location) => location.name)} />
            )
          }
        />
        <KeyValueRow
          label="Expires"
          value={key.expires_at ? <RelativeTime value={key.expires_at} /> : "Never"}
        />
        <KeyValueRow
          label="Last Used"
          value={key.last_used_at ? <RelativeTime value={key.last_used_at} /> : "Never"}
        />
        <KeyValueRow label="Created" value={<RelativeTime value={key.created_at} />} />
        <KeyValueRow label="Updated" value={<RelativeTime value={key.updated_at} />} />
      </KeyValueSection>
      <QueryError error={deletion.error} />
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Delete App Key?"
        description="Paired devices using this key will no longer accept check-ins."
        confirmLabel="Delete"
        variant="destructive"
        pending={deletion.isPending}
        onConfirm={() => {
          void deletion
            .mutateAsync(key.id)
            .then(() => navigate({ to: "/app-keys" }))
            .catch(() => undefined);
        }}
      />
    </PageShell>
  );
}

function localDateTime(value: string) {
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

function CreatedKeyDialog({
  created,
  onClose,
}: {
  created: AppKeyCreatedKey;
  onClose: () => void;
}) {
  const secret = created.api_key;
  const pairing = pairingPayload(runtime.serverURL ?? globalThis.location.origin, secret);
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Save This Key</DialogTitle>
          <DialogDescription>
            This key is shown once. Scan the pairing QR code from the companion app, then select its
            location.
          </DialogDescription>
        </DialogHeader>
        <div className="flex gap-2">
          <Input aria-label="New app key" readOnly value={secret} className="font-mono" />
          <Button
            aria-label="Copy app key"
            variant="outline"
            onClick={() => {
              void navigator.clipboard
                .writeText(secret)
                .then(() => toast.add({ title: "Copied", type: "success" }))
                .catch(() => toast.add({ title: "Copy Failed", type: "error" }));
            }}
          >
            <Copy />
          </Button>
        </div>
        <div className="w-fit rounded-xl bg-white p-4">
          <QRCodeSVG value={pairing} size={240} title="Companion pairing QR code" />
        </div>
        {!created.all_locations && created.locations.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Assign a location before pairing a device.
          </p>
        ) : null}

        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
