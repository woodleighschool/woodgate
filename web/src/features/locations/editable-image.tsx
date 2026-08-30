import { Image, Upload, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from "@components/ui/attachment";
import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { Field, FieldLabel } from "@components/ui/field";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import { useLocationBackgrounds, useLocationLogos } from "@features/resources/queries";
import { cn } from "@lib/utils";

const IMAGE_ACCEPT = "image/png,image/jpeg,image/webp";

export type LocationImageValue =
  | { kind: "none" }
  | { kind: "stored"; objectID: number; filename: string; url: string }
  | { kind: "upload"; file: File };

type LocationImageKind = "background" | "logo";

export function EditableLocationImage({
  kind,
  value,
  onChange,
}: {
  kind: LocationImageKind;
  value: LocationImageValue;
  onChange: (value: LocationImageValue) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const label = kind === "background" ? "Background" : "Logo";
  const previewURL = useMemo(
    () => (value.kind === "upload" ? URL.createObjectURL(value.file) : ""),
    [value],
  );
  const displayURL = value.kind === "stored" ? value.url : previewURL;
  const hasImage = value.kind !== "none";

  useEffect(() => {
    if (!previewURL) return undefined;
    return () => URL.revokeObjectURL(previewURL);
  }, [previewURL]);

  function resetInput() {
    if (inputRef.current) inputRef.current.value = "";
  }

  return (
    <Field>
      <FieldLabel>{label}</FieldLabel>
      <div className="relative w-full">
        <input
          ref={inputRef}
          type="file"
          accept={IMAGE_ACCEPT}
          hidden
          onChange={(event) => {
            const next = event.target.files?.[0];
            resetInput();
            if (!next) return;
            onChange({ kind: "upload", file: next });
            setPickerOpen(false);
          }}
        />
        <Attachment state={hasImage ? "done" : "idle"} className="w-full">
          <AttachmentMedia variant="image" className="bg-transparent">
            <ImagePreview src={displayURL} kind={kind} />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>
              {value.kind === "upload"
                ? value.file.name
                : value.kind === "stored"
                  ? value.filename
                  : `Choose a ${label}`}
            </AttachmentTitle>
            <AttachmentDescription>
              {value.kind === "upload"
                ? `${formatBytes(value.file.size)} selected`
                : value.kind === "stored"
                  ? `Select to replace or choose another uploaded ${label.toLowerCase()}.`
                  : `Upload a new image or choose an uploaded ${label.toLowerCase()}.`}
            </AttachmentDescription>
          </AttachmentContent>
          {hasImage ? (
            <AttachmentActions>
              <AttachmentAction
                type="button"
                aria-label={`Remove ${label.toLowerCase()}`}
                onClick={() => {
                  onChange({ kind: "none" });
                  resetInput();
                }}
              >
                <X />
              </AttachmentAction>
            </AttachmentActions>
          ) : null}
          <AttachmentTrigger onClick={() => setPickerOpen(true)} />
        </Attachment>
      </div>
      <LocationImagePicker
        kind={kind}
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onUpload={() => inputRef.current?.click()}
        onPick={(object) => {
          onChange({ kind: "stored", ...object });
          setPickerOpen(false);
        }}
      />
    </Field>
  );
}

function LocationImagePicker({
  kind,
  open,
  onOpenChange,
  onUpload,
  onPick,
}: {
  kind: LocationImageKind;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpload: () => void;
  onPick: (object: { objectID: number; filename: string; url: string }) => void;
}) {
  const backgrounds = useLocationBackgrounds(open && kind === "background");
  const logos = useLocationLogos(open && kind === "logo");
  const query = kind === "background" ? backgrounds : logos;
  const items = query.data?.items ?? [];
  const label = kind === "background" ? "Background" : "Logo";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Choose {label}</DialogTitle>
        </DialogHeader>
        {items.length > 0 ? (
          <div className="grid max-h-80 grid-cols-[repeat(auto-fill,4rem)] gap-1 overflow-y-auto p-1">
            {items.map((object) => (
              <Tooltip key={object.id}>
                <TooltipTrigger
                  render={
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-lg"
                      className="size-16"
                      disabled={!object.content_url}
                      onClick={() =>
                        object.content_url
                          ? onPick({
                              objectID: object.id,
                              filename: object.filename,
                              url: object.content_url,
                            })
                          : undefined
                      }
                    />
                  }
                >
                  <ImagePreview src={object.content_url} kind={kind} className="size-14" />
                </TooltipTrigger>
                <TooltipContent>{object.filename}</TooltipContent>
              </Tooltip>
            ))}
          </div>
        ) : (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {query.isLoading ? `Loading ${label}s…` : `No ${label}s Uploaded Yet`}
          </p>
        )}
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
          <Button type="button" onClick={onUpload}>
            <Upload data-icon="inline-start" />
            Upload New
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ImagePreview({
  src,
  kind,
  className,
}: {
  src?: string;
  kind: LocationImageKind;
  className?: string;
}) {
  if (!src) {
    return <Image className={cn("size-5 text-muted-foreground", className)} aria-hidden="true" />;
  }
  return (
    <img
      src={src}
      alt=""
      className={cn(
        "block size-10 rounded-md",
        kind === "background" ? "object-cover" : "object-contain",
        className,
      )}
    />
  );
}

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / 1024 ** exponent).toFixed(exponent ? 1 : 0)} ${units[exponent]}`;
}
