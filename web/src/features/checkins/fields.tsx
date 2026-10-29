import { revalidateLogic, useForm } from "@tanstack/react-form";
import { format, parseISO } from "date-fns";
import { CalendarDays } from "lucide-react";
import { useState } from "react";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import { Calendar } from "@components/ui/calendar";
import { FieldGroup } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@components/ui/popover";
import { Textarea } from "@components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@components/ui/toggle-group";
import { ValidatedFormField } from "@components/validated-form-field";
import { CheckinLocationCombobox, CheckinPersonCombobox } from "@features/checkins/choices";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { Checkin, CheckinCreate } from "@lib/api";

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
  occurredDate: z.iso.date({ error: "Enter a valid date." }).or(z.literal("")),
  occurredTime: z.iso.time({ error: "Enter a valid time." }).or(z.literal("")),
});

export function CheckinForm({
  title,
  initial,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  title: string;
  initial?: Checkin;
  onSubmit: (body: CheckinCreate, occurredAt?: string) => Promise<number>;
  onSuccess: (id: number) => void;
  onCancel: () => void;
}) {
  const [error, setError] = useState<Error | null>(null);
  const [dateOpen, setDateOpen] = useState(false);
  const initialTime = initial ? format(new Date(initial.occurred_at), "yyyy-MM-dd'T'HH:mm:ss") : "";
  const form = useForm({
    defaultValues: {
      location: initial?.location ?? null,
      person: initial?.person ?? null,
      direction: initial?.direction ?? null,
      notes: initial?.notes ?? "",
      occurredDate: initialTime.slice(0, 10),
      occurredTime: initialTime.slice(11),
    },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: {
      onDynamic: schema
        .refine((value) => !initial || value.occurredDate !== "", {
          path: ["occurredDate"],
          message: "Date is required.",
        })
        .refine((value) => !initial || value.occurredTime !== "", {
          path: ["occurredTime"],
          message: "Time is required.",
        }),
    },
    onSubmit: async ({ value, formApi }) => {
      if (!value.location || !value.person || !value.direction) return;
      setError(null);
      try {
        const occurredAt = `${value.occurredDate}T${value.occurredTime}`;
        const id = await onSubmit(
          {
            location_id: value.location.id,
            user_id: value.person.id,
            direction: value.direction,
            notes: value.notes.trim(),
          },
          initial
            ? occurredAt === initialTime
              ? initial.occurred_at
              : new Date(occurredAt).toISOString()
            : undefined,
        );
        formApi.reset(value);
        onSuccess(id);
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to save check-in."));
      }
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: onCancel,
  });
  return (
    <>
      <PageShell>
        <PageHeader title={title} />
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
          {initial ? (
            <FieldGroup className="grid grid-cols-1 sm:grid-cols-2">
              <form.Field name="occurredDate">
                {(field) => (
                  <ValidatedFormField field={field} label="Date" htmlFor="checkin-date" required>
                    {(control) => (
                      <Popover
                        open={dateOpen}
                        onOpenChange={(open) => {
                          setDateOpen(open);
                          if (!open) field.handleBlur();
                        }}
                      >
                        <PopoverTrigger
                          {...control}
                          render={
                            <Button variant="outline" className="justify-start font-normal" />
                          }
                        >
                          <CalendarDays data-icon="inline-start" />
                          {field.state.value
                            ? format(parseISO(field.state.value), "PPP")
                            : "Select date"}
                        </PopoverTrigger>
                        <PopoverContent className="w-auto p-0" align="start">
                          <Calendar
                            mode="single"
                            required
                            defaultMonth={parseISO(field.state.value)}
                            selected={parseISO(field.state.value)}
                            onSelect={(date) => {
                              field.handleChange(format(date, "yyyy-MM-dd"));
                              setDateOpen(false);
                            }}
                          />
                        </PopoverContent>
                      </Popover>
                    )}
                  </ValidatedFormField>
                )}
              </form.Field>
              <form.Field name="occurredTime">
                {(field) => (
                  <ValidatedFormField field={field} label="Time" htmlFor="checkin-time" required>
                    {(control) => (
                      <Input
                        {...control}
                        type="time"
                        step="1"
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </ValidatedFormField>
                )}
              </form.Field>
            </FieldGroup>
          ) : null}
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
        <QueryError title="Failed to Save Check-in" error={error} />
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
