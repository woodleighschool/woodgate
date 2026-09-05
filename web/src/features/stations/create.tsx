import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { useCreateStation } from "@features/resources/queries";
import { StationForm } from "@features/stations/fields";
import { StationPairingDialog } from "@features/stations/pairing-dialog";
import type { StationPairing } from "@lib/api";

export function StationCreatePage() {
  const navigate = useNavigate();
  const create = useCreateStation();
  const [created, setCreated] = useState<StationPairing>();
  return (
    <>
      <StationForm
        title="Create Station"
        onSubmit={async (body) => {
          const result = await create.mutateAsync(body);
          setCreated(result);
          return result.station.id;
        }}
        onSuccess={() => undefined}
        onCancel={() => void navigate({ to: "/stations" })}
      />
      <StationPairingDialog
        pairing={created}
        onDone={() => {
          if (created) {
            void navigate({ to: "/stations/$id", params: { id: String(created.station.id) } });
          }
        }}
      />
    </>
  );
}
