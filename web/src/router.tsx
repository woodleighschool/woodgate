import { QueryClient } from "@tanstack/react-query";
import {
  createHashHistory,
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  redirect,
  useParams,
} from "@tanstack/react-router";
import type { ReactNode } from "react";

import { AppLayout, navigation } from "@components/layout/app-layout";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Skeleton } from "@components/ui/skeleton";
import { APIKeysList, APIKeyCreate, APIKeyShow } from "@features/api-keys/pages";
import { AssetsList, AssetCreate, AssetEdit, AssetShow } from "@features/assets/pages";
import { canAccess, permissionsQuery } from "@features/auth/access";
import { LoginPage } from "@features/auth/login";
import { CheckinsList, CheckinShow } from "@features/checkins/pages";
import { GroupsList, GroupShow } from "@features/groups/pages";
import { LocationsList, LocationCreate, LocationEdit } from "@features/locations/pages";
import { UsersList, UserShow } from "@features/users/pages";
import { ApiError, type PermissionAction } from "@lib/api";
import { sessionTransport } from "@lib/session";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (count, error) => !(error instanceof ApiError && error.status < 500) && count < 2,
      refetchOnWindowFocus: true,
    },
  },
});
const root = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: Outlet,
  validateSearch: (search: Record<string, unknown>) => search,
  notFoundComponent: () => (
    <PageShell>
      <PageHeader title="Page Not Found" description="The requested page is unavailable." />
    </PageShell>
  ),
  errorComponent: ({ error, reset }) => (
    <PageShell>
      <QueryError error={error} onRetry={reset} />
    </PageShell>
  ),
  pendingComponent: () => (
    <PageShell>
      <Skeleton className="h-80 w-full" />
    </PageShell>
  ),
});
const login = createRoute({ getParentRoute: () => root, path: "/login", component: LoginPage });
const authenticated = createRoute({
  getParentRoute: () => root,
  id: "authenticated",
  beforeLoad: async ({ context, location }) => {
    try {
      await context.queryClient.ensureQueryData({ ...permissionsQuery, revalidateIfStale: true });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401)
        throw redirect({ to: "/login", search: { from: location.href } });
      throw error;
    }
  },
  component: AppLayout,
});
const index = createRoute({
  getParentRoute: () => authenticated,
  path: "/",
  beforeLoad: async ({ context }) => {
    const permissions = await context.queryClient.ensureQueryData(permissionsQuery);
    const first = navigation.find((entry) => canAccess(permissions, entry.resource, "read"));
    if (first) throw redirect({ to: `/${first.resource}` });
  },
  component: () => (
    <PageShell>
      <PageHeader
        title="No Access"
        description="Your account does not have access to any resources."
      />
    </PageShell>
  ),
});
function page<const Path extends string>(
  path: Path,
  resource: string,
  action: PermissionAction,
  render: (id: string) => ReactNode,
) {
  return createRoute({
    getParentRoute: () => authenticated,
    path,
    beforeLoad: async ({ context }) => {
      const permissions = await context.queryClient.ensureQueryData(permissionsQuery);
      if (!canAccess(permissions, resource, action))
        throw new ApiError(403, { detail: "You do not have access to this page." });
    },
    component: function ResourcePage() {
      const params = useParams({ strict: false }) as { id?: string };
      return render(params.id ?? "");
    },
  });
}
const pages = [
  page("/locations", "locations", "read", () => <LocationsList />),
  page("/locations/create", "locations", "create", () => <LocationCreate />),
  page("/locations/$id", "locations", "read", (id) => <LocationEdit id={id} />),
  page("/checkins", "checkins", "read", () => <CheckinsList />),
  page("/checkins/$id/show", "checkins", "read", (id) => <CheckinShow id={id} />),
  page("/users", "users", "read", () => <UsersList />),
  page("/users/$id/show", "users", "read", (id) => <UserShow id={id} />),
  page("/users/$id/show/$tab", "users", "read", (id) => <UserShow id={id} />),
  page("/groups", "groups", "read", () => <GroupsList />),
  page("/groups/$id/show", "groups", "read", (id) => <GroupShow id={id} />),
  page("/groups/$id/show/$tab", "groups", "read", (id) => <GroupShow id={id} />),
  page("/assets", "assets", "read", () => <AssetsList />),
  page("/assets/create", "assets", "create", () => <AssetCreate />),
  page("/assets/$id", "assets", "write", (id) => <AssetEdit id={id} />),
  page("/assets/$id/show", "assets", "read", (id) => <AssetShow id={id} />),
  page("/api-keys", "api-keys", "read", () => <APIKeysList />),
  page("/api-keys/create", "api-keys", "create", () => <APIKeyCreate />),
  page("/api-keys/$id/show", "api-keys", "read", (id) => <APIKeyShow id={id} />),
  page("/api-keys/$id/show/$tab", "api-keys", "read", (id) => <APIKeyShow id={id} />),
];
export const router = createRouter({
  routeTree: root.addChildren([login, authenticated.addChildren([index, ...pages])]),
  history: createHashHistory(),
  context: { queryClient },
  defaultPreload: "intent",
});
sessionTransport.onExpired(() => {
  const from = router.history.location.href;
  queryClient.clear();
  if (from === "/login" || from.startsWith("/login?")) return;
  void router.navigate({ to: "/login", search: { from }, replace: true, ignoreBlocker: true });
});
declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
