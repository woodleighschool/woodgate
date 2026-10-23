import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { useStation, useUpdateStation } from "@features/resources/queries";
import { StationForm } from "@features/stations/fields";

export function StationEditPage() {
  const navigate = useNavigate();
  const { id } = useParams({ from: "/_authenticated/stations/$id/edit" });
  const query = useStation(id);
  const update = useUpdateStation(id);
  if (!query.data) {
    return <QueryGate title="Failed to Load Station" error={query.error} />;
  }
  return (
    <StationForm
      title="Edit Station"
      initial={query.data}
      onSubmit={async (body) => (await update.mutateAsync(body)).id}
      onSuccess={(savedID) => void navigate({ to: "/stations/$id", params: { id: savedID } })}
      onCancel={() => void navigate({ to: "/stations/$id", params: { id: id } })}
    />
  );
}
