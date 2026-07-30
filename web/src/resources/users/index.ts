import PeopleIcon from "@mui/icons-material/People";
import type { ResourceProps } from "react-admin";

import { UserList } from "@/resources/users/list";
import { UserShow } from "@/resources/users/show";

const users: Partial<ResourceProps> = {
  icon: PeopleIcon,
  recordRepresentation: "display_name",
  list: UserList,
  show: UserShow,
};

export default users;
