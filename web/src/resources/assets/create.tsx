import type { ReactElement } from "react";
import { Create, FileInput, ImageField, ListButton, SimpleForm, TextInput, TopToolbar, required } from "react-admin";

const AssetCreateActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const AssetCreate = (): ReactElement => (
  <Create redirect="edit" actions={<AssetCreateActions />}>
    <SimpleForm>
      <TextInput source="name" label="Name" helperText="Optional cosmetic name" fullWidth />
      <FileInput
        source="file"
        label="Asset File"
        accept={{ "image/png": [], "image/jpeg": [] }}
        validate={[required()]}
      >
        <ImageField source="src" title="title" />
      </FileInput>
    </SimpleForm>
  </Create>
);
