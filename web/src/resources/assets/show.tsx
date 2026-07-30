import type { ReactElement } from "react";
import {
  CanAccess,
  DateField,
  DeleteButton,
  EditButton,
  ImageField,
  ListButton,
  Show,
  SimpleShowLayout,
  TextField,
  TopToolbar,
  useRecordContext,
} from "react-admin";

const AssetShowActions = (): ReactElement => <AssetShowActionsBody />;

const AssetShowActionsBody = (): ReactElement => {
  const record = useRecordContext<{ type?: string }>();

  return (
    <TopToolbar>
      <ListButton />
      {record?.type === "asset" ? (
        <CanAccess action="edit" resource="assets">
          <EditButton />
        </CanAccess>
      ) : undefined}
      <CanAccess action="delete" resource="assets">
        <DeleteButton mutationMode="pessimistic" />
      </CanAccess>
    </TopToolbar>
  );
};

export const AssetShow = (): ReactElement => (
  <Show actions={<AssetShowActions />}>
    <SimpleShowLayout>
      <ImageField source="url" title="name" label="Image" />
      <TextField source="name" label="Name" emptyText="-" />
      <TextField source="type" label="Type" />
      <TextField source="url" label="URL" />
      <DateField source="created_at" label="Created" showTime />
      <DateField source="updated_at" label="Updated" showTime />
    </SimpleShowLayout>
  </Show>
);
