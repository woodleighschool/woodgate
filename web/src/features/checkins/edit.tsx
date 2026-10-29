import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { CheckinForm } from "@features/checkins/fields";
import { useCheckin, useUpdateCheckin } from "@features/resources/queries";

export function CheckinEditPage() {
  const navigate = useNavigate();
  const { id } = useParams({ from: "/_authenticated/checkins/$id/edit" });
  const query = useCheckin(id);
  const update = useUpdateCheckin(id);
  if (!query.data) {
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
    <CheckinForm
      key={checkin.id}
      title="Edit Check-in"
      initial={checkin}
      onSubmit={async (body, occurredAt) =>
        (await update.mutateAsync({ ...body, occurred_at: occurredAt ?? checkin.occurred_at })).id
      }
      onSuccess={(savedID) => void navigate({ to: "/checkins/$id", params: { id: savedID } })}
      onCancel={() => void navigate({ to: "/checkins/$id", params: { id } })}
    />
  );
}
