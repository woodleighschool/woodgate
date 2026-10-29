import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Textarea } from "@components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@components/ui/toggle-group";
import { ValidatedFormField } from "@components/validated-form-field";
import { CheckinLocationCombobox, CheckinPersonCombobox } from "@features/checkins/choices";
import { useCreateCheckin } from "@features/resources/queries";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { CheckinCreate, CheckinLocationSummary, CheckinPersonSummary } from "@lib/api";

const schema = z.object({
  location: z
    .object({ id: z.number().positive(), name: z.string() })
    .nullable()
    .refine((value) => value !== null, "Location is required."),
  person: z
    .object({
      id: z.number().positive(),
      name: z.string(),
      email: z.string(),
      department: z.string().optional(),
    })
    .nullable()
    .refine((value) => value !== null, "Person is required."),
  direction: z
    .enum(["check_in", "check_out"])
    .nullable()
    .refine((value) => value !== null, "Direction is required."),
  notes: z.string(),
});

export function CheckinCreatePage() {
  const navigate = useNavigate();
  const create = useCreateCheckin();
  const form = useForm({
    defaultValues: {
      location: null as CheckinLocationSummary | null,
      person: null as CheckinPersonSummary | null,
      direction: null as CheckinCreate["direction"] | null,
      notes: "",
    },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      if (!value.location || !value.person || !value.direction) return;
      try {
        const checkin = await create.mutateAsync({
          location_id: value.location.id,
          user_id: value.person.id,
          direction: value.direction,
          notes: value.notes.trim(),
        });
        formApi.reset(value);
        void navigate({ to: "/checkins/$id", params: { id: checkin.id } });
      } catch {
        // The mutation error is displayed with the form so it can be retried.
      }
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: () => void navigate({ to: "/checkins" }),
  });
  return (
    <>
      <PageShell>
        <PageHeader title="Create Check-in" description="Record an arrival or departure now." />
        <FieldGroup className="max-w-3xl">
          <form.Field name="location">
            {(field) => (
              <ValidatedFormField
                field={field}
                label="Location"
                htmlFor="checkin-location"
                required
              >
                {(control) => (
                  <CheckinLocationCombobox
                    {...control}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(location) => {
                      if (location?.id !== field.state.value?.id)
                        form.setFieldValue("person", null);
                      field.handleChange(location);
                    }}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>
          <form.Subscribe selector={(state) => state.values.location?.id}>
            {(locationID) => (
              <form.Field name="person">
                {(field) => (
                  <ValidatedFormField
                    field={field}
                    label="Person"
                    htmlFor="checkin-person"
                    required
                  >
                    {(control) =>
                      locationID ? (
                        <CheckinPersonCombobox
                          {...control}
                          key={locationID}
                          locationID={locationID}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={field.handleChange}
                        />
                      ) : (
                        <Input {...control} disabled placeholder="Select a location first" />
                      )
                    }
                  </ValidatedFormField>
                )}
              </form.Field>
            )}
          </form.Subscribe>
          <form.Field name="direction">
            {(field) => (
              <ValidatedFormField
                field={field}
                label="Direction"
                htmlFor="checkin-direction"
                required
              >
                {(control) => (
                  <ToggleGroup
                    {...control}
                    aria-label="Direction"
                    variant="outline"
                    value={field.state.value ? [field.state.value] : []}
                    onBlur={field.handleBlur}
                    onValueChange={(values) => {
                      const direction = values[0];
                      field.handleChange(
                        direction === "check_in" || direction === "check_out" ? direction : null,
                      );
                    }}
                  >
                    <ToggleGroupItem value="check_in">Check In</ToggleGroupItem>
                    <ToggleGroupItem value="check_out">Check Out</ToggleGroupItem>
                  </ToggleGroup>
                )}
              </ValidatedFormField>
            )}
          </form.Field>
          <form.Field name="notes">
            {(field) => (
              <ValidatedFormField field={field} label="Notes" htmlFor="checkin-notes">
                {(control) => (
                  <Textarea
                    {...control}
                    value={field.state.value}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                  />
                )}
              </ValidatedFormField>
            )}
          </form.Field>
        </FieldGroup>
        <QueryError title="Failed to Create Check-in" error={create.error} />
        <FormActions form={form} submitLabel="Create" onCancel={exitGuard.requestDiscard} />
      </PageShell>
      {exitGuard.dialog}
    </>
  );
}
