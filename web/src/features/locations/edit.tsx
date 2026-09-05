import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { LocationForm, uploadLocationImages } from "@features/locations/fields";
import {
  useLocation,
  useUpdateLocation,
  useUploadLocationBackground,
  useUploadLocationLogo,
} from "@features/resources/queries";
import { parseRouteID } from "@lib/route-params";

export function LocationEditPage() {
  const navigate = useNavigate();
  const { id: rawID } = useParams({ from: "/_authenticated/locations/$id/edit" });
  const id = parseRouteID(rawID);
  const query = useLocation(id);
  const update = useUpdateLocation(id ?? 0);
  const backgroundUpload = useUploadLocationBackground();
  const logoUpload = useUploadLocationLogo();
  if (id === null || query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Location"
        error={query.error ?? { message: "Invalid location." }}
      />
    );
  }
  return (
    <LocationForm
      title="Edit Location"
      initial={query.data}
      onSubmit={async (body, images) => {
        const location = await update.mutateAsync(body);
        await uploadLocationImages(location.id, images, backgroundUpload.upload, logoUpload.upload);
        return location.id;
      }}
      onSuccess={(savedID) =>
        void navigate({ to: "/locations/$id", params: { id: String(savedID) } })
      }
      onCancel={() => void navigate({ to: "/locations/$id", params: { id: String(id) } })}
    />
  );
}
