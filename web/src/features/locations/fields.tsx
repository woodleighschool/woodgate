import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useState } from "react";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Switch } from "@components/ui/switch";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { GroupPicker } from "@features/directory/groups/group-picker";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { GroupSummary, Location, LocationMutation } from "@lib/api";

import { EditableLocationImage, type LocationImageValue } from "./editable-image";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required."),
  description: z.string(),
  enabled: z.boolean(),
  notes: z.boolean(),
  photo: z.boolean(),
  background: z.custom<LocationImageValue>(),
  logo: z.custom<LocationImageValue>(),
  groups: z.custom<GroupSummary[]>(),
});

type FormState = z.infer<typeof schema>;

export function LocationForm({
  title,
  initial,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  title: string;
  initial?: Location;
  onSubmit: (body: LocationMutation, images: LocationImages) => Promise<number>;
  onSuccess: (id: number) => void;
  onCancel: () => void;
}) {
  const [error, setError] = useState<Error | null>(null);
  const form = useForm({
    defaultValues: {
      name: initial?.name ?? "",
      description: initial?.description ?? "",
      enabled: initial?.enabled ?? true,
      notes: initial?.notes ?? false,
      photo: initial?.photo ?? false,
      background: locationImage(initial, "background"),
      logo: locationImage(initial, "logo"),
      groups: initial?.groups ?? [],
    } satisfies FormState,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      setError(null);
      try {
        const id = await onSubmit(
          {
            name: value.name.trim(),
            description: value.description.trim(),
            enabled: value.enabled,
            notes: value.notes,
            photo: value.photo,
            group_ids: value.groups.map((group) => group.id),
          },
          { background: value.background, logo: value.logo },
        );
        formApi.reset(value);
        onSuccess(id);
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to save location."));
      }
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
              <ValidatedFormField field={field} label="Name" htmlFor="location-name" required>
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
          <form.Field name="description">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="location-description">Description</FieldLabel>
                <Textarea
                  id="location-description"
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                />
              </Field>
            )}
          </form.Field>
          {LOCATION_TOGGLES.map(({ name, label }) => (
            <form.Field key={name} name={name}>
              {(field) => (
                <Field orientation="horizontal">
                  <Switch
                    id={`location-${name}`}
                    checked={field.state.value}
                    onCheckedChange={field.handleChange}
                  />
                  <FieldLabel htmlFor={`location-${name}`}>{label}</FieldLabel>
                </Field>
              )}
            </form.Field>
          ))}
          <form.Field name="background">
            {(field) => (
              <EditableLocationImage
                kind="background"
                value={field.state.value}
                onChange={field.handleChange}
              />
            )}
          </form.Field>
          <form.Field name="logo">
            {(field) => (
              <EditableLocationImage
                kind="logo"
                value={field.state.value}
                onChange={field.handleChange}
              />
            )}
          </form.Field>
          <form.Field name="groups">
            {(field) => (
              <Field>
                <FieldLabel htmlFor="location-groups">Directory Groups</FieldLabel>
                <GroupPicker
                  id="location-groups"
                  value={field.state.value}
                  onChange={field.handleChange}
                />
              </Field>
            )}
          </form.Field>
        </FieldGroup>
        <QueryError error={error} />
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

export interface LocationImages {
  background: LocationImageValue;
  logo: LocationImageValue;
}

export async function uploadLocationImages(
  images: LocationImages,
  uploadBackground: (value: { file: File }) => Promise<number>,
  uploadLogo: (value: { file: File }) => Promise<number>,
): Promise<Pick<LocationMutation, "background_object_id" | "logo_object_id">> {
  const [backgroundID, logoID] = await Promise.all([
    imageObjectID(images.background, uploadBackground),
    imageObjectID(images.logo, uploadLogo),
  ]);
  return { background_object_id: backgroundID, logo_object_id: logoID };
}

async function imageObjectID(
  value: LocationImageValue,
  upload: (value: { file: File }) => Promise<number>,
): Promise<number | undefined> {
  if (value.kind === "stored") return value.objectID;
  if (value.kind === "upload") return upload({ file: value.file });
  return undefined;
}

function locationImage(
  location: Location | undefined,
  kind: "background" | "logo",
): LocationImageValue {
  const objectID = location?.[`${kind}_object_id`];
  const file = location?.[`${kind}_file`];
  const url = location?.[`${kind}_url`];
  return objectID && file && url
    ? { kind: "stored", objectID, filename: file.filename, url }
    : { kind: "none" };
}

const LOCATION_TOGGLES = [
  { name: "enabled", label: "Enabled" },
  { name: "notes", label: "Collect notes" },
  { name: "photo", label: "Require a photo" },
] as const;
