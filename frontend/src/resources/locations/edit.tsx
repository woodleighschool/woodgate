import { LocationFields } from "@/resources/locations/fields";
import type { ReactElement } from "react";
import {
  CanAccess,
  DateField,
  DeleteButton,
  Edit,
  Labeled,
  ListButton,
  SimpleForm,
  TextField,
  TopToolbar,
} from "react-admin";

const LocationEditActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
    <CanAccess action="delete" resource="locations">
      <DeleteButton mutationMode="pessimistic" />
    </CanAccess>
  </TopToolbar>
);

export const LocationEdit = (): ReactElement => (
  <Edit mutationMode="optimistic" redirect="edit" actions={<LocationEditActions />}>
    <SimpleForm>
      <Labeled label="ID">
        <TextField source="id" />
      </Labeled>
      <LocationFields />
      <Labeled label="Created">
        <DateField source="created_at" showTime />
      </Labeled>
      <Labeled label="Updated">
        <DateField source="updated_at" showTime />
      </Labeled>
    </SimpleForm>
  </Edit>
);
