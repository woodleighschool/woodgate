import { z } from "zod";

import { ApiError, xsrfHeaders } from "@lib/api";
import { sessionTransport } from "@lib/session";

const authUserSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  email: z.string().optional(),
  picture: z.string().optional(),
  aud: z.string().optional(),
  ip: z.string().optional(),
  attrs: z.record(z.string(), z.unknown()).optional(),
  role: z.string().optional(),
});

const permissionGrantSchema = z.object({
  resource: z.enum(["users", "groups", "locations", "checkins", "assets", "api_keys"]),
  action: z.enum(["read", "create", "write", "delete"]),
  location_id: z.string().nullable().optional(),
  asset_type: z.enum(["asset", "photo"]).nullable().optional(),
});

const resourceCapabilitySchema = z.object({
  read: z.boolean(),
  create: z.boolean(),
  write: z.boolean(),
  delete: z.boolean(),
});

const authPermissionsSchema = z.object({
  principal: z.object({
    type: z.enum(["user", "api_key"]),
    id: z.string(),
    display_name: z.string().optional(),
    email: z.string().optional(),
    name: z.string().optional(),
  }),
  admin: z.boolean(),
  access: z.array(permissionGrantSchema),
  capabilities: z.record(z.string(), resourceCapabilitySchema),
});

export interface LocalLoginRequest {
  user: string;
  passwd: string;
  aud?: string;
}

// go-pkgz/auth user payload returned by GET /auth/user.
export type AuthUser = z.infer<typeof authUserSchema>;
export type AuthPermissions = z.infer<typeof authPermissionsSchema>;
export type ResourceCapability = z.infer<typeof resourceCapabilitySchema>;

const authFetch = (path: string, init?: RequestInit): Promise<Response> =>
  (path === "/auth/list" || path.startsWith("/auth/local/login") ? fetch : sessionTransport.fetch)(
    path,
    { ...init, credentials: "include", headers: xsrfHeaders(init?.headers) },
  );

const expectOk = (response: Response): Promise<void> => {
  if (!response.ok) {
    throw new ApiError(response.status, { detail: response.statusText || "Request failed" });
  }
  return Promise.resolve();
};

const expectJson = async <T>(response: Response, schema: z.ZodType<T>): Promise<T> => {
  if (!response.ok) {
    throw new ApiError(response.status, { detail: response.statusText || "Request failed" });
  }
  const data: unknown = await response.json();
  return schema.parse(data);
};

export const authApi = {
  getUser: (signal?: AbortSignal): Promise<AuthUser> =>
    authFetch("/auth/user", signal ? { signal } : undefined).then((response) =>
      expectJson(response, authUserSchema),
    ),

  getPermissions: (signal?: AbortSignal): Promise<AuthPermissions> =>
    authFetch("/auth/me", signal ? { signal } : undefined).then((response) =>
      expectJson(response, authPermissionsSchema),
    ),

  listProviders: (signal?: AbortSignal): Promise<string[]> =>
    authFetch("/auth/list", signal ? { signal } : undefined).then((response) =>
      expectJson(response, z.array(z.string())),
    ),

  loginLocal: async (body: LocalLoginRequest): Promise<void> => {
    await authFetch("/auth/local/login?session=1", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(expectOk);
    sessionTransport.renew();
  },

  logout: async (): Promise<void> => {
    await authFetch("/auth/logout", { method: "POST" }).then(expectOk);
    sessionTransport.renew();
  },
};
