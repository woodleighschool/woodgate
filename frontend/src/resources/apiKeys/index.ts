import { APIKeyCreate } from "@/resources/apiKeys/create";
import { APIKeyList } from "@/resources/apiKeys/list";
import { APIKeyShow } from "@/resources/apiKeys/show";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import type { ResourceProps } from "react-admin";

const apiKeys: Partial<ResourceProps> = {
  icon: VpnKeyIcon,
  options: { label: "API Keys" },
  recordRepresentation: "name",
  list: APIKeyList,
  show: APIKeyShow,
  create: APIKeyCreate,
};

export default apiKeys;
