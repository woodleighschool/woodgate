import type { ReactElement } from "react";
import { Create, ListButton, SimpleForm, TextInput, TopToolbar, useRedirect } from "react-admin";

interface APIKeyShowState {
  baseUrl: string;
  secret: string;
}

const isCreatedAPIKey = (value: unknown): value is { id: string; secret: string } =>
  typeof value === "object" &&
  value !== null &&
  typeof Reflect.get(value, "id") === "string" &&
  typeof Reflect.get(value, "secret") === "string";

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
          if (!isCreatedAPIKey(data)) {
            throw new Error("API key response is missing its ID or secret");
          }
          const record = data;
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
