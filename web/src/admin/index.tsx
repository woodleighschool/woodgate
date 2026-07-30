import { AdminLayout } from "@/admin/layout";
import { LoginPage } from "@/admin/login";
import { darkTheme, lightTheme } from "@/admin/theme";
import { authProvider } from "@/providers/authProvider";
import { dataProvider } from "@/providers/dataProvider";
import apiKeys from "@/resources/apiKeys";
import assets from "@/resources/assets";
import checkins from "@/resources/checkins";
import groups from "@/resources/groups";
import locations from "@/resources/locations";
import users from "@/resources/users";
import type { ReactElement } from "react";
import { Admin, Resource, type RaThemeOptions } from "react-admin";

export const App = (): ReactElement => (
  <Admin
    dataProvider={dataProvider}
    authProvider={authProvider}
    loginPage={LoginPage}
    theme={lightTheme as RaThemeOptions}
    darkTheme={darkTheme as RaThemeOptions}
    layout={AdminLayout}
    title="WoodGate"
    requireAuth
  >
    <Resource name="locations" {...locations} />
    <Resource name="checkins" {...checkins} />
    <Resource name="users" {...users} />
    <Resource name="groups" {...groups} />
    <Resource name="assets" {...assets} />
    <Resource name="api-keys" {...apiKeys} />
  </Admin>
);
