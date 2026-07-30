import { CheckinList } from "@/resources/checkins/list";
import { CheckinShow } from "@/resources/checkins/show";
import FactCheckIcon from "@mui/icons-material/FactCheck";
import type { ResourceProps } from "react-admin";

const checkins: Partial<ResourceProps> = {
  icon: FactCheckIcon,
  recordRepresentation: "id",
  list: CheckinList,
  show: CheckinShow,
};

export default checkins;
