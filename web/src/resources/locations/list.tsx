import type { ReactElement } from "react";
import {
  BooleanField,
  CanAccess,
  ChipField,
  DataTable,
  FunctionField,
  List,
  NullableBooleanInput,
  ReferenceArrayField,
  ReferenceField,
  SearchInput,
  SingleFieldList,
  TextField,
} from "react-admin";

const locationFilters = [
  <SearchInput key="search" source="search" alwaysOn />,
  <NullableBooleanInput key="enabled" source="enabled" />,
];

export const LocationList = (): ReactElement => (
  <List sort={{ field: "name", order: "ASC" }} filters={locationFilters}>
    <DataTable rowClick="edit">
      <DataTable.Col source="name" label="Name" />
      <DataTable.Col source="enabled" label="Enabled">
        <BooleanField source="enabled" />
      </DataTable.Col>
      <DataTable.Col source="notes" label="Notes">
        <BooleanField source="notes" />
      </DataTable.Col>
      <DataTable.Col source="photo" label="Photo">
        <BooleanField source="photo" />
      </DataTable.Col>
      <DataTable.Col source="background_asset_id" label="Background Asset">
        <CanAccess
          action="show"
          resource="assets"
          accessDenied={<TextField source="background_asset_id" emptyText="-" />}
        >
          <ReferenceField source="background_asset_id" reference="assets" empty={<>-</>}>
            <TextField source="name" />
          </ReferenceField>
        </CanAccess>
      </DataTable.Col>
      <DataTable.Col source="logo_asset_id" label="Logo Asset">
        <CanAccess action="show" resource="assets" accessDenied={<TextField source="logo_asset_id" emptyText="-" />}>
          <ReferenceField source="logo_asset_id" reference="assets" empty={<>-</>}>
            <TextField source="name" />
          </ReferenceField>
        </CanAccess>
      </DataTable.Col>
      <DataTable.Col source="group_ids" label="Groups">
        <CanAccess
          action="list"
          resource="groups"
          accessDenied={
            <FunctionField
              render={(record: { group_ids?: string[] }): string =>
                record.group_ids && record.group_ids.length > 0 ? record.group_ids.join(", ") : "-"
              }
            />
          }
        >
          <ReferenceArrayField source="group_ids" reference="groups">
            <SingleFieldList linkType={false}>
              <ChipField source="name" size="small" />
            </SingleFieldList>
          </ReferenceArrayField>
        </CanAccess>
      </DataTable.Col>
    </DataTable>
  </List>
);
