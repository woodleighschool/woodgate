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

// go-pkgz/auth user payload returned by GET /auth/user.
export interface AuthUser {
  id: string;
  name?: string;
  email?: string;
  picture?: string;
  aud?: string;
  ip?: string;
  attrs?: Record<string, unknown>;
  role?: string;
}

export interface AuthPermissions {
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
  getUser: (signal?: AbortSignal): Promise<AuthUser> =>
    authFetch("/auth/user", signal ? { signal } : undefined).then(expectJson<AuthUser>),

  getPermissions: (signal?: AbortSignal): Promise<AuthPermissions> =>
    authFetch("/auth/me", signal ? { signal } : undefined).then(expectJson<AuthPermissions>),

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
