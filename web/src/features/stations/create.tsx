import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { useCreateStation } from "@features/resources/queries";
import { StationForm } from "@features/stations/fields";
import { StationPairingDialog } from "@features/stations/pairing-dialog";
import type { StationPairing } from "@lib/api";

export function StationCreatePage() {
  const navigate = useNavigate();
  const [created, setCreated] = useState<StationPairing>();
  const create = useCreateStation(setCreated);
  return (
    <>
      <StationForm
        title="Create Station"
        onSubmit={async (body) => {
          const result = await create.mutateAsync(body);
          return result.id;
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
