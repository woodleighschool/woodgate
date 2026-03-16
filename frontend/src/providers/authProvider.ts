import { getCurrentUser, isAuthError, loginLocal, logout } from "@/api/auth";
import { authApi, type AuthPermissions } from "@/api/authClient";
import { canAccess } from "@/auth/access";
import type { AuthProvider, UserIdentity } from "react-admin";

const getCurrentPermissions = async (signal?: AbortSignal): Promise<AuthPermissions | undefined> => {
  try {
    return await authApi.getPermissions(signal);
  } catch (error) {
    if (isAuthError(error)) {
      return undefined;
    }
    throw error;
  }
};

export const authProvider: AuthProvider = {
  login({ username, password }: { username: string; password: string }): Promise<void> {
    return loginLocal(username, password);
  },

  logout(): Promise<void> {
    return logout();
  },

  async checkAuth(): Promise<void> {
    const user = await getCurrentUser();
    if (!user) {
      throw new Error("Not authenticated");
    }
  },

  checkError(error: unknown): Promise<void> {
    if (isAuthError(error)) {
      return Promise.reject(new Error("Not authenticated"));
    }
    return Promise.resolve();
  },

  async getIdentity(): Promise<UserIdentity> {
    const user = await getCurrentUser();
    if (!user) {
      throw new Error("Not authenticated");
    }

    const fullName = user.name?.trim() ? user.name : (user.email ?? "Unknown User");

    return {
      id: user.id,
      fullName,
    };
  },

  getPermissions(): Promise<AuthPermissions | undefined> {
    return getCurrentPermissions();
  },

  async canAccess({ action, resource, signal }): Promise<boolean> {
    return canAccess(await getCurrentPermissions(signal), resource, action);
  },

  supportAbortSignal: true,
};
