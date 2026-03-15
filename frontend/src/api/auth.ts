import { authApi, type AuthMe } from "@/api/authClient";

export interface AuthProviders {
  microsoft: boolean;
  local: boolean;
}

let cachedAuthMe: AuthMe | undefined;
let cachedAuthMePromise: Promise<AuthMe | undefined> | undefined;

export const isAuthError = (error: unknown): boolean => {
  if (typeof error !== "object" || error === null) {
    return false;
  }
  const status = (error as { status?: number }).status;
  return status === 401 || status === 403;
};

export async function getAuthMe(signal?: AbortSignal): Promise<AuthMe | undefined> {
  if (cachedAuthMe) {
    return cachedAuthMe;
  }
  if (cachedAuthMePromise) {
    return cachedAuthMePromise;
  }

  cachedAuthMePromise = (async (): Promise<AuthMe | undefined> => {
    try {
      const me = await authApi.getMe(signal);
      cachedAuthMe = me;
      return me;
    } catch (error) {
      if (isAuthError(error)) {
        cachedAuthMe = undefined;
        return undefined;
      }
      throw error;
    } finally {
      cachedAuthMePromise = undefined;
    }
  })();

  return cachedAuthMePromise;
}

export const invalidateAuthMe = (): void => {
  cachedAuthMe = undefined;
  cachedAuthMePromise = undefined;
};

export async function loginLocal(username: string, password: string): Promise<void> {
  await authApi.loginLocal({ user: username, passwd: password, aud: globalThis.location.origin });
  invalidateAuthMe();
}

export async function logout(): Promise<void> {
  try {
    await authApi.logout();
  } catch (error) {
    if (isAuthError(error)) {
      invalidateAuthMe();
      return;
    }
    throw error;
  }
  invalidateAuthMe();
}

export async function listAuthProviders(signal?: AbortSignal): Promise<AuthProviders> {
  const providers = await authApi.listProviders(signal);
  const normalized = new Set(providers.map((entry): string => entry.trim().toLowerCase()));

  return {
    microsoft: normalized.has("microsoft"),
    local: normalized.has("local"),
  };
}
