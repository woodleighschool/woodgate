import { QRCodeSVG } from "qrcode.react";

import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { Field, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import type { StationPairing } from "@lib/api";

export function StationPairingDialog({
  pairing,
  onDone,
}: {
  pairing?: StationPairing;
  onDone: () => void;
}) {
  return (
    <Dialog
      open={pairing !== undefined}
      onOpenChange={(open) => {
        if (!open) onDone();
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Pair {pairing?.station.name ?? "Station"}</DialogTitle>
          <DialogDescription>
            Scan the configuration QR code in WoodGate or with the device camera. The Station key is
            not shown again.
          </DialogDescription>
        </DialogHeader>
        {pairing ? (
          <Tabs defaultValue="qr">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="qr">QR Code</TabsTrigger>
              <TabsTrigger value="manual">Manual</TabsTrigger>
            </TabsList>
            <TabsContent value="qr" className="flex justify-center py-4">
              <QRCodeSVG value={pairing.url} size={280} level="M" marginSize={2} />
            </TabsContent>
            <TabsContent value="manual" className="py-2">
              <FieldGroup>
                <Field>
                  <FieldLabel htmlFor="station-pairing-server">Server URL</FieldLabel>
                  <Input id="station-pairing-server" value={pairing.server_url} readOnly />
                </Field>
                <Field>
                  <FieldLabel htmlFor="station-pairing-key">Station Key</FieldLabel>
                  <Input
                    id="station-pairing-key"
                    value={pairing.key}
                    readOnly
                    className="font-mono"
                  />
                </Field>
              </FieldGroup>
            </TabsContent>
          </Tabs>
        ) : null}
        <DialogFooter>
          <Button onClick={onDone}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
