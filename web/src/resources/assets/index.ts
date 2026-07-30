import { AssetCreate } from "@/resources/assets/create";
import { AssetEdit } from "@/resources/assets/edit";
import { AssetList } from "@/resources/assets/list";
import { AssetShow } from "@/resources/assets/show";
import PermMediaIcon from "@mui/icons-material/PermMedia";
import type { ResourceProps } from "react-admin";

const assets: Partial<ResourceProps> = {
  icon: PermMediaIcon,
  recordRepresentation: "name",
  list: AssetList,
  show: AssetShow,
  create: AssetCreate,
  edit: AssetEdit,
};

export default assets;
