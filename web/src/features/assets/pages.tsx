import { useForm } from "@tanstack/react-form";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Image, Plus, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { DataTableColumnDef } from "@components/data-table/types";
import { FacetedFilter } from "@components/faceted-filter";
import { FormActions } from "@components/form-actions";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryError } from "@components/query-error";
import { ResourceDataTable } from "@components/resource-data-table";
import { Button, buttonVariants } from "@components/ui/button";
import { FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Skeleton } from "@components/ui/skeleton";
import { toast } from "@components/ui/toast";
import { ValidatedFormField } from "@components/validated-form-field";
import { useCanAccess } from "@features/auth/access";
import { exportRows } from "@features/resources/shared";
import { useResourceTable } from "@features/resources/table";
import {
  createAsset,
  deleteAsset,
  getAsset,
  listAssets,
  patchAsset,
  unwrap,
  type Asset,
  type ListAssetsData,
} from "@lib/api";
import { formatDateTime } from "@lib/utils";

const columns: DataTableColumnDef<Asset>[] = [
  {
    id: "preview",
    header: "Preview",
    enableSorting: false,
    size: 80,
    cell: ({ row }) => (
      <img
        src={row.original.url}
        alt={row.original.name ?? "Asset preview"}
        className="size-14 rounded-md object-cover"
        loading="lazy"
      />
    ),
  },
  {
    id: "name",
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <TextLink to="/assets/$id/show" params={{ id: row.original.id }}>
        {row.original.name || "Untitled"}
      </TextLink>
    ),
  },
  { id: "type", accessorKey: "type", header: "Type" },
  {
    id: "url",
    accessorKey: "url",
    header: "URL",
    enableSorting: false,
    cell: ({ row }) => (
      <a href={row.original.url} target="_blank" rel="noreferrer" data-text-link>
        {row.original.url}
      </a>
    ),
  },
  {
    id: "updated_at",
    accessorKey: "updated_at",
    header: "Updated",
    cell: ({ row }) => formatDateTime(row.original.updated_at),
  },
];

export function AssetsList() {
  const { tableSearch, params, search, setFilter } = useResourceTable("name.asc");
  const queryClient = useQueryClient();
  const type = search.type === "asset" || search.type === "photo" ? search.type : undefined;
  const query = { ...params, type } satisfies NonNullable<ListAssetsData["query"]>;
  const assets = useQuery({
    queryKey: ["assets", query],
    queryFn: ({ signal }) => unwrap(listAssets({ query, signal })),
    placeholderData: keepPreviousData,
  });
  const canCreate = useCanAccess("assets", "create");
  const canDelete = useCanAccess("assets", "delete");
  return (
    <PageShell>
      <PageHeader
        title="Assets"
        actions={
          canCreate ? (
            <Link to="/assets/create" className={buttonVariants({ size: "sm" })}>
              <Plus data-icon="inline-start" />
              Create
            </Link>
          ) : null
        }
      />
      <ResourceDataTable
        data={assets.data?.rows ?? []}
        count={assets.data?.total ?? 0}
        columns={columns}
        tableSearch={{ ...tableSearch, isFiltered: tableSearch.isFiltered || Boolean(type) }}
        loading={assets.isLoading}
        pending={assets.isPlaceholderData}
        error={assets.error}
        onRetry={() => void assets.refetch()}
        icon={<Image />}
        emptyTitle="Assets"
        emptyDescription="Upload images to use for location branding."
        getRowId={(row) => row.id}
        onDeleteRows={
          canDelete
            ? async (rows) => {
                try {
                  for (const row of rows) await unwrap(deleteAsset({ path: { id: row.id } }));
                } finally {
                  await queryClient.invalidateQueries({ queryKey: ["assets"] });
                }
              }
            : undefined
        }
        exportOptions={{
          filename: "assets",
          columns: [
            { header: "id", value: (row) => row.id },
            { header: "name", value: (row) => row.name },
            { header: "type", value: (row) => row.type },
            { header: "url", value: (row) => row.url },
            { header: "created_at", value: (row) => row.created_at },
            { header: "updated_at", value: (row) => row.updated_at },
          ],
          loadRows: () =>
            exportRows((offset) => unwrap(listAssets({ query: { ...query, offset, limit: 250 } }))),
        }}
        filters={
          <FacetedFilter
            title="Type"
            options={[
              { value: "asset", label: "Asset" },
              { value: "photo", label: "Photo" },
            ]}
            value={type ? [type] : []}
            onValueChange={(values) => setFilter("type", values[0])}
            multiple={false}
          />
        }
      />
    </PageShell>
  );
}

export function AssetCreate() {
  return (
    <PageShell>
      <PageHeader title="Create asset" actions={<AssetListLink />} />
      <AssetForm />
    </PageShell>
  );
}

export function AssetEdit({ id }: { id: string }) {
  return <AssetDetail id={id} edit />;
}

export function AssetShow({ id }: { id: string }) {
  return <AssetDetail id={id} />;
}

function AssetDetail({ id, edit = false }: { id: string; edit?: boolean }) {
  const asset = useQuery({
    queryKey: ["assets", id],
    queryFn: ({ signal }) => unwrap(getAsset({ path: { id }, signal })),
  });
  const canWrite = useCanAccess("assets", "write");
  if (asset.error)
    return (
      <PageShell>
        <QueryError
          title="Could not load asset"
          error={asset.error}
          onRetry={() => void asset.refetch()}
        />
      </PageShell>
    );
  if (!asset.data)
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-72 w-full" />
      </PageShell>
    );
  const record = asset.data;
  const editable = record.type === "asset" && canWrite;
  return (
    <PageShell>
      <PageHeader
        title={record.name || (record.type === "photo" ? "Photo" : "Asset")}
        actions={
          <>
            <AssetListLink />
            {editable && !edit ? (
              <Link
                to="/assets/$id"
                params={{ id }}
                className={buttonVariants({ variant: "outline", size: "sm" })}
              >
                Edit
              </Link>
            ) : null}
            <AssetDelete asset={record} />
          </>
        }
      />
      {edit && editable ? (
        <AssetForm key={`${id}:${record.updated_at}`} asset={record} />
      ) : (
        <>
          <AssetImage asset={record} />
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={record.name} />
            <KeyValueRow label="Type" value={record.type} />
            <KeyValueRow
              label="URL"
              value={
                <a href={record.url} target="_blank" rel="noreferrer" data-text-link>
                  {record.url}
                </a>
              }
            />
            <KeyValueRow label="Created" value={formatDateTime(record.created_at)} />
            <KeyValueRow label="Updated" value={formatDateTime(record.updated_at)} />
          </KeyValueSection>
        </>
      )}
    </PageShell>
  );
}

function AssetListLink() {
  return (
    <Link to="/assets" className={buttonVariants({ variant: "outline", size: "sm" })}>
      List
    </Link>
  );
}

function AssetImage({ asset }: { asset: Asset }) {
  return (
    <a href={asset.url} target="_blank" rel="noreferrer" className="w-fit">
      <img
        src={asset.url}
        alt={asset.name || (asset.type === "photo" ? "Check-in photo" : "Asset")}
        className="max-h-96 max-w-full rounded-lg object-contain"
      />
    </a>
  );
}

function AssetForm({ asset }: { asset?: Asset }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [error, setError] = useState<Error | null>(null);
  const [preview, setPreview] = useState<string>();
  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview);
    };
  }, [preview]);
  const form = useForm({
    defaultValues: { name: asset?.name ?? "", file: null as File | null },
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        const record = asset
          ? await unwrap(
              patchAsset({
                path: { id: asset.id },
                body: { name: value.name, ...(value.file ? { file: value.file } : {}) },
              }),
            )
          : await unwrap(createAsset({ body: { name: value.name, file: value.file! } }));
        await queryClient.invalidateQueries({ queryKey: ["assets"] });
        toast.add({ title: asset ? "Asset updated" : "Asset created", type: "success" });
        await navigate({ to: "/assets/$id", params: { id: record.id } });
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Could not save asset"));
      }
    },
  });
  return (
    <form
      className="flex max-w-2xl flex-col gap-6"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      {asset ? <AssetImage asset={asset} /> : null}
      <FieldGroup>
        <form.Field name="name">
          {(field) => (
            <ValidatedFormField
              field={field}
              label="Name"
              htmlFor="asset-name"
              description="Optional cosmetic name"
            >
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
        <form.Field
          name="file"
          validators={{
            onChange: ({ value }) => (!asset && !value ? "Choose an image file" : undefined),
          }}
        >
          {(field) => (
            <ValidatedFormField
              field={field}
              label={asset ? "Replace asset file" : "Asset file"}
              htmlFor="asset-file"
              required={!asset}
              description="PNG or JPEG image"
            >
              {(control) => (
                <Input
                  {...control}
                  type="file"
                  accept="image/png,image/jpeg"
                  onBlur={field.handleBlur}
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    field.handleChange(file ?? null);
                    setPreview(file ? URL.createObjectURL(file) : undefined);
                  }}
                />
              )}
            </ValidatedFormField>
          )}
        </form.Field>
      </FieldGroup>
      {preview ? (
        <img
          src={preview}
          alt="Selected upload"
          className="max-h-72 max-w-full rounded-lg object-contain"
        />
      ) : null}
      <QueryError title="Could not save asset" error={error} />
      <FormActions
        nativeSubmit
        form={form}
        submitLabel={asset ? "Save" : "Create"}
        onCancel={() => void navigate({ to: "/assets" })}
      />
    </form>
  );
}

function AssetDelete({ asset }: { asset: Asset }) {
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const canDelete = useCanAccess("assets", "delete");
  const remove = useMutation({
    mutationFn: () => unwrap(deleteAsset({ path: { id: asset.id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["assets"] });
      toast.add({ title: "Asset deleted", type: "success" });
      await navigate({ to: "/assets" });
    },
    onError: (error) => {
      toast.add({ title: "Could not delete asset", description: error.message, type: "error" });
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
        title="Delete asset?"
        description={`Delete ${asset.name || "this image"}?`}
        confirmLabel="Delete"
        variant="destructive"
        pending={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </>
  );
}
