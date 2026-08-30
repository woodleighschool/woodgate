import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Field, FieldGroup, FieldLabel, FieldTitle } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Switch } from "@components/ui/switch";
import { ValidatedFormField } from "@components/validated-form-field";
import { LocationCombobox } from "@features/resources/resource-comboboxes";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { Station, StationMutation } from "@lib/api";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required."),
  location_id: z.string().min(1, "Location is required."),
  enabled: z.boolean(),
});

export function StationForm({
  title,
  initial,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  title: string;
  initial?: Station;
  onSubmit: (body: StationMutation) => Promise<number>;
  onSuccess: (id: number) => void;
  onCancel: () => void;
}) {
  const form = useForm({
    defaultValues: {
      name: initial?.name ?? "",
      location_id: initial?.location_id?.toString() ?? "",
      enabled: initial?.enabled ?? true,
    },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit({
        name: value.name.trim(),
        location_id: Number(value.location_id),
        enabled: value.enabled,
      });
      formApi.reset(value);
      onSuccess(id);
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel });
  return (
    <>
      <PageShell>
        <PageHeader title={title} />
        <FieldGroup className="max-w-3xl">
          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Name" htmlFor="station-name" required>
                {(control) => (
                  <Input
                    {...control}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>
          <form.Field name="location_id">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="station-location">Location</FieldLabel>
                <LocationCombobox
                  id="station-location"
                  required
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={field.handleChange}
                />
              </Field>
            )}
          </form.Field>
          <form.Field name="enabled">
            {(field) => (
              <Field orientation="horizontal">
                <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                <FieldTitle>Enabled</FieldTitle>
              </Field>
            )}
          </form.Field>
        </FieldGroup>
        <FormActions
          form={form}
          submitLabel={initial ? "Save" : "Create"}
          onCancel={exitGuard.requestDiscard}
        />
      </PageShell>
      {exitGuard.dialog}
    </>
  );
}
