import type { ReactElement } from "react";
import { Create, ListButton, SimpleForm, TextInput, TopToolbar, useRedirect } from "react-admin";

interface APIKeyShowState {
  baseUrl: string;
  secret: string;
}

const APIKeyCreateActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
  </TopToolbar>
);

export const APIKeyCreate = (): ReactElement => {
  const redirect = useRedirect();

  return (
    <Create
      actions={<APIKeyCreateActions />}
      mutationOptions={{
        onSuccess: (data): void => {
          const record = data as { id: string; secret: string };
          const state: APIKeyShowState = {
            baseUrl: globalThis.location.origin,
            secret: record.secret,
          };
          redirect("show", "api-keys", record.id, record, state);
        },
      }}
    >
      <SimpleForm>
        <TextInput source="name" label="Name" fullWidth />
      </SimpleForm>
    </Create>
  );
};
