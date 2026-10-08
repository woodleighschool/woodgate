import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { LocationForm, uploadLocationImages } from "@features/locations/fields";
import {
  useLocation,
  useUpdateLocation,
  useUploadLocationBackground,
  useUploadLocationLogo,
} from "@features/resources/queries";

export function LocationEditPage() {
  const navigate = useNavigate();
  const { id } = useParams({ from: "/_authenticated/locations/$id/edit" });
  const query = useLocation(id);
  const update = useUpdateLocation(id);
  const backgroundUpload = useUploadLocationBackground();
  const logoUpload = useUploadLocationLogo();
  if (query.error || !query.data) {
    return <QueryGate title="Failed to Load Location" error={query.error} />;
  }
  return (
    <LocationForm
      title="Edit Location"
      initial={query.data}
      onSubmit={async (body, images) => {
        const attachments = await uploadLocationImages(
          images,
          backgroundUpload.upload,
          logoUpload.upload,
        );
        return (await update.mutateAsync({ ...body, ...attachments })).id;
      }}
      onSuccess={(savedID) => void navigate({ to: "/locations/$id", params: { id: savedID } })}
      onCancel={() => void navigate({ to: "/locations/$id", params: { id } })}
    />
  );
}
