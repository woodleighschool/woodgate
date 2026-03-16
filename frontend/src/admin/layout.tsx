import GitHubIcon from "@mui/icons-material/GitHub";
import { Box, IconButton, Typography } from "@mui/material";
import type { ComponentProps, ReactElement } from "react";
import { AppBar, Layout, TitlePortal } from "react-admin";

const repoUrl = "https://github.com/woodleighschool/woodgate";

const AppToolbar = (): ReactElement => (
  <Box sx={{ display: "flex", alignItems: "center", width: "100%", gap: 2 }}>
    <Typography variant="h5">WoodGate</Typography>
    <TitlePortal />
    <Box sx={{ flex: 1 }} />
    <IconButton color="inherit" component="a" href={repoUrl} target="_blank" rel="noreferrer">
      <GitHubIcon />
    </IconButton>
  </Box>
);

const AdminAppBar = (): ReactElement => (
  <AppBar toolbar={<AppToolbar />} sx={{ "& .RaUserMenu-userButton": { whiteSpace: "nowrap" } }} />
);

type LayoutProperties = ComponentProps<typeof Layout>;

export const AdminLayout = ({ children, ...rest }: LayoutProperties): ReactElement => (
  <Layout {...rest} appBar={AdminAppBar}>
    {children}
  </Layout>
);
