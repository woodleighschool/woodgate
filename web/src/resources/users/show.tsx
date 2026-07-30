import { AccessTab } from "@/resources/shared/accessTab";
import { SourceField } from "@/resources/shared/sourceField";
import type { ReactElement } from "react";
import {
  DataTable,
  DateField,
  Labeled,
  ListButton,
  Pagination,
  ReferenceField,
  ReferenceManyField,
  Show,
  TabbedShowLayout,
  TextField,
  TopToolbar,
  useCanAccess,
} from "react-admin";

const UserShowActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const UserShow = (): ReactElement => <UserShowBody />;

const UserShowBody = (): ReactElement => {
  const { canAccess: canListGroups } = useCanAccess({ action: "list", resource: "groups" });
  const { canAccess: canListCheckins } = useCanAccess({ action: "list", resource: "checkins" });
  const { canAccess: canWriteUsers } = useCanAccess({ action: "write", resource: "users" });

  return (
    <Show actions={<UserShowActions />}>
      <TabbedShowLayout>
        <TabbedShowLayout.Tab label="Overview">
          <TextField source="display_name" label="Name" />
          <TextField source="upn" label="UPN" />
          <TextField source="department" />
          <Labeled label="Source">
            <SourceField />
          </Labeled>
          <DateField source="created_at" label="Created" showTime />
          <DateField source="updated_at" label="Updated" showTime />
        </TabbedShowLayout.Tab>
        {canListGroups ? (
          <TabbedShowLayout.Tab label="Groups">
            <ReferenceManyField reference="group-memberships" target="user_id" pagination={<Pagination />}>
              <DataTable rowClick={false} bulkActionButtons={false}>
                <DataTable.Col source="group_id" label="Group">
                  <ReferenceField source="group_id" reference="groups">
                    <TextField source="name" />
                  </ReferenceField>
                </DataTable.Col>
              </DataTable>
            </ReferenceManyField>
          </TabbedShowLayout.Tab>
        ) : undefined}
        {canListCheckins ? (
          <TabbedShowLayout.Tab label="Check-ins">
            <ReferenceManyField reference="checkins" target="user_id" pagination={<Pagination />}>
              <DataTable rowClick="show" bulkActionButtons={false}>
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
            </ReferenceManyField>
          </TabbedShowLayout.Tab>
        ) : undefined}
        {canWriteUsers ? (
          <TabbedShowLayout.Tab label="Access">
            <AccessTab resource="users" />
          </TabbedShowLayout.Tab>
        ) : undefined}
      </TabbedShowLayout>
    </Show>
  );
};
