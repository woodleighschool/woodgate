import MicrosoftIcon from "@mui/icons-material/Microsoft";
import StorageOutlinedIcon from "@mui/icons-material/StorageOutlined";
import { Tooltip } from "@mui/material";
import type { ReactElement } from "react";
import { useRecordContext } from "react-admin";

interface SourceFieldProperties {
  source?: string;
}

export const SourceField = ({ source = "source" }: SourceFieldProperties): ReactElement => {
  const record = useRecordContext<Record<string, unknown>>();
  const value = typeof record?.[source] === "string" ? record[source] : undefined;

  if (!record) {
    return <></>;
  }

  if (value === "entra") {
    return (
      <Tooltip title="Entra">
        <MicrosoftIcon fontSize="small" color="action" />
      </Tooltip>
    );
  }

  return (
    <Tooltip title="Local">
      <StorageOutlinedIcon fontSize="small" color="action" />
    </Tooltip>
  );
};
