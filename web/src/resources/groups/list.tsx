import type { ReactElement } from "react";
import { DataTable, List, NumberField, SearchInput } from "react-admin";

const groupFilters = [<SearchInput key="search" source="search" alwaysOn />];

export const GroupList = (): ReactElement => (
  <List sort={{ field: "name", order: "ASC" }} filters={groupFilters}>
    <DataTable rowClick="show">
      <DataTable.Col source="name" label="Name" />
      <DataTable.Col source="description" label="Description" />
      <DataTable.Col source="member_count" label="Members">
        <NumberField source="member_count" />
      </DataTable.Col>
    </DataTable>
  </List>
);
