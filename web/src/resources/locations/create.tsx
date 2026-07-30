import { LocationFields } from "@/resources/locations/fields";
import type { ReactElement } from "react";
import { Create, ListButton, SimpleForm, TopToolbar } from "react-admin";

const LocationCreateActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const LocationCreate = (): ReactElement => (
  <Create redirect="edit" actions={<LocationCreateActions />}>
    <SimpleForm>
      <LocationFields />
    </SimpleForm>
  </Create>
);
