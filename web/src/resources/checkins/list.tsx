import type { ReactElement } from "react";
import {
  DataTable,
  DateField,
  FunctionField,
  ImageField,
  List,
  SearchInput,
  SelectInput,
  useGetList,
} from "react-admin";

import type { Checkin, DepartmentOption } from "@/api/types";
import { CHECKIN_DIRECTION_CHOICES, CHECKIN_DIRECTION_LABELS } from "@/resources/checkins/choices";

interface DepartmentInputProps {
  source: string;
  label: string;
}

const DepartmentInput = ({ source, label }: DepartmentInputProps): ReactElement => {
  const { data: departments = [], isPending } = useGetList<DepartmentOption>(
    "checkin-departments",
    {
      pagination: { page: 1, perPage: 250 },
      sort: { field: "name", order: "ASC" },
      filter: {},
    },
  );

  return (
    <SelectInput
      source={source}
      label={label}
      choices={departments}
      isPending={isPending}
      optionText="name"
    />
  );
};

const checkinFilters = [
  <SearchInput key="search" source="search" alwaysOn />,
  <DepartmentInput key="department" source="department" label="Department" />,
  <SelectInput key="direction" source="direction" choices={CHECKIN_DIRECTION_CHOICES} />,
];

export const CheckinList = (): ReactElement => (
  <List sort={{ field: "created_at", order: "DESC" }} filters={checkinFilters}>
    <DataTable rowClick="show">
      <DataTable.Col source="photo_url" label="Photo" disableSort>
        <ImageField
          source="photo_url"
          title="user_display_name"
          sx={{ "& img": { width: 56, height: 56, objectFit: "cover" } }}
        />
      </DataTable.Col>
      <DataTable.Col source="user_display_name" label="User" />
      <DataTable.Col source="department" label="Department" />
      <DataTable.Col source="location_name" label="Location" />
      <DataTable.Col source="direction" label="Direction">
        <FunctionField<Checkin>
          source="direction"
          render={(record) => CHECKIN_DIRECTION_LABELS[record.direction]}
        />
      </DataTable.Col>
      <DataTable.Col source="notes" label="Notes" />
      <DataTable.Col source="created_at" label="Created">
        <DateField source="created_at" showTime />
      </DataTable.Col>
    </DataTable>
  </List>
);
