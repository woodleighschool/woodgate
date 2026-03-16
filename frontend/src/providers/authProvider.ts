import { getAuthMe, isUnauthorizedError, loginLocal, logout } from "@/api/auth";
import type { AuthMe } from "@/api/authClient";
import { canAccess } from "@/auth/access";
import type { AuthProvider, UserIdentity } from "react-admin";

export const authProvider: AuthProvider = {
  login({ username, password }: { username: string; password: string }): Promise<void> {
    return loginLocal(username, password);
  },

  logout(): Promise<void> {
    return logout();
  },

  async checkAuth(): Promise<void> {
    const me = await getAuthMe();
    if (!me) {
      throw new Error("Not authenticated");
    }
  },

  checkError(error: unknown): Promise<void> {
    if (isUnauthorizedError(error)) {
      return Promise.reject(new Error("Not authenticated"));
    }
    return Promise.resolve();
  },

  async getIdentity(): Promise<UserIdentity> {
    const me = await getAuthMe();
    if (!me) {
      throw new Error("Not authenticated");
    }

    const identity: UserIdentity = {
      id: me.principal.id,
      fullName: me.principal.display_name ?? me.principal.name ?? me.principal.email ?? "Unknown User",
    };
    return identity;
  },

  async getPermissions(): Promise<AuthMe | undefined> {
    return getAuthMe();
  },

  async canAccess({ action, resource, signal }): Promise<boolean> {
    return canAccess(await getAuthMe(signal), resource, action);
  },

  supportAbortSignal: true,
};
