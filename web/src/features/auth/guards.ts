import type { QueryClient } from "@tanstack/react-query";
import { redirect } from "@tanstack/react-router";

import { accountQueryOptions } from "@features/account/queries";
import { sessionQueryOptions } from "@features/auth/queries";
import { canAccess } from "@features/authz/permissions";
import type { PermissionLevel, SessionBody } from "@lib/api";

export type SessionUser = NonNullable<SessionBody["user"]>;

async function loadSession(queryClient: QueryClient): Promise<SessionBody> {
  return queryClient.fetchQuery(sessionQueryOptions);
}

/** Authenticated route guard. Redirects to login if no user is signed in. */
export async function requireUser(queryClient: QueryClient): Promise<SessionUser> {
  const session = await loadSession(queryClient);
  if (!session.user) throw redirect({ to: "/login" });
  return session.user;
}

/** Resource route guard backed by the signed-in account's type-wide permissions. */
export async function requirePermission(
  queryClient: QueryClient,
  resource: string,
  access: PermissionLevel,
  onForbidden: () => never,
): Promise<void> {
  const account = await queryClient.fetchQuery(accountQueryOptions);
  if (!canAccess(account, resource, access)) onForbidden();
}

/** Root entry point: route to login or the app shell. */
export async function redirectForEntry(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (!session.user) throw redirect({ to: "/login" });
  throw redirect({ to: "/checkins" });
}

/** Login page guard: send an already-signed-in user to the app. */
export async function redirectAuthenticatedFromLogin(queryClient: QueryClient): Promise<void> {
  const session = await loadSession(queryClient);
  if (session.user) throw redirect({ to: "/checkins" });
}
