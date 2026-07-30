import { LocationCreate } from "@/resources/locations/create";
import { LocationEdit } from "@/resources/locations/edit";
import { LocationList } from "@/resources/locations/list";
import PlaceIcon from "@mui/icons-material/Place";
import type { ResourceProps } from "react-admin";

const locations: Partial<ResourceProps> = {
  icon: PlaceIcon,
  recordRepresentation: "name",
  list: LocationList,
  create: LocationCreate,
  edit: LocationEdit,
};

export default locations;
