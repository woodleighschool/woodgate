import { getCookie, XSRF_COOKIE_NAME, XSRF_HEADER_NAME } from "@/api/cookies";
import type { PermissionGrant } from "@/api/types";
import { HttpError } from "react-admin";

export interface LocalLoginRequest {
  user: string;
  passwd: string;
  aud?: string;
}

export interface ResourceCapability {
  read: boolean;
  create: boolean;
  write: boolean;
  delete: boolean;
}

export interface AuthMe {
  principal: {
    type: "user" | "api_key";
    id: string;
    display_name?: string;
    email?: string;
    name?: string;
  };
  admin: boolean;
  access: PermissionGrant[];
  capabilities: Record<string, ResourceCapability>;
}

const withXsrfHeaders = (headers?: HeadersInit): Headers => {
  const result = new Headers(headers);
  const token = getCookie(XSRF_COOKIE_NAME);
  if (token) {
    result.set(XSRF_HEADER_NAME, token);
  }
  return result;
};

const authFetch = (path: string, init?: RequestInit): Promise<Response> =>
  fetch(path, { ...init, credentials: "include", headers: withXsrfHeaders(init?.headers) });

const expectOk = (response: Response): Promise<void> => {
  if (!response.ok) {
    throw new HttpError(response.statusText || "Request failed", response.status);
  }
  return Promise.resolve();
};

const expectJson = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    throw new HttpError(response.statusText || "Request failed", response.status);
  }
  return response.json() as Promise<T>;
};

export const authApi = {
  getMe: (signal?: AbortSignal): Promise<AuthMe> =>
    authFetch("/auth/me", signal ? { signal } : undefined).then(expectJson<AuthMe>),

  listProviders: (signal?: AbortSignal): Promise<string[]> =>
    authFetch("/auth/list", signal ? { signal } : undefined).then(expectJson<string[]>),

  loginLocal: (body: LocalLoginRequest): Promise<void> =>
    authFetch("/auth/local/login?session=1", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(expectOk),

  logout: (): Promise<void> => authFetch("/auth/logout", { method: "POST" }).then(expectOk),
};
