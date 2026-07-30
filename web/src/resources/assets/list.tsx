import type { ReactElement } from "react";
import { DataTable, DateField, ImageField, List, SearchInput, SelectInput, TextField } from "react-admin";

const assetFilters = [
  <SearchInput key="search" source="search" alwaysOn />,
  <SelectInput
    key="type"
    source="type"
    choices={[
      { id: "asset", name: "Asset" },
      { id: "photo", name: "Photo" },
    ]}
  />,
];

export const AssetList = (): ReactElement => (
  <List sort={{ field: "name", order: "ASC" }} filters={assetFilters}>
    <DataTable>
      <DataTable.Col source="url" label="Preview">
        <ImageField source="url" title="name" sx={{ "& img": { width: 56, height: 56, objectFit: "cover" } }} />
      </DataTable.Col>
      <DataTable.Col source="name" label="Name">
        <TextField source="name" emptyText="-" />
      </DataTable.Col>
      <DataTable.Col source="type" label="Type" />
      <DataTable.Col source="url" label="URL">
        <TextField source="url" />
      </DataTable.Col>
      <DataTable.Col source="updated_at" label="Updated">
        <DateField source="updated_at" showTime />
      </DataTable.Col>
    </DataTable>
  </List>
);
