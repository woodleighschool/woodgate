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
      <ReferenceField source="user_id" reference="users" label="User">
        <TextField source="display_name" />
      </ReferenceField>
      <ReferenceField source="user_id" reference="users" label="Department">
        <TextField source="department" />
      </ReferenceField>
      <ReferenceField source="location_id" reference="locations" label="Location">
        <TextField source="name" />
      </ReferenceField>
      <TextField source="direction" label="Direction" />
      <TextField source="notes" label="Notes" />
      <ReferenceField source="asset_id" reference="assets" label="Photo">
        <ImageField source="url" title="name" />
      </ReferenceField>
      <CreatedByField />
      <DateField source="created_at" label="Created" showTime />
    </SimpleShowLayout>
  </Show>
);
