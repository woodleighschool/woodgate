import { trimmedRequired } from "@/resources/shared/validation";
import type { ReactElement } from "react";
import {
  AutocompleteArrayInput,
  BooleanInput,
  ReferenceArrayInput,
  ReferenceInput,
  SelectInput,
  TextInput,
} from "react-admin";

export const LocationFields = (): ReactElement => (
  <>
    <TextInput source="name" label="Name" validate={[trimmedRequired("Name")]} fullWidth />
    <TextInput source="description" label="Description" fullWidth multiline minRows={3} />
    <BooleanInput source="enabled" label="Enabled" />
    <BooleanInput source="notes" label="Notes" />
    <BooleanInput source="photo" label="Photo" />
    <ReferenceInput source="background_asset_id" reference="assets" filter={{ type: "asset" }}>
      <SelectInput optionText="name" label="Background Asset" />
    </ReferenceInput>
    <ReferenceInput source="logo_asset_id" reference="assets" filter={{ type: "asset" }}>
      <SelectInput optionText="name" label="Logo Asset" />
    </ReferenceInput>
    <ReferenceArrayInput source="group_ids" reference="groups">
      <AutocompleteArrayInput
        optionText="name"
        label="Groups"
        filterToQuery={(q: string): { search: string } => ({ search: q })}
      />
    </ReferenceArrayInput>
  </>
);
