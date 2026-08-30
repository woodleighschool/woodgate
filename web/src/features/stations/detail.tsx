import { useNavigate, useParams } from "@tanstack/react-router";
import { KeyRound, Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { AsyncButton } from "@components/async-button";
import { BooleanIndicator } from "@components/boolean-indicator";
import { ConfirmDialog } from "@components/confirm-dialog";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Input } from "@components/ui/input";
import { useAccount } from "@features/account/queries";
import { canAccess } from "@features/authz/permissions";
import { useDeleteStation, useRotateStationKey, useStation } from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";

export function StationDetailPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/stations/$id" });
  const id = parseRouteID(rawID);
  const query = useStation(id);
  const rotate = useRotateStationKey();
  const remove = useDeleteStation();
  const account = useAccount();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null)
    return <QueryGate title="Failed to Load Station" error={{ message: "Invalid station." }} />;
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Station"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const station = query.data;
  const key = rotate.data?.secret;
  return (
    <>
      <PageShell>
        <PageHeader
          title={station.name}
          description="A Station represents one dedicated check-in device."
          actions={
            canAccess(account.data, "stations", "edit") ? (
              <>
                <Button
                  size="sm"
                  variant="outline"
                  render={<Link to="/stations/$id/edit" params={{ id: String(station.id) }} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                <AsyncButton
                  size="sm"
                  icon={<KeyRound data-icon="inline-start" />}
                  isPending={rotate.isPending}
                  onClick={() => setConfirmOpen(true)}
                >
                  Rotate Key
                </AsyncButton>
                <Button size="sm" variant="destructive" onClick={() => setDeleteOpen(true)}>
                  <Trash2 data-icon="inline-start" />
                  Delete
                </Button>
              </>
            ) : null
          }
        />

        {key ? (
          <section className="flex max-w-3xl flex-col gap-2">
            <h2 className="text-base font-semibold">New Station Key</h2>
            <p className="text-sm text-muted-foreground">
              Copy this key now. It will not be shown again.
            </p>
            <Input value={key} readOnly className="font-mono" />
          </section>
        ) : null}

        <KeyValueSection title="Station">
          <KeyValueRow label="Location" value={`Location #${station.location_id}`} />
          <KeyValueRow
            label="Enabled"
            value={<BooleanIndicator value={station.enabled ?? true} />}
          />
          <KeyValueRow label="Key Prefix" value={station.secret_prefix ?? "-"} />
          <KeyValueRow
            label="Last Used"
            value={station.last_seen_at ? new Date(station.last_seen_at).toLocaleString() : "Never"}
          />
        </KeyValueSection>
      </PageShell>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Rotate Station Key?"
        description="The current key stops working immediately."
        confirmLabel="Rotate"
        pending={rotate.isPending}
        onConfirm={() => {
          void rotate.mutateAsync(id).then(() => setConfirmOpen(false));
        }}
      />
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Station?"
        description="The device will disconnect and its key will stop working."
        confirmLabel="Delete"
        pending={remove.isPending}
        onConfirm={() => {
          void remove.mutateAsync(station.id).then(() => navigate({ to: "/stations" }));
        }}
      />
    </>
  );
}
