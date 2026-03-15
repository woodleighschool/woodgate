import type { ReactElement } from "react";
import {
  DataTable,
  DateField,
  ListButton,
  NumberField,
  Pagination,
  ReferenceField,
  ReferenceManyField,
  Show,
  TabbedShowLayout,
  TextField,
  TopToolbar,
  useCanAccess,
} from "react-admin";

const GroupShowActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const GroupShow = (): ReactElement => <GroupShowBody />;

const GroupShowBody = (): ReactElement => {
  const { canAccess: canListMemberships } = useCanAccess({ action: "list", resource: "groups" });

  return (
    <Show actions={<GroupShowActions />}>
      <TabbedShowLayout>
        <TabbedShowLayout.Tab label="Overview">
          <TextField source="id" />
          <TextField source="name" label="Name" />
          <TextField source="description" label="Description" />
          <NumberField source="member_count" label="Members" />
          <DateField source="created_at" label="Created" showTime />
          <DateField source="updated_at" label="Updated" showTime />
        </TabbedShowLayout.Tab>
        {canListMemberships ? (
          <TabbedShowLayout.Tab label="Users">
            <ReferenceManyField reference="group-memberships" target="group_id" pagination={<Pagination />}>
              <DataTable rowClick={false} bulkActionButtons={false}>
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
              </DataTable>
            </ReferenceManyField>
          </TabbedShowLayout.Tab>
        ) : undefined}
      </TabbedShowLayout>
    </Show>
  );
};
