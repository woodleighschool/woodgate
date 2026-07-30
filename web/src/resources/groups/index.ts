import { GroupList } from "@/resources/groups/list";
import { GroupShow } from "@/resources/groups/show";
import GroupsIcon from "@mui/icons-material/Groups";
import type { ResourceProps } from "react-admin";

const groups: Partial<ResourceProps> = {
  icon: GroupsIcon,
  recordRepresentation: "name",
  list: GroupList,
  show: GroupShow,
};

export default groups;
