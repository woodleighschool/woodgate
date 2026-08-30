import { useParams } from "@tanstack/react-router";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryGate } from "@components/query-gate";
import { Badge } from "@components/ui/badge";
import { useCheckin } from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";
import { nonEmpty } from "@lib/utils";

export function CheckinDetailPage() {
  const { id: rawID } = useParams({ from: "/_authenticated/checkins/$id" });
  const id = parseRouteID(rawID);
  const query = useCheckin(id);
  if (id === null)
    return <QueryGate title="Failed to Load Check-in" error={{ message: "Invalid check-in." }} />;
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Check-in"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }
  const checkin = query.data;
  return (
    <PageShell>
      <PageHeader title="Check-in Details" />
      {checkin.photo_url ? (
        <div className="flex max-h-128 max-w-3xl items-center justify-center overflow-hidden rounded-xl border bg-muted/40 p-4">
          <img
            src={checkin.photo_url}
            alt="Check-in attachment"
            className="max-h-120 max-w-full object-contain"
          />
        </div>
      ) : null}
      <KeyValueSection title="Record">
        <KeyValueRow label="Person" value={checkin.user_name || `User #${checkin.user_id}`} />
        <KeyValueRow label="Department" value={nonEmpty(checkin.department) ?? "-"} />
        <KeyValueRow label="Location" value={checkin.location_name || "-"} />
        <KeyValueRow
          label="Direction"
          value={
            <Badge variant="secondary">
              {checkin.direction === "check_in" ? "Check in" : "Check out"}
            </Badge>
          }
        />
        <KeyValueRow label="Time" value={new Date(checkin.created_at).toLocaleString()} />
        <KeyValueRow label="Notes" value={nonEmpty(checkin.notes) ?? "-"} />
      </KeyValueSection>
    </PageShell>
  );
}
