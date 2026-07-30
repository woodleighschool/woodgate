import type { ReactElement } from "react";
import {
  DateField,
  ImageField,
  Labeled,
  ListButton,
  ReferenceField,
  Show,
  SimpleShowLayout,
  TextField,
  TopToolbar,
  useRecordContext,
} from "react-admin";

const CreatedByField = (): ReactElement | undefined => {
  const record = useRecordContext();
  if (!record) return undefined;
  if (record.created_by_kind === "user") {
    return (
      <Labeled label="Created By User">
        <ReferenceField source="created_by_id" reference="users">
          <TextField source="display_name" />
        </ReferenceField>
      </Labeled>
    );
  }
  if (record.created_by_kind === "api_key") {
    return (
      <Labeled label="Created By API Key">
        <ReferenceField source="created_by_id" reference="api-keys">
          <TextField source="name" />
        </ReferenceField>
      </Labeled>
    );
  }
  return undefined;
};

const CheckinShowActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const CheckinShow = (): ReactElement => (
  <Show actions={<CheckinShowActions />}>
    <SimpleShowLayout>
      <TextField source="user_display_name" label="User" />
      <TextField source="department" label="Department" />
      <TextField source="location_name" label="Location" />
      <TextField source="direction" label="Direction" />
      <TextField source="notes" label="Notes" />
      <ImageField source="photo_url" title="user_display_name" label="Photo" />
      <CreatedByField />
      <DateField source="created_at" label="Created" showTime />
    </SimpleShowLayout>
  </Show>
);
