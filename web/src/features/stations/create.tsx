import { useNavigate } from "@tanstack/react-router";
import { QRCodeSVG } from "qrcode.react";
import { useState } from "react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { Button } from "@components/ui/button";
import { Input } from "@components/ui/input";
import { useCreateStation } from "@features/resources/queries";
import { StationForm } from "@features/stations/fields";
import type { StationKey } from "@lib/api";
import { runtime } from "@lib/runtime";

export function StationCreatePage() {
  const navigate = useNavigate();
  const create = useCreateStation();
  const [created, setCreated] = useState<StationKey>();
  if (created?.station && created.secret) {
    const pairing = JSON.stringify({
      base_url: runtime.serverURL ?? window.location.origin,
      station_secret: created.secret,
    });
    return (
      <PageShell>
        <PageHeader
          title="Pair Station"
          description="Scan this now. The Station secret is not shown again."
        />
        <div className="flex max-w-3xl flex-col items-start gap-4 rounded-xl border p-6">
          <QRCodeSVG value={pairing} size={280} level="M" marginSize={2} />
          <Input value={created.secret} readOnly className="font-mono" />
          <Button
            render={<Link to="/stations/$id" params={{ id: String(created.station.id) }} />}
            nativeButton={false}
          >
            Done
          </Button>
        </div>
      </PageShell>
    );
  }
  return (
    <StationForm
      title="Create Station"
      onSubmit={async (body) => {
        const result = await create.mutateAsync(body);
        setCreated(result);
        return result.station?.id ?? 0;
      }}
      onSuccess={() => undefined}
      onCancel={() => void navigate({ to: "/stations" })}
    />
  );
}
