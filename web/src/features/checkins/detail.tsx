import { useParams } from "@tanstack/react-router";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryGate } from "@components/query-gate";
import { RelativeTime } from "@components/relative-time";
import { Badge } from "@components/ui/badge";
import { checkinPersonLabel } from "@features/checkins/presentation";
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
        <KeyValueRow label="Person" value={checkinPersonLabel(checkin.person)} />
        <KeyValueRow label="Email" value={checkin.person.email} />
        <KeyValueRow label="Department" value={nonEmpty(checkin.person.department) ?? "-"} />
        <KeyValueRow label="Location" value={checkin.location.name} />
        <KeyValueRow
          label="Direction"
          value={
            <Badge variant="secondary">
              {checkin.direction === "check_in" ? "Check in" : "Check out"}
            </Badge>
          }
        />
        <KeyValueRow label="Time" value={<RelativeTime value={checkin.created_at} />} />
        <KeyValueRow label="Notes" value={nonEmpty(checkin.notes) ?? "-"} />
        {checkin.station ? <KeyValueRow label="Station" value={checkin.station.name} /> : null}
        {checkin.created_by ? (
          <KeyValueRow label="Recorded By" value={checkinPersonLabel(checkin.created_by)} />
        ) : null}
      </KeyValueSection>
    </PageShell>
  );
}
