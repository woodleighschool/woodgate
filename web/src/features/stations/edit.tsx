import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { useStation, useUpdateStation } from "@features/resources/queries";
import { StationForm } from "@features/stations/fields";
import { parseRouteID } from "@lib/route-params";

export function StationEditPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/stations/$id/edit" });
  const id = parseRouteID(rawID);
  const query = useStation(id);
  const update = useUpdateStation(id ?? 0);
  if (id === null || query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Station"
        error={query.error ?? { message: "Invalid Station." }}
      />
    );
  }
  return (
    <StationForm
      title="Edit Station"
      initial={query.data}
      onSubmit={async (body) => (await update.mutateAsync(body)).id}
      onSuccess={(savedID) =>
        void navigate({ to: "/stations/$id", params: { id: String(savedID) } })
      }
      onCancel={() => void navigate({ to: "/stations/$id", params: { id: String(id) } })}
    />
  );
}
