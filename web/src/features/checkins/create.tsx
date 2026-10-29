import { useNavigate } from "@tanstack/react-router";

import { CheckinForm } from "@features/checkins/fields";
import { useCreateCheckin } from "@features/resources/queries";

export function CheckinCreatePage() {
  const navigate = useNavigate();
  const create = useCreateCheckin();
  return (
    <CheckinForm
      title="Create Check-in"
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => void navigate({ to: "/checkins/$id", params: { id } })}
      onCancel={() => void navigate({ to: "/checkins" })}
    />
  );
}
