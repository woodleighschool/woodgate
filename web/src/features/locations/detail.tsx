import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { ConfirmDialog } from "@components/confirm-dialog";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { useAccount } from "@features/account/queries";
import { canAccess } from "@features/authz/permissions";
import { useDeleteLocation, useLocation } from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";
import { nonEmpty } from "@lib/utils";

export function LocationDetailPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/locations/$id" });
  const id = parseRouteID(rawID);
  const query = useLocation(id);
  const account = useAccount();
  const remove = useDeleteLocation();
  const [confirmOpen, setConfirmOpen] = useState(false);
  if (id === null)
    return <QueryGate title="Failed to Load Location" error={{ message: "Invalid location." }} />;
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Location"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  const location = query.data;
  return (
    <PageShell>
      <PageHeader
        title={location.name}
        description={nonEmpty(location.description)}
        actions={
          canAccess(account.data, "locations", "edit") ? (
            <>
              <Button
                size="sm"
                variant="outline"
                render={<Link to="/locations/$id/edit" params={{ id: String(location.id) }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button size="sm" variant="destructive" onClick={() => setConfirmOpen(true)}>
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />
      {location.background_url || location.logo_url ? (
        <section className="space-y-3">
          <h2 className="text-sm font-medium">Artwork</h2>
          <div className="grid max-w-3xl gap-4 sm:grid-cols-2">
            {location.background_url ? (
              <ArtworkPreview label="Background" src={location.background_url} fit="cover" />
            ) : null}
            {location.logo_url ? (
              <ArtworkPreview label="Logo" src={location.logo_url} fit="contain" />
            ) : null}
          </div>
        </section>
      ) : null}
      <KeyValueSection title="Check-in Settings">
        <KeyValueRow
          label="Enabled"
          value={<BooleanIndicator value={location.enabled ?? true} />}
        />
        <KeyValueRow label="Notes" value={<BooleanIndicator value={location.notes ?? false} />} />
        <KeyValueRow label="Photos" value={<BooleanIndicator value={location.photo ?? false} />} />
        <KeyValueRow label="Directory Groups" value={String(location.group_ids?.length ?? 0)} />
      </KeyValueSection>
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Delete Location?"
        description="The location can only be deleted when no Stations or check-ins reference it."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => {
          void remove.mutateAsync(location.id).then(() => navigate({ to: "/locations" }));
        }}
      />
    </PageShell>
  );
}

function ArtworkPreview({
  label,
  src,
  fit,
}: {
  label: string;
  src: string;
  fit: "cover" | "contain";
}) {
  return (
    <figure className="overflow-hidden rounded-xl border bg-muted/40">
      <div className="flex aspect-video items-center justify-center p-4">
        <img
          src={src}
          alt=""
          className={
            fit === "cover" ? "size-full object-cover" : "max-h-full max-w-full object-contain"
          }
        />
      </div>
      <figcaption className="border-t px-3 py-2 text-sm text-muted-foreground">{label}</figcaption>
    </figure>
  );
}
