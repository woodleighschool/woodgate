import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { EnumBadge } from "@components/enum-badge";
import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@components/ui/field";
import { Input } from "@components/ui/input";
import { RadioGroup, RadioGroupItem } from "@components/ui/radio-group";
import { ValidatedFormField } from "@components/validated-form-field";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { User, UserCreate, UserMutation } from "@lib/api";
import { emailAddress } from "@lib/form-validation";

interface UserCreateFormState {
  email: string;
  name: string;
  access_enabled: boolean;
  password: string;
}

const userCreateFormSchema = z.object({
  email: emailAddress(),
  name: z.string().trim(),
  access_enabled: z.boolean(),
  password: z.string().min(12, "Password must be at least 12 characters."),
});

export function UserCreateForm({
  onSubmit,
  onSuccess,
  onCancel,
}: {
  onSubmit: (body: UserCreate) => Promise<number>;
  onSuccess: (id: number) => void;
  onCancel: () => void;
}) {
  const form = useForm({
    defaultValues: {
      email: "",
      name: "",
      access_enabled: true as boolean,
      password: "",
    } satisfies UserCreateFormState,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: userCreateFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit({
        email: value.email.trim(),
        name: value.name.trim(),
        access_enabled: value.access_enabled,
        password: value.password,
      });
      // Re-baseline before navigating so the exit guard sees saved state.
      formApi.reset(value);
      onSuccess(id);
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: onCancel,
  });

  return (
    <>
      <PageShell>
        <PageHeader title="Create User" />

        <FieldGroup className="max-w-3xl">
          <form.Field name="email">
            {(field) => (
              <ValidatedFormField field={field} label="Email" htmlFor="user-email" required>
                {(control) => (
                  <Input
                    {...control}
                    type="email"
                    required
                    autoComplete="off"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>

          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Name" htmlFor="user-name">
                {(control) => (
                  <Input
                    {...control}
                    type="text"
                    autoComplete="off"
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>

          <form.Field name="access_enabled">
            {(field) => (
              <FieldSet>
                <FieldLegend variant="label">Sign-in Access</FieldLegend>
                <RadioGroup
                  name={field.name}
                  value={field.state.value ? "enabled" : "disabled"}
                  className="grid gap-2 md:grid-cols-2"
                  onValueChange={(value) => field.handleChange(value === "enabled")}
                >
                  {ACCESS_OPTIONS.map((option) => (
                    <FieldLabel key={option.value} htmlFor={`user-access-${option.value}`}>
                      <Field orientation="horizontal">
                        <RadioGroupItem id={`user-access-${option.value}`} value={option.value} />
                        <FieldContent>
                          <FieldTitle>{option.label}</FieldTitle>
                          <FieldDescription>{option.description}</FieldDescription>
                        </FieldContent>
                      </Field>
                    </FieldLabel>
                  ))}
                </RadioGroup>
              </FieldSet>
            )}
          </form.Field>

          <form.Field name="password">
            {(field) => (
              <ValidatedFormField field={field} label="Password" htmlFor="user-password" required>
                {(control) => (
                  <Input
                    {...control}
                    type="password"
                    autoComplete="new-password"
                    required
                    minLength={12}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>
        </FieldGroup>

        <FormActions form={form} submitLabel="Create" onCancel={exitGuard.requestDiscard} />
      </PageShell>

      {exitGuard.dialog}
    </>
  );
}

interface UserFormState {
  name: string;
  access_enabled: boolean;
  password: string;
}
export function userFromDetail(user: User): UserFormState {
  return {
    name: user.name,
    access_enabled: user.access_enabled,
    password: "",
  };
}
const userFormSchema = z.object({
  name: z.string(),
  access_enabled: z.boolean(),
  password: z
    .string()
    .refine(
      (value) => value.trim() === "" || value.length >= 12,
      "Password must be at least 12 characters.",
    ),
});
export function UserForm({
  user,
  initial,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  user: User;
  initial: UserFormState;
  onSubmit: (body: UserMutation) => Promise<void>;
  onSuccess?: () => void;
  onCancel: () => void;
}) {
  const isLocal = user.source === "local";
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: userFormSchema },
    onSubmit: async ({ value, formApi }) => {
      await onSubmit({
        name: isLocal ? value.name.trim() : user.name,
        access_enabled: value.access_enabled,
        password: isLocal && value.password.trim() !== "" ? value.password : undefined,
      });
      // Re-baseline so the saved values count as unchanged.
      formApi.reset({ ...value, password: "" });
      onSuccess?.();
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: onCancel,
  });
  return (
    <>
      <PageShell>
        <PageHeader
          title="Edit User"
          context={<EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />}
        />

        <FieldGroup className="max-w-3xl">
          <Field>
            <FieldLabel htmlFor="user-email">Email</FieldLabel>
            <Input id="user-email" type="email" value={user.email} disabled />
          </Field>

          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Display Name" htmlFor="user-name">
                {(control) => (
                  <Input
                    {...control}
                    type="text"
                    autoComplete="off"
                    disabled={!isLocal}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>

          <form.Field name="access_enabled">
            {(field) => (
              <FieldSet>
                <FieldLegend variant="label">Sign-in Access</FieldLegend>
                <RadioGroup
                  name={field.name}
                  value={field.state.value ? "enabled" : "disabled"}
                  className="grid gap-2 md:grid-cols-2"
                  onValueChange={(value) => field.handleChange(value === "enabled")}
                >
                  {ACCESS_OPTIONS.map((option) => (
                    <FieldLabel key={option.value} htmlFor={`user-access-${option.value}`}>
                      <Field orientation="horizontal">
                        <RadioGroupItem id={`user-access-${option.value}`} value={option.value} />
                        <FieldContent>
                          <FieldTitle>{option.label}</FieldTitle>
                          <FieldDescription>{option.description}</FieldDescription>
                        </FieldContent>
                      </Field>
                    </FieldLabel>
                  ))}
                </RadioGroup>
              </FieldSet>
            )}
          </form.Field>

          {isLocal ? (
            <form.Field name="password">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Password"
                  htmlFor="user-password"
                  description="Set a new password."
                >
                  {(control) => (
                    <Input
                      {...control}
                      type="password"
                      autoComplete="new-password"
                      minLength={12}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
          ) : null}
        </FieldGroup>

        <FormActions form={form} submitLabel="Save" onCancel={exitGuard.requestDiscard} />

        {exitGuard.dialog}
      </PageShell>
    </>
  );
}

const ACCESS_OPTIONS = [
  { value: "enabled", label: "Enabled", description: "Can sign in to the app." },
  { value: "disabled", label: "Disabled", description: "Cannot sign in to the app." },
] as const;
