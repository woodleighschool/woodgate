import { authApi, type AuthMe } from "@/api/authClient";

export interface AuthProviders {
  microsoft: boolean;
  local: boolean;
}

let cachedAuthMe: AuthMe | undefined;
let cachedAuthMePromise: Promise<AuthMe | undefined> | undefined;

const statusCodeOf = (error: unknown): number | undefined => {
  if (typeof error !== "object" || error === null) {
    return undefined;
  }
  return (error as { status?: number }).status;
};

export const isUnauthorizedError = (error: unknown): boolean => statusCodeOf(error) === 401;

const isTerminalSessionError = (error: unknown): boolean => {
  const status = statusCodeOf(error);
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
      if (isTerminalSessionError(error)) {
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
    if (isTerminalSessionError(error)) {
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
