import type { Access } from "@woodleighschool/authz";
import { permissionLevel } from "@woodleighschool/authz";
import { useState } from "react";

import { AsyncButton } from "@components/async-button";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Button } from "@components/ui/button";
import { Checkbox } from "@components/ui/checkbox";
import { Field, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@components/ui/field";
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
import { useAuthzResources } from "@features/resources/queries";
import type { AuthzRole, AuthzRoleMutation } from "@lib/api";

export function RoleForm({
  title,
  initial,
  pending,
  onSubmit,
  onCancel,
}: {
  title: string;
  initial?: AuthzRole;
  pending: boolean;
  onSubmit: (body: AuthzRoleMutation) => void;
  onCancel: () => void;
}) {
  const resources = useAuthzResources();
  const [key, setKey] = useState(initial?.key ?? "");
  const [name, setName] = useState(initial?.name ?? "");
  const [description, setDescription] = useState(initial?.description ?? "");
  const [permissions, setPermissions] = useState<Record<string, Access>>(
    Object.fromEntries(
      Object.entries(initial?.permissions ?? {}).map(([resource, level]) => [
        resource,
        permissionLevel(level),
      ]),
    ),
  );
  return (
    <PageShell>
      <PageHeader title={title} />
      <FieldGroup className="max-w-3xl">
        {!initial ? (
          <Field>
            <FieldLabel htmlFor="role-key">Key</FieldLabel>
            <Input
              id="role-key"
              required
              value={key}
              onChange={(event) => setKey(event.target.value.toLowerCase().replaceAll(" ", "-"))}
            />
          </Field>
        ) : null}
        <Field>
          <FieldLabel htmlFor="role-name">Name</FieldLabel>
          <Input
            id="role-name"
            required
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="role-description">Description</FieldLabel>
          <Textarea
            id="role-description"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>
        <FieldSet className="gap-2">
          <FieldLegend variant="label">Resource Access</FieldLegend>
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
                  const level = permissionLevel(permissions[resource.resource]);
                  return (
                    <TableRow key={resource.resource}>
                      <TableHead scope="row" className="h-9">
                        {resource.display_name}
                      </TableHead>
                      <TableCell>
                        <div className="flex justify-center">
                          <Checkbox
                            checked={level === "view" || level === "edit"}
                            onCheckedChange={(checked) =>
                              setPermissions((current) => ({
                                ...current,
                                [resource.resource]: !checked
                                  ? "none"
                                  : current[resource.resource] === "edit"
                                    ? "edit"
                                    : "view",
                              }))
                            }
                          />
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-center">
                          <Checkbox
                            checked={level === "edit"}
                            onCheckedChange={(checked) =>
                              setPermissions((current) => ({
                                ...current,
                                [resource.resource]: checked ? "edit" : "view",
                              }))
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
      </FieldGroup>
      <Field orientation="horizontal">
        <AsyncButton
          size="sm"
          isPending={pending}
          disabled={!name.trim() || (!initial && !key.trim())}
          onClick={() =>
            onSubmit({
              key: initial ? undefined : key.trim(),
              name: name.trim(),
              description: description.trim(),
              permissions,
            })
          }
        >
          {initial ? "Save" : "Create"}
        </AsyncButton>
        <Button size="sm" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </Field>
    </PageShell>
  );
}
