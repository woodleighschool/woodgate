import { authApi, type AuthUser } from "@/api/authClient";

export interface AuthProviders {
  microsoft: boolean;
  local: boolean;
}

export const isAuthError = (error: unknown): boolean => {
  if (typeof error !== "object" || error === null) {
    return false;
  }
  const status = (error as { status?: number }).status;
  return status === 401;
};

export async function getCurrentUser(signal?: AbortSignal): Promise<AuthUser | undefined> {
  try {
    return await authApi.getUser(signal);
  } catch (error) {
    if (isAuthError(error)) {
      return undefined;
    }
    throw error;
  }
}

export async function loginLocal(username: string, password: string): Promise<void> {
  await authApi.loginLocal({ user: username, passwd: password, aud: globalThis.location.origin });
}

export async function logout(): Promise<void> {
  try {
    await authApi.logout();
  } catch (error) {
    if (isAuthError(error)) {
      return;
    }
    throw error;
  }
}

export async function listAuthProviders(signal?: AbortSignal): Promise<AuthProviders> {
  const providers = await authApi.listProviders(signal);
  const normalized = new Set(providers.map((entry): string => entry.trim().toLowerCase()));

  return {
    microsoft: normalized.has("microsoft"),
    local: normalized.has("local"),
  };
}
