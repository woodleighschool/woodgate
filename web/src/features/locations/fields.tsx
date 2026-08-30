import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Field, FieldGroup, FieldLabel, FieldTitle } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Switch } from "@components/ui/switch";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { GroupPicker } from "@features/directory/groups/group-picker";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { Location, LocationMutation } from "@lib/api";

import { EditableLocationImage, type LocationImageValue } from "./editable-image";

const schema = z.object({
  name: z.string().trim().min(1, "Name is required."),
  description: z.string(),
  enabled: z.boolean(),
  notes: z.boolean(),
  photo: z.boolean(),
  background: z.custom<LocationImageValue>(),
  logo: z.custom<LocationImageValue>(),
  group_ids: z.array(z.number()),
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
  const form = useForm({
    defaultValues: {
      name: initial?.name ?? "",
      description: initial?.description ?? "",
      enabled: initial?.enabled ?? true,
      notes: initial?.notes ?? false,
      photo: initial?.photo ?? false,
      background: locationImage(initial, "background"),
      logo: locationImage(initial, "logo"),
      group_ids: initial?.group_ids ?? [],
    } satisfies FormState,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(
        {
          name: value.name.trim(),
          description: value.description.trim(),
          enabled: value.enabled,
          notes: value.notes,
          photo: value.photo,
          background_object_id: mutationObjectID(value.background, initial?.background_object_id),
          logo_object_id: mutationObjectID(value.logo, initial?.logo_object_id),
          group_ids: value.group_ids,
        },
        { background: value.background, logo: value.logo },
      );
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
                  <Switch checked={field.state.value} onCheckedChange={field.handleChange} />
                  <FieldTitle>{label}</FieldTitle>
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
          <form.Field name="group_ids">
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
  locationID: number,
  images: LocationImages,
  uploadBackground: (value: { locationID: number; file: File }) => Promise<unknown>,
  uploadLogo: (value: { locationID: number; file: File }) => Promise<unknown>,
) {
  const uploads: Promise<unknown>[] = [];
  if (images.background.kind === "upload") {
    uploads.push(uploadBackground({ locationID, file: images.background.file }));
  }
  if (images.logo.kind === "upload") {
    uploads.push(uploadLogo({ locationID, file: images.logo.file }));
  }
  await Promise.all(uploads);
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

function mutationObjectID(value: LocationImageValue, currentID?: number): number | undefined {
  if (value.kind === "stored") return value.objectID;
  if (value.kind === "upload") return currentID;
  return undefined;
}

const LOCATION_TOGGLES = [
  { name: "enabled", label: "Enabled" },
  { name: "notes", label: "Collect notes" },
  { name: "photo", label: "Require a photo" },
] as const;
