import { useNavigate } from "@tanstack/react-router";

import { LocationForm, uploadLocationImages } from "@features/locations/fields";
import {
  useCreateLocation,
  useUploadLocationBackground,
  useUploadLocationLogo,
} from "@features/resources/queries";

export function LocationCreatePage() {
  const navigate = useNavigate();
  const create = useCreateLocation();
  const backgroundUpload = useUploadLocationBackground();
  const logoUpload = useUploadLocationLogo();
  return (
    <LocationForm
      title="Create Location"
      onSubmit={async (body, images) => {
        const location = await create.mutateAsync(body);
        await uploadLocationImages(location.id, images, backgroundUpload.upload, logoUpload.upload);
        return location.id;
      }}
      onSuccess={(id) => void navigate({ to: "/locations/$id", params: { id: String(id) } })}
      onCancel={() => void navigate({ to: "/locations" })}
    />
  );
}
