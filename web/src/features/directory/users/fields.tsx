import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { EnumBadge } from "@components/enum-badge";
import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { ValidatedFormField } from "@components/validated-form-field";
import { RolePicker } from "@features/authz/role-picker";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { User, UserCreate, UserMutation } from "@lib/api";
import { emailAddress } from "@lib/form-validation";

interface UserCreateFormState {
  email: string;
  name: string;
  password: string;
  role_ids: number[];
}

const userCreateFormSchema = z.object({
  email: emailAddress(),
  name: z.string().trim(),
  password: z.string().min(12, "Password must be at least 12 characters."),
  role_ids: z.array(z.number()),
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
      password: "",
      role_ids: [] as number[],
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
        password: value.password,
        role_ids: value.role_ids,
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

          <form.Field name="role_ids">
            {(field) => <RolePicker value={field.state.value} onChange={field.handleChange} />}
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
  password: string;
  role_ids: number[];
}
export function userFromDetail(user: User): UserFormState {
  return {
    name: user.name,
    password: "",
    role_ids: user.roles.map((role) => role.id),
  };
}
const userFormSchema = z.object({
  name: z.string(),
  role_ids: z.array(z.number()),
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
        password: isLocal && value.password.trim() !== "" ? value.password : undefined,
        role_ids: value.role_ids,
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

          <form.Field name="role_ids">
            {(field) => (
              <RolePicker
                value={field.state.value}
                onChange={field.handleChange}
                description="Direct roles are combined with roles inherited from directory groups. No effective role means No Access."
              />
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
