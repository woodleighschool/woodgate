import type { ReactElement } from "react";
import {
  CanAccess,
  DeleteButton,
  Edit,
  FileInput,
  ImageField,
  ListButton,
  SimpleForm,
  TextInput,
  TopToolbar,
  useRecordContext,
} from "react-admin";

const AssetEditActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
    <CanAccess action="delete" resource="assets">
      <DeleteButton mutationMode="pessimistic" />
    </CanAccess>
  </TopToolbar>
);

export const AssetEdit = (): ReactElement => (
  <Edit mutationMode="optimistic" redirect="edit" actions={<AssetEditActions />}>
    <AssetEditForm />
  </Edit>
);

const AssetEditForm = (): ReactElement => {
  const record = useRecordContext<{ type?: string }>();
  const isPhoto = record?.type === "photo";

  return (
    <SimpleForm toolbar={isPhoto ? false : undefined}>
      <ImageField source="url" title="name" label="Current Image" />
      <TextInput source="name" label="Name" helperText="Optional cosmetic name" fullWidth disabled={isPhoto} />
      <TextInput source="type" label="Type" disabled fullWidth />
      {isPhoto ? undefined : (
        <FileInput source="file" label="Replace Asset File" accept={{ "image/png": [], "image/jpeg": [] }}>
          <ImageField source="src" title="title" />
        </FileInput>
      )}
    </SimpleForm>
  );
};
