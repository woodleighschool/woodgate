import type { ReactElement } from "react";
import { DataTable, DateField, List, SearchInput } from "react-admin";

const apiKeyFilters = [<SearchInput key="search" source="search" alwaysOn />];

export const APIKeyList = (): ReactElement => (
  <List sort={{ field: "created_at", order: "DESC" }} filters={apiKeyFilters}>
    <DataTable rowClick="show">
      <DataTable.Col source="name" label="Name" />
      <DataTable.Col source="key_prefix" label="Prefix" />
      <DataTable.Col source="last_used_at" label="Last Used">
        <DateField source="last_used_at" showTime />
      </DataTable.Col>
      <DataTable.Col source="expires_at" label="Expires">
        <DateField source="expires_at" showTime />
      </DataTable.Col>
      <DataTable.Col source="created_at" label="Created">
        <DateField source="created_at" showTime />
      </DataTable.Col>
    </DataTable>
  </List>
);
