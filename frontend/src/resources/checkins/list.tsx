import { CHECKIN_DIRECTION_CHOICES } from "@/resources/checkins/choices";
import type { ReactElement } from "react";
import {
  DataTable,
  DateField,
  ImageField,
  List,
  ReferenceField,
  SearchInput,
  SelectInput,
  TextField,
} from "react-admin";

const checkinFilters = [
  <SearchInput key="search" source="search" alwaysOn />,
  <SelectInput key="direction" source="direction" choices={CHECKIN_DIRECTION_CHOICES} />,
];

export const CheckinList = (): ReactElement => (
  <List sort={{ field: "created_at", order: "DESC" }} filters={checkinFilters}>
    <DataTable rowClick="show">
      <DataTable.Col source="asset_id" label="Photo">
        <ReferenceField source="asset_id" reference="assets">
          <ImageField source="url" title="name" sx={{ "& img": { width: 56, height: 56, objectFit: "cover" } }} />
        </ReferenceField>
      </DataTable.Col>
      <DataTable.Col source="user_id" label="User">
        <ReferenceField source="user_id" reference="users">
          <TextField source="display_name" />
        </ReferenceField>
      </DataTable.Col>
      <DataTable.Col source="user_id" label="Department">
        <ReferenceField source="user_id" reference="users">
          <TextField source="department" />
        </ReferenceField>
      </DataTable.Col>
      <DataTable.Col source="location_id" label="Location">
        <ReferenceField source="location_id" reference="locations">
          <TextField source="name" />
        </ReferenceField>
      </DataTable.Col>
      <DataTable.Col source="direction" label="Direction" />
      <DataTable.Col source="notes" label="Notes" />
      <DataTable.Col source="created_at" label="Created">
        <DateField source="created_at" showTime />
      </DataTable.Col>
    </DataTable>
  </List>
);
