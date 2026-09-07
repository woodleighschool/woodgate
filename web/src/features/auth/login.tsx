import { useForm } from "@tanstack/react-form";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useState } from "react";

import { FormActions } from "@components/form-actions";
import { Logo } from "@components/logo";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@components/ui/card";
import { FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { ValidatedFormField } from "@components/validated-form-field";
import { requiredString } from "@lib/form-validation";

import { authApi } from "./api";

export function LoginPage() {
  const providers = useQuery({
    queryKey: ["auth", "providers"],
    queryFn: ({ signal }) => authApi.listProviders(signal),
    retry: 1,
  });
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { from?: string };
  const [error, setError] = useState<Error | null>(null);
  const returnTo =
    search.from?.startsWith("/") &&
    !search.from.startsWith("//") &&
    !search.from.startsWith("/login")
      ? search.from
      : "/";
  const form = useForm({
    defaultValues: { username: "", password: "" },
    onSubmit: async ({ value }) => {
      setError(null);
      try {
        await authApi.loginLocal({
          user: value.username,
          passwd: value.password,
          aud: globalThis.location.origin,
        });
        queryClient.clear();
        await navigate({ to: returnTo, replace: true });
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to sign in"));
      }
    },
  });
  const available = providers.data?.map((provider) => provider.trim().toLowerCase());
  const oauth = new URLSearchParams({
    site: globalThis.location.origin,
    from: `${globalThis.location.origin}/#${returnTo}`,
  });
  return (
    <main className="flex min-h-svh items-center justify-center bg-muted/30 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="mb-4">
            <Logo className="size-12" />
          </div>
          <CardTitle>Sign In</CardTitle>
          <CardDescription>Use a Microsoft account or a local account.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {!available || available.includes("microsoft") ? (
            <Button
              nativeButton={false}
              className="w-full"
              render={
                <a
                  aria-label="Continue With Microsoft"
                  href={`/auth/microsoft/login?${oauth.toString()}`}
                />
              }
            >
              Continue With Microsoft
            </Button>
          ) : null}
          {!available || available.includes("local") ? (
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void form.handleSubmit();
              }}
            >
              <FieldGroup>
                <form.Field name="username" validators={{ onSubmit: requiredString("Username") }}>
                  {(field) => (
                    <ValidatedFormField field={field} label="Username" htmlFor="username" required>
                      {(control) => (
                        <Input
                          {...control}
                          name="username"
                          autoComplete="username"
                          value={field.state.value}
                          onChange={(event) => field.handleChange(event.target.value)}
                          onBlur={field.handleBlur}
                        />
                      )}
                    </ValidatedFormField>
                  )}
                </form.Field>
                <form.Field name="password" validators={{ onSubmit: requiredString("Password") }}>
                  {(field) => (
                    <ValidatedFormField field={field} label="Password" htmlFor="password" required>
                      {(control) => (
                        <Input
                          {...control}
                          name="password"
                          type="password"
                          autoComplete="current-password"
                          value={field.state.value}
                          onChange={(event) => field.handleChange(event.target.value)}
                          onBlur={field.handleBlur}
                        />
                      )}
                    </ValidatedFormField>
                  )}
                </form.Field>
                <QueryError title="Unable to sign in" error={error} />
                <FormActions nativeSubmit form={form} submitLabel="Sign In" />
              </FieldGroup>
            </form>
          ) : null}
          {available?.length === 0 ? (
            <p className="text-sm text-muted-foreground">No sign-in providers are available.</p>
          ) : null}
        </CardContent>
      </Card>
    </main>
  );
}
