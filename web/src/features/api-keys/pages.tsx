import { useForm } from "@tanstack/react-form";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "@tanstack/react-router";
import { Copy, KeyRound, Plus, QrCode, Trash2 } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { useEffect, useState } from "react";

import { AsyncButton } from "@components/async-button";
import { ConfirmDialog } from "@components/confirm-dialog";
import type { DataTableColumnDef } from "@components/data-table/types";
import { FormActions } from "@components/form-actions";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Alert, AlertDescription, AlertTitle } from "@components/ui/alert";
import { Button, buttonVariants } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { Skeleton } from "@components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import { toast } from "@components/ui/toast";
import { ValidatedFormField } from "@components/validated-form-field";
import { AccessEditor } from "@features/access/access-editor";
import { loadAccessLocations } from "@features/access/locations";
import { useCanAccess } from "@features/auth/access";
import { exportRows } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import {
  createApiKey,
  deleteApiKey,
  getApiKey,
  listApiKeys,
  patchApiKey,
  unwrap,
  type ApiKey,
} from "@lib/api";
import { formatDateTime } from "@lib/utils";

import { appKeyGrants, pairingPayload } from "./pairing";

const createdSecrets = new Map<string, string>();

const columns: DataTableColumnDef<ApiKey>[] = [
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink to="/api-keys/$id/show" params={{ id: row.original.id }}>
        {row.original.name || "Untitled"}
      </TextLink>
    ),
  },
  { id: "key_prefix", accessorKey: "key_prefix", header: "Prefix" },
  {
    id: "last_used_at",
    accessorKey: "last_used_at",
    header: "Last used",
    cell: ({ row }) => formatDateTime(row.original.last_used_at),
  },
  {
    id: "expires_at",
    accessorKey: "expires_at",
    header: "Expires",
    cell: ({ row }) => formatDateTime(row.original.expires_at),
  },
  {
    id: "created_at",
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => formatDateTime(row.original.created_at),
  },
];

export function APIKeysList() {
  const { tableSearch, params } = useResourceTable("created_at.desc");
  const queryClient = useQueryClient();
  const keys = useQuery({
    queryKey: ["api-keys", params],
    queryFn: ({ signal }) => unwrap(listApiKeys({ query: params, signal })),
    placeholderData: keepPreviousData,
  });
  const canCreate = useCanAccess("api-keys", "create");
  const canDelete = useCanAccess("api-keys", "delete");
  return (
    <PageShell>
      <PageHeader
        title="API keys"
        actions={
          canCreate ? (
            <Link to="/api-keys/create" className={buttonVariants({ size: "sm" })}>
              <Plus data-icon="inline-start" />
              Create
            </Link>
          ) : null
        }
      />
      <ResourceDataTable
        data={keys.data?.rows ?? []}
        count={keys.data?.total ?? 0}
        columns={columns}
        tableSearch={tableSearch}
        loading={keys.isLoading}
        pending={keys.isPlaceholderData}
        error={keys.error}
        onRetry={() => void keys.refetch()}
        icon={<KeyRound />}
        emptyTitle="API keys"
        emptyDescription="Create an API key to connect an app or integration."
        getRowId={(row) => row.id}
        onDeleteRows={
          canDelete
            ? async (rows) => {
                try {
                  for (const row of rows) await unwrap(deleteApiKey({ path: { id: row.id } }));
                } finally {
                  await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
                }
              }
            : undefined
        }
        exportOptions={{
          filename: "api-keys",
          columns: [
            { header: "id", value: (row) => row.id },
            { header: "name", value: (row) => row.name },
            { header: "key_prefix", value: (row) => row.key_prefix },
            { header: "last_used_at", value: (row) => row.last_used_at },
            { header: "expires_at", value: (row) => row.expires_at },
            { header: "admin", value: (row) => row.admin },
            { header: "access", value: (row) => JSON.stringify(row.access) },
            { header: "created_at", value: (row) => row.created_at },
          ],
          loadRows: () =>
            exportRows((offset) =>
              unwrap(listApiKeys({ query: { ...params, offset, limit: 250 } })),
            ),
        }}
      />
    </PageShell>
  );
}

export function APIKeyCreate() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [error, setError] = useState<Error | null>(null);
  const form = useForm({
    defaultValues: { name: "" },
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        const record = await unwrap(createApiKey({ body: value }));
        createdSecrets.set(record.id, record.secret);
        await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
        await navigate({ to: "/api-keys/$id/show", params: { id: record.id } });
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Could not create API key"));
      }
    },
  });
  return (
    <PageShell>
      <PageHeader title="Create API key" actions={<APIKeyListLink />} />
      <form
        className="flex max-w-2xl flex-col gap-6"
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <FieldGroup>
          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Name" htmlFor="api-key-name">
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
        </FieldGroup>
        <QueryError title="Could not create API key" error={error} />
        <FormActions
          nativeSubmit
          form={form}
          submitLabel="Create"
          onCancel={() => void navigate({ to: "/api-keys" })}
        />
      </form>
    </PageShell>
  );
}

export function APIKeyShow({ id }: { id: string }) {
  return <APIKeyDetail key={id} id={id} />;
}

function APIKeyDetail({ id }: { id: string }) {
  const navigate = useNavigate();
  const params = useParams({ strict: false }) as { tab?: string };
  const [secret] = useState(() => createdSecrets.get(id));
  useEffect(() => {
    createdSecrets.delete(id);
  }, [id]);
  const key = useQuery({
    queryKey: ["api-keys", id],
    queryFn: ({ signal }) => unwrap(getApiKey({ path: { id }, signal })),
  });
  const canWrite = useCanAccess("api-keys", "write");
  if (key.error)
    return (
      <PageShell>
        <QueryError
          title="Could not load API key"
          error={key.error}
          onRetry={() => void key.refetch()}
        />
      </PageShell>
    );
  if (!key.data)
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-72 w-full" />
      </PageShell>
    );
  const record = key.data;
  return (
    <PageShell>
      <PageHeader
        title={record.name || "API key"}
        actions={
          <>
            <APIKeyListLink />
            <APIKeyDelete record={record} />
          </>
        }
      />
      <Tabs
        value={canWrite && params.tab === "1" ? "access" : "overview"}
        onValueChange={(value) => {
          void navigate({
            to: value === "access" ? "/api-keys/$id/show/$tab" : "/api-keys/$id/show",
            params: { id, tab: "1" },
            search: {},
          });
        }}
      >
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          {canWrite ? <TabsTrigger value="access">Access</TabsTrigger> : null}
        </TabsList>
        <TabsContent value="overview" className="flex flex-col gap-6 pt-4">
          {secret ? (
            <Alert>
              <AlertTitle>API key created</AlertTitle>
              <AlertDescription>
                Copy the secret now. It will not be shown again after leaving this page.
              </AlertDescription>
            </Alert>
          ) : null}
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={record.name} />
            <KeyValueRow label="Prefix" value={record.key_prefix} />
            {secret ? (
              <KeyValueRow
                label="Secret"
                value={
                  <div className="flex flex-wrap items-center gap-3">
                    <code className="break-all">{secret}</code>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        void navigator.clipboard.writeText(secret).then(
                          () => toast.add({ title: "Secret copied", type: "success" }),
                          () => toast.add({ title: "Could not copy secret", type: "error" }),
                        );
                      }}
                    >
                      <Copy data-icon="inline-start" />
                      Copy
                    </Button>
                    {canWrite ? <PairingQR id={id} secret={secret} /> : null}
                  </div>
                }
              />
            ) : null}
            <KeyValueRow label="Last used" value={formatDateTime(record.last_used_at)} />
            <KeyValueRow label="Expires" value={formatDateTime(record.expires_at)} />
            <KeyValueRow label="Created" value={formatDateTime(record.created_at)} />
          </KeyValueSection>
        </TabsContent>
        {canWrite ? (
          <TabsContent value="access" className="pt-4">
            <AccessEditor resource="api-keys" record={record} />
          </TabsContent>
        ) : null}
      </Tabs>
    </PageShell>
  );
}

function APIKeyListLink() {
  return (
    <Link to="/api-keys" className={buttonVariants({ variant: "outline", size: "sm" })}>
      List
    </Link>
  );
}

function PairingQR({ id, secret }: { id: string; secret: string }) {
  const [open, setOpen] = useState(false);
  const [qrOpen, setQROpen] = useState(false);
  const [locationId, setLocationId] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const canReadLocations = useCanAccess("locations", "read");
  const locations = useQuery({
    queryKey: ["locations", "pairing"],
    queryFn: ({ signal }) => loadAccessLocations(signal, true),
    enabled: open && canReadLocations,
  });
  const apply = useMutation({
    mutationFn: () =>
      unwrap(
        patchApiKey({ path: { id }, body: { admin: false, access: appKeyGrants(locationId!) } }),
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setOpen(false);
      setQROpen(true);
    },
  });
  const items = (locations.data ?? []).map((location) => ({
    value: location.id,
    label: location.name,
  }));
  return (
    <>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <QrCode data-icon="inline-start" />
        Show pairing QR
      </Button>
      <Dialog
        open={open}
        onOpenChange={(next) => {
          if (!apply.isPending) setOpen(next);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Apply app permissions</DialogTitle>
            <DialogDescription>
              This sets the key’s permissions to the minimum required by the app: read locations,
              read users, read reusable assets, and create check-ins for the selected location.
            </DialogDescription>
          </DialogHeader>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="pairing-location">Location</FieldLabel>
              <Select
                items={items}
                value={locationId}
                onValueChange={setLocationId}
                disabled={locations.isLoading || apply.isPending || !canReadLocations}
              >
                <SelectTrigger id="pairing-location" className="w-full">
                  <SelectValue
                    placeholder={locations.isLoading ? "Loading locations…" : "Select location"}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {items.map((location) => (
                      <SelectItem key={location.value} value={location.value}>
                        {location.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </FieldGroup>
          {!canReadLocations ? (
            <Alert>
              <AlertDescription>
                Location read access is required to select a pairing location.
              </AlertDescription>
            </Alert>
          ) : null}
          {locations.isSuccess && items.length === 0 ? (
            <Alert>
              <AlertDescription>No enabled locations are available.</AlertDescription>
            </Alert>
          ) : null}
          <QueryError
            title="Could not load locations"
            error={locations.error}
            onRetry={() => void locations.refetch()}
          />
          <QueryError title="Could not apply app permissions" error={apply.error} />
          <DialogFooter>
            <Button variant="outline" disabled={apply.isPending} onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <AsyncButton
              isPending={apply.isPending}
              disabled={!locationId || !canReadLocations}
              onClick={() => apply.mutate()}
            >
              Apply & show QR
            </AsyncButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={qrOpen} onOpenChange={setQROpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Pairing QR</DialogTitle>
            <DialogDescription>
              Scan this code in the companion app to connect it.
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-center">
            <QRCodeSVG
              value={pairingPayload(secret, globalThis.location.origin)}
              size={280}
              marginSize={4}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setQROpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function APIKeyDelete({ record }: { record: ApiKey }) {
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const canDelete = useCanAccess("api-keys", "delete");
  const remove = useMutation({
    mutationFn: () => unwrap(deleteApiKey({ path: { id: record.id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      toast.add({ title: "API key deleted", type: "success" });
      await navigate({ to: "/api-keys" });
    },
    onError: (error) => {
      toast.add({ title: "Could not delete API key", description: error.message, type: "error" });
    },
  });
  if (!canDelete) return null;
  return (
    <>
      <Button size="sm" variant="destructive" onClick={() => setOpen(true)}>
        <Trash2 data-icon="inline-start" />
        Delete
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title="Delete API key?"
        description={`Apps using ${record.name || "this key"} will no longer be able to connect.`}
        confirmLabel="Delete"
        variant="destructive"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </>
  );
}
