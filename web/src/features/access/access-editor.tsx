import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";

import { AsyncButton } from "@components/async-button";
import { QueryError } from "@components/query-error";
import { Alert, AlertDescription } from "@components/ui/alert";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@components/ui/card";
import { Checkbox } from "@components/ui/checkbox";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Switch } from "@components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { toast } from "@components/ui/toast";
import { useCanAccess } from "@features/auth/access";
import {
  patchApiKey,
  patchUser,
  unwrap,
  type PermissionGrant,
  type PermissionResource,
} from "@lib/api";

import {
  accessSignature,
  hasGrant,
  locationsForAccess,
  permissionActions,
  toggleGrant,
} from "./grants";
import { loadAccessLocations } from "./locations";

interface AccessRecord {
  id: string;
  admin?: boolean;
  access?: PermissionGrant[];
}

const globalResources: { label: string; resource: PermissionResource }[] = [
  { label: "Users", resource: "users" },
  { label: "Groups", resource: "groups" },
  { label: "Locations", resource: "locations" },
  { label: "Assets", resource: "assets" },
  { label: "Check-ins", resource: "checkins" },
  { label: "API keys", resource: "api_keys" },
];

export function AccessEditor({
  resource,
  record,
}: {
  resource: "users" | "api-keys";
  record: AccessRecord;
}) {
  const canReadLocations = useCanAccess("locations", "read");
  const locations = useQuery({
    queryKey: ["locations", "access"],
    queryFn: ({ signal }) => loadAccessLocations(signal),
    enabled: canReadLocations,
  });

  return (
    <AccessForm
      key={`${record.id}:${accessSignature(Boolean(record.admin), record.access)}`}
      resource={resource}
      record={record}
      locations={locationsForAccess(record.access ?? [], locations.data ?? [])}
      locationsUnavailable={Boolean(locations.error) || !canReadLocations}
      locationsLoading={locations.isLoading}
    />
  );
}

function AccessForm({
  resource,
  record,
  locations,
  locationsUnavailable,
  locationsLoading,
}: {
  resource: "users" | "api-keys";
  record: AccessRecord;
  locations: { id: string; name: string }[];
  locationsUnavailable: boolean;
  locationsLoading: boolean;
}) {
  const queryClient = useQueryClient();
  const [admin, setAdmin] = useState(Boolean(record.admin));
  const [access, setAccess] = useState(record.access ?? []);
  const canWrite = useCanAccess(resource, "write");
  const save = useMutation({
    mutationFn: async () => {
      const options = { path: { id: record.id }, body: { admin, access } };
      if (resource === "users") return unwrap(patchUser(options));
      return unwrap(patchApiKey(options));
    },
    onSuccess: async () => {
      toast.add({ title: "Access updated", type: "success" });
      await queryClient.invalidateQueries();
    },
  });
  const dirty =
    accessSignature(admin, access) !== accessSignature(Boolean(record.admin), record.access);
  const disabled = !canWrite || save.isPending;
  const row = (label: string, scope: Omit<PermissionGrant, "action">) => (
    <TableRow key={`${scope.resource}:${scope.location_id ?? ""}:${scope.asset_type ?? ""}`}>
      <TableHead scope="row">{label}</TableHead>
      {permissionActions.map((action) => {
        const target = { ...scope, action };
        return (
          <TableCell key={action}>
            <Checkbox
              aria-label={`${label}: ${action}`}
              checked={hasGrant(access, target)}
              disabled={disabled}
              onCheckedChange={(checked) =>
                setAccess((current) => toggleGrant(current, target, checked))
              }
            />
          </TableCell>
        );
      })}
    </TableRow>
  );

  return (
    <div className="flex flex-col gap-6">
      <Alert>
        <AlertDescription>
          Admin bypasses all individual grants. Global grants apply across the whole resource.
          Scoped asset-type and location grants only matter when the matching global grant is not
          enabled.
        </AlertDescription>
      </Alert>
      <FieldGroup>
        <Field orientation="horizontal">
          <Switch
            id={`admin-${record.id}`}
            checked={admin}
            onCheckedChange={setAdmin}
            disabled={disabled}
          />
          <FieldLabel htmlFor={`admin-${record.id}`}>Administrator</FieldLabel>
        </Field>
      </FieldGroup>
      <AccessSection
        title="Global access"
        description="Resource-wide access for the admin UI and API."
      >
        {globalResources.map(({ label, resource: permissionResource }) =>
          row(label, { resource: permissionResource }),
        )}
      </AccessSection>
      <AccessSection
        title="Assets by type"
        description="Type-scoped access for reusable assets and check-in photos."
      >
        {row("Assets", { resource: "assets", asset_type: "asset" })}
        {row("Photos", { resource: "assets", asset_type: "photo" })}
      </AccessSection>
      {locationsUnavailable ? (
        <Alert>
          <AlertDescription>
            Location names could not be loaded. Existing location-scoped access is still shown by
            ID.
          </AlertDescription>
        </Alert>
      ) : null}
      <AccessSection
        title="Check-ins by location"
        description="Location-scoped access for the check-in workflow."
      >
        {locations.map((location) =>
          row(location.name, { resource: "checkins", location_id: location.id }),
        )}
        {locations.length === 0 ? (
          <TableRow>
            <TableCell colSpan={5}>
              {locationsLoading ? "Loading locations…" : "No locations are available to assign."}
            </TableCell>
          </TableRow>
        ) : null}
      </AccessSection>
      <QueryError title="Could not update access" error={save.error} />
      {canWrite ? (
        <AsyncButton
          className="w-fit"
          isPending={save.isPending}
          disabled={!dirty}
          onClick={() => save.mutate()}
        >
          Save access
        </AsyncButton>
      ) : null}
    </div>
  );
}

function AccessSection({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Resource</TableHead>
              {permissionActions.map((action) => (
                <TableHead key={action} className="capitalize">
                  {action}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>{children}</TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
