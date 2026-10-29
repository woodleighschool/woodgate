import { revalidateLogic, useForm } from "@tanstack/react-form";
import { permissionLevel } from "@woodleighschool/authz";
import { useState } from "react";
import { z } from "zod";

import { FormActions } from "@components/form-actions";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Checkbox } from "@components/ui/checkbox";
import { FieldGroup, FieldLegend, FieldSet } from "@components/ui/field";
import { Input } from "@components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { useAuthzResources } from "@features/resources/queries";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { AuthzRole, AuthzRoleMutation } from "@lib/api";

const schema = z.object({
  key: z.string().trim().min(1, "Key is required."),
  name: z.string().trim().min(1, "Name is required."),
  description: z.string(),
  permissions: z.record(z.string(), z.enum(["none", "view", "edit"])),
});

export function RoleForm({
  title,
  initial,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  title: string;
  initial?: AuthzRole;
  onSubmit: (body: AuthzRoleMutation) => Promise<number>;
  onSuccess: (id: number) => void;
  onCancel: () => void;
}) {
  const resources = useAuthzResources();
  const [error, setError] = useState<Error | null>(null);
  const form = useForm({
    defaultValues: {
      key: initial?.key ?? "",
      name: initial?.name ?? "",
      description: initial?.description ?? "",
      permissions: Object.fromEntries(
        Object.entries(initial?.permissions ?? {}).map(([resource, level]) => [
          resource,
          permissionLevel(level),
        ]),
      ),
    },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      setError(null);
      try {
        const id = await onSubmit({
          key: initial ? undefined : value.key.trim(),
          name: value.name.trim(),
          description: value.description.trim(),
          permissions: value.permissions,
        });
        formApi.reset(value);
        onSuccess(id);
      } catch (cause) {
        setError(cause instanceof Error ? cause : new Error("Unable to save role."));
      }
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel });
  return (
    <>
      <PageShell>
        <PageHeader title={title} />
        <FieldGroup className="max-w-3xl">
          {!initial ? (
            <form.Field name="key">
              {(field) => (
                <ValidatedFormField field={field} label="Key" htmlFor="role-key" required>
                  {(control) => (
                    <Input
                      {...control}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) =>
                        field.handleChange(event.target.value.toLowerCase().replaceAll(" ", "-"))
                      }
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
          ) : null}
          <form.Field name="name">
            {(field) => (
              <ValidatedFormField field={field} label="Name" htmlFor="role-name" required>
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
              <ValidatedFormField field={field} label="Description" htmlFor="role-description">
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
          <form.Field name="permissions">
            {(field) => (
              <FieldSet className="gap-2">
                <FieldLegend variant="label">Resource Access</FieldLegend>
                <QueryError
                  title="Failed to Load Resources"
                  error={resources.error}
                  onRetry={() => void resources.refetch()}
                />
                <div className="overflow-hidden rounded-xl border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Resource</TableHead>
                        <TableHead className="w-20 text-center">View</TableHead>
                        <TableHead className="w-20 text-center">Edit</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(resources.data ?? []).map((resource) => {
                        const level = permissionLevel(field.state.value[resource.resource]);
                        return (
                          <TableRow key={resource.resource}>
                            <TableHead scope="row" className="h-9">
                              {resource.display_name}
                            </TableHead>
                            <TableCell>
                              <div className="flex justify-center">
                                <Checkbox
                                  aria-label={`View ${resource.display_name}`}
                                  checked={level === "view" || level === "edit"}
                                  onCheckedChange={(checked) =>
                                    field.handleChange({
                                      ...field.state.value,
                                      [resource.resource]: !checked
                                        ? "none"
                                        : level === "edit"
                                          ? "edit"
                                          : "view",
                                    })
                                  }
                                />
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex justify-center">
                                <Checkbox
                                  aria-label={`Edit ${resource.display_name}`}
                                  checked={level === "edit"}
                                  onCheckedChange={(checked) =>
                                    field.handleChange({
                                      ...field.state.value,
                                      [resource.resource]: checked ? "edit" : "view",
                                    })
                                  }
                                />
                              </div>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </div>
              </FieldSet>
            )}
          </form.Field>
        </FieldGroup>
        <QueryError title="Failed to Save Role" error={error} />
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
