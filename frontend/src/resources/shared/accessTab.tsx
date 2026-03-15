import { invalidateAuthMe } from "@/api/auth";
import type { AssetType, PermissionAction, PermissionGrant, PermissionResource } from "@/api/types";
import {
  Alert,
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  Paper,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import type { ReactElement } from "react";
import { useState } from "react";
import { useGetList, useNotify, useRecordContext, useRefresh, useUpdate } from "react-admin";

type ParentResource = "users" | "api-keys";

interface AccessRecord {
  id: string;
  admin?: boolean;
  access?: PermissionGrant[];
}

interface LocationRecord {
  id: string;
  name: string;
}

const actions: PermissionAction[] = ["read", "create", "write", "delete"];

const globalAccessRows: { label: string; resource: PermissionResource }[] = [
  { label: "Users", resource: "users" },
  { label: "Groups", resource: "groups" },
  { label: "Locations", resource: "locations" },
  { label: "API keys", resource: "api_keys" },
];

const assetTypeRows: { label: string; assetType: AssetType }[] = [
  { label: "Assets", assetType: "asset" },
  { label: "Photos", assetType: "photo" },
];

const locationRecordsForAccess = (
  grants: PermissionGrant[] | undefined,
  locations: LocationRecord[],
): LocationRecord[] => {
  const items = new Map(locations.map((location): [string, LocationRecord] => [location.id, location]));

  for (const grant of grants ?? []) {
    if (grant.resource !== "checkins" || !grant.location_id || items.has(grant.location_id)) {
      continue;
    }

    items.set(grant.location_id, {
      id: grant.location_id,
      name: grant.location_id,
    });
  }

  return [...items.values()].toSorted((left, right): number => left.name.localeCompare(right.name));
};

const normalizeAccess = (grants: PermissionGrant[] | undefined): PermissionGrant[] =>
  (grants ?? []).toSorted((left, right): number => {
    const leftKey = `${left.resource}:${left.location_id ?? ""}:${left.asset_type ?? ""}:${left.action}`;
    const rightKey = `${right.resource}:${right.location_id ?? ""}:${right.asset_type ?? ""}:${right.action}`;
    return leftKey.localeCompare(rightKey);
  });

const accessSignature = (admin: boolean, grants: PermissionGrant[] | undefined): string =>
  JSON.stringify({ admin, access: normalizeAccess(grants) });

const hasGrant = (
  grants: PermissionGrant[],
  resource: PermissionResource,
  action: PermissionAction,
  locationId?: string,
  assetType?: AssetType,
): boolean =>
  grants.some(
    (grant): boolean =>
      grant.resource === resource &&
      grant.action === action &&
      (grant.location_id ?? undefined) === locationId &&
      (grant.asset_type ?? undefined) === assetType,
  );

const toggleGrant = (
  grants: PermissionGrant[],
  resource: PermissionResource,
  action: PermissionAction,
  enabled: boolean,
  locationId?: string,
  assetType?: AssetType,
): PermissionGrant[] => {
  const nextGrants = grants.filter(
    (grant): boolean =>
      !(
        grant.resource === resource &&
        grant.action === action &&
        (grant.location_id ?? undefined) === locationId &&
        (grant.asset_type ?? undefined) === assetType
      ),
  );
  if (!enabled) {
    return normalizeAccess(nextGrants);
  }

  return normalizeAccess([
    ...nextGrants,
    {
      resource,
      action,
      ...(locationId ? { location_id: locationId } : {}),
      ...(assetType ? { asset_type: assetType } : {}),
    },
  ]);
};

const AccessSection = ({
  description,
  rows,
  title,
}: {
  description: string;
  rows: ReactElement[];
  title: string;
}): ReactElement => (
  <Paper variant="outlined" sx={{ overflow: "hidden" }}>
    <Box sx={{ px: 2.5, py: 2, borderBottom: 1, borderColor: "divider", backgroundColor: "action.hover" }}>
      <Typography variant="h6">{title}</Typography>
      <Typography variant="body2" color="text.secondary">
        {description}
      </Typography>
    </Box>
    <Table size="small">
      <TableHead>
        <TableRow>
          <TableCell>Resource</TableCell>
          <TableCell align="center">Read</TableCell>
          <TableCell align="center">Create</TableCell>
          <TableCell align="center">Write</TableCell>
          <TableCell align="center">Delete</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>{rows}</TableBody>
    </Table>
  </Paper>
);

const AccessRow = ({
  label,
  locationId,
  onToggle,
  resource,
  grants,
  assetType,
}: {
  label: string;
  locationId?: string;
  onToggle: (action: PermissionAction, enabled: boolean, locationId?: string, assetType?: AssetType) => void;
  resource: PermissionResource;
  grants: PermissionGrant[];
  assetType?: AssetType;
}): ReactElement => (
  <TableRow hover>
    <TableCell component="th" scope="row">
      {label}
    </TableCell>
    {actions.map(
      (action): ReactElement => (
        <TableCell key={action} align="center">
          <Checkbox
            checked={hasGrant(grants, resource, action, locationId, assetType)}
            onChange={(_event, checked): void => {
              onToggle(action, checked, locationId, assetType);
            }}
          />
        </TableCell>
      ),
    )}
  </TableRow>
);

const AccessEditor = ({
  initialAccess,
  initialAdmin,
  locations,
  resource,
  record,
}: {
  initialAccess: PermissionGrant[];
  initialAdmin: boolean;
  locations: LocationRecord[];
  record: AccessRecord;
  resource: ParentResource;
}): ReactElement => {
  const notify = useNotify();
  const refresh = useRefresh();
  const [update, { isPending }] = useUpdate();
  const [admin, setAdmin] = useState<boolean>(initialAdmin);
  const [access, setAccess] = useState<PermissionGrant[]>(initialAccess);

  const isDirty = accessSignature(admin, access) !== accessSignature(Boolean(record.admin), record.access);

  const save = async (): Promise<void> => {
    await update(
      resource,
      {
        id: record.id,
        data: { admin, access },
        previousData: record,
      },
      {
        onSuccess: (): void => {
          invalidateAuthMe();
          notify("Access updated.", { type: "success" });
          refresh();
        },
        onError: (error): void => {
          notify(error instanceof Error ? error.message : "Failed to update access.", { type: "error" });
        },
      },
    );
  };

  const globalRows = globalAccessRows.map(
    ({ label, resource: permissionResource }): ReactElement => (
      <AccessRow
        key={permissionResource}
        label={label}
        resource={permissionResource}
        grants={access}
        onToggle={(action: PermissionAction, enabled: boolean): void => {
          setAccess((current): PermissionGrant[] => toggleGrant(current, permissionResource, action, enabled));
        }}
      />
    ),
  );

  const checkinRows = locations.map(
    (location): ReactElement => (
      <AccessRow
        key={location.id}
        label={location.name}
        locationId={location.id}
        resource="checkins"
        grants={access}
        onToggle={(action: PermissionAction, enabled: boolean, nextLocationId?: string): void => {
          setAccess((current): PermissionGrant[] => toggleGrant(current, "checkins", action, enabled, nextLocationId));
        }}
      />
    ),
  );

  const assetRows = assetTypeRows.map(
    ({ label, assetType }): ReactElement => (
      <AccessRow
        key={assetType}
        label={label}
        assetType={assetType}
        resource="assets"
        grants={access}
        onToggle={(
          action: PermissionAction,
          enabled: boolean,
          _locationId?: string,
          nextAssetType?: AssetType,
        ): void => {
          setAccess((current): PermissionGrant[] =>
            toggleGrant(current, "assets", action, enabled, undefined, nextAssetType),
          );
        }}
      />
    ),
  );

  return (
    <Stack spacing={2.5} sx={{ pt: 1 }}>
      <Alert severity="info">
        Admin bypasses all individual grants. Direct access grants control which resources and actions are available in
        the UI and API.
      </Alert>
      <Paper variant="outlined" sx={{ p: 2.5 }}>
        <FormControlLabel
          control={
            <Switch
              checked={admin}
              onChange={(_event, checked): void => {
                setAdmin(checked);
              }}
            />
          }
          label="Administrator"
        />
      </Paper>
      <AccessSection title="Admin resources" description="Global access for the admin surface." rows={globalRows} />
      <AccessSection
        title="Assets by type"
        description="Type-scoped access for reusable assets and check-in photos."
        rows={assetRows}
      />
      {checkinRows.length > 0 ? (
        <AccessSection
          title="Check-ins by location"
          description="Location-scoped access for the check-in workflow."
          rows={checkinRows}
        />
      ) : (
        <Paper variant="outlined" sx={{ p: 2.5 }}>
          <Typography variant="h6">Check-ins by location</Typography>
          <Typography variant="body2" color="text.secondary">
            No locations are available to assign.
          </Typography>
        </Paper>
      )}
      <Box sx={{ display: "flex", justifyContent: "flex-end" }}>
        <Button
          variant="contained"
          disabled={!isDirty || isPending}
          onClick={(): void => {
            save().catch((): undefined => undefined);
          }}
        >
          Save access
        </Button>
      </Box>
    </Stack>
  );
};

export const AccessTab = ({ resource }: { resource: ParentResource }): ReactElement => {
  const record = useRecordContext<AccessRecord>();
  const {
    data: locations = [],
    error,
    isPending,
  } = useGetList<LocationRecord>("locations", {
    pagination: { page: 1, perPage: 200 },
    sort: { field: "name", order: "ASC" },
    filter: {},
  });

  if (!record) {
    return <Typography color="text.secondary">Loading access...</Typography>;
  }

  if (isPending) {
    return <Typography color="text.secondary">Loading locations...</Typography>;
  }

  const resolvedLocations = locationRecordsForAccess(record.access, locations);
  const editorKey = accessSignature(Boolean(record.admin), record.access);
  return (
    <Stack spacing={2.5} sx={{ pt: 1 }}>
      {error ? (
        <Alert severity="warning">
          Location names could not be loaded. Existing location-scoped access is still shown by ID.
        </Alert>
      ) : undefined}
      <AccessEditor
        key={editorKey}
        initialAdmin={Boolean(record.admin)}
        initialAccess={normalizeAccess(record.access)}
        locations={resolvedLocations}
        record={record}
        resource={resource}
      />
    </Stack>
  );
};
