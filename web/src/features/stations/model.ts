import type { Station } from "@lib/api";
import type { StatusMetadataMap } from "@lib/enum-metadata";

export const STATION_STATES = {
  offline: { name: "Offline" },
  online: { name: "Online", variant: "success" },
  incompatible: {
    name: "Incompatible",
    description: "This Station does not support the server's protocol version.",
    variant: "warning",
  },
} satisfies StatusMetadataMap<Station["state"]>;
