import { SourceField } from "@/resources/shared/sourceField";
import type { ReactElement } from "react";
import { DataTable, DateField, List, SearchInput } from "react-admin";

const userFilters = [<SearchInput key="search" source="search" alwaysOn />];

export const UserList = (): ReactElement => (
  <List sort={{ field: "display_name", order: "ASC" }} filters={userFilters}>
    <DataTable rowClick="show">
      <DataTable.Col source="display_name" label="Name" />
      <DataTable.Col source="upn" label="UPN" />
      <DataTable.Col source="department" label="Department" />
      <DataTable.Col source="source" label="Source">
        <SourceField />
      </DataTable.Col>
      <DataTable.Col source="updated_at" label="Updated">
        <DateField source="updated_at" showTime />
      </DataTable.Col>
    </DataTable>
  </List>
);
