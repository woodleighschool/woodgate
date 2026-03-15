import { AccessTab } from "@/resources/shared/accessTab";
import {
  Alert,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Typography,
} from "@mui/material";
import { QRCodeSVG } from "qrcode.react";
import type { ReactElement } from "react";
import { useState } from "react";
import {
  CanAccess,
  DateField,
  DeleteButton,
  Labeled,
  ListButton,
  Show,
  TabbedShowLayout,
  TextField,
  TopToolbar,
  useCanAccess,
  useGetList,
  useNotify,
  useRecordContext,
  useUpdate,
} from "react-admin";
import { useLocation } from "react-router-dom";

// Minimum grants the app needs:
//   - locations:read       — list and get locations
//   - users:read           — list users for a selected location roster
//   - assets:read(asset)   — fetch reusable location branding assets
//   - checkins:create      — submit check-ins (requires a specific location_id)
const appKeyGrants = (
  locationId: string,
): { resource: string; action: string; location_id?: string; asset_type?: "asset" | "photo" }[] => [
  { resource: "locations", action: "read" },
  { resource: "users", action: "read" },
  { resource: "assets", action: "read", asset_type: "asset" },
  { resource: "checkins", action: "create", location_id: locationId },
];

interface APIKeyShowState {
  baseUrl?: string;
  secret?: string;
}

const APIKeyShowActions = (): ReactElement => (
  <TopToolbar>
    <ListButton />
    <CanAccess action="delete" resource="api-keys">
      <DeleteButton mutationMode="pessimistic" />
    </CanAccess>
  </TopToolbar>
);

export const APIKeyShow = (): ReactElement => <APIKeyShowBody />;

const PairingQRButton = ({ secret, baseUrl }: { secret: string; baseUrl: string }): ReactElement => {
  const record = useRecordContext<{ id: string }>();
  const notify = useNotify();
  const [update, { isPending: applying }] = useUpdate();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [qrOpen, setQROpen] = useState(false);
  const [locationId, setLocationId] = useState("");
  const qrPayload = JSON.stringify({ api_key: secret, base_url: baseUrl });

  const { data: locations = [], isPending: locationsLoading } = useGetList<{ id: string; name: string }>(
    "locations",
    { pagination: { page: 1, perPage: 250 }, sort: { field: "name", order: "ASC" }, filter: { enabled: true } },
    { enabled: confirmOpen },
  );

  const handleApply = async (): Promise<void> => {
    if (!record?.id || !locationId) return;
    await update(
      "api-keys",
      { id: record.id, data: { admin: false, access: appKeyGrants(locationId) }, previousData: {} },
      {
        onSuccess: (): void => {
          setConfirmOpen(false);
          setQROpen(true);
        },
        onError: (): void => {
          notify("Could not apply app permissions", { type: "error" });
        },
      },
    );
  };

  return (
    <>
      <Button
        variant="outlined"
        size="small"
        onClick={(): void => {
          setConfirmOpen(true);
        }}
        disabled={!record?.id}
      >
        Show Pairing QR
      </Button>

      <Dialog
        open={confirmOpen}
        onClose={(): void => {
          setConfirmOpen(false);
        }}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>Apply App Permissions</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ pt: 1 }}>
            <DialogContentText>
              This will set the key&apos;s permissions to the minimum required by the app: read locations, read users
              for the selected location, read reusable assets, and create check-ins for that location.
            </DialogContentText>
            <FormControl fullWidth size="small" disabled={locationsLoading || applying}>
              <InputLabel id="location-label">Location</InputLabel>
              <Select
                labelId="location-label"
                label="Location"
                value={locationId}
                onChange={(event): void => {
                  setLocationId(event.target.value);
                }}
              >
                {locations.map(
                  (loc): ReactElement => (
                    <MenuItem key={loc.id} value={loc.id}>
                      {loc.name}
                    </MenuItem>
                  ),
                )}
              </Select>
            </FormControl>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={(): void => {
              setConfirmOpen(false);
            }}
            disabled={applying}
          >
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={(): void => {
              handleApply().catch((): undefined => undefined);
            }}
            disabled={!locationId || applying}
            startIcon={applying ? <CircularProgress size={16} /> : undefined}
          >
            Apply & Show QR
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={qrOpen}
        onClose={(): void => {
          setQROpen(false);
        }}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Pairing QR</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ alignItems: "center", py: 1 }}>
            <QRCodeSVG value={qrPayload} size={280} marginSize={4} />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={(): void => {
              setQROpen(false);
            }}
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

const APIKeyShowBody = (): ReactElement => {
  const { canAccess: canWriteAPIKeys } = useCanAccess({ action: "write", resource: "api-keys" });
  const location = useLocation();
  const state = location.state as APIKeyShowState | undefined;
  const baseUrl = state?.baseUrl;
  const secret = state?.secret;

  return (
    <Show actions={<APIKeyShowActions />}>
      <TabbedShowLayout>
        <TabbedShowLayout.Tab label="Overview">
          {secret ? (
            <Alert severity="success">
              API key created. Copy it now; the secret will not be shown again after this session.
            </Alert>
          ) : undefined}
          <TextField source="id" />
          <TextField source="name" label="Name" />
          <TextField source="key_prefix" label="Prefix" />
          {secret && baseUrl ? (
            <Labeled label="Secret">
              <Stack direction="row" spacing={2} alignItems="center" useFlexGap flexWrap="wrap">
                <Typography component="code" sx={{ wordBreak: "break-all" }}>
                  {secret}
                </Typography>
                <PairingQRButton secret={secret} baseUrl={baseUrl} />
              </Stack>
            </Labeled>
          ) : undefined}
          <DateField source="last_used_at" label="Last Used" showTime />
          <DateField source="expires_at" label="Expires" showTime />
          <DateField source="created_at" label="Created" showTime />
        </TabbedShowLayout.Tab>
        {canWriteAPIKeys ? (
          <TabbedShowLayout.Tab label="Access">
            <AccessTab resource="api-keys" />
          </TabbedShowLayout.Tab>
        ) : undefined}
      </TabbedShowLayout>
    </Show>
  );
};
