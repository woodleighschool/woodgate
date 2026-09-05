import { client } from "@lib/api-client/client.gen";
import {
  createRole,
  deleteRole,
  getRole,
  listAuthorizationResources,
  listRoles,
  rotateStationKey,
  updateRole,
} from "@lib/api-client/sdk.gen";
import type {
  CheckinLocation,
  CheckinLocationMutation,
  ErrorModel,
  Role,
  RoleMutation,
} from "@lib/api-client/types.gen";

export * from "@lib/api-client/sdk.gen";
export type * from "@lib/api-client/types.gen";

export {
  createRole as createAuthzRole,
  deleteRole as deleteAuthzRole,
  getRole as getAuthzRole,
  listAuthorizationResources as listAuthzResources,
  listRoles as listAuthzRoles,
  rotateStationKey,
  updateRole as updateAuthzRole,
};

export type AuthzRole = Role;
export type AuthzRoleMutation = RoleMutation;
export type Location = CheckinLocation;
export type LocationMutation = CheckinLocationMutation;

export interface Page<T> {
  count: number;
  items: T[];
}

client.setConfig({
  credentials: "same-origin",
  querySerializer: { array: { style: "form", explode: false } },
});

client.interceptors.request.use((request) => {
  if (!request.headers.has("Accept")) {
    request.headers.set("Accept", "application/json");
  }
  return request;
});

let unauthorizedHandler: (() => void) | undefined;

client.interceptors.response.use((response) => {
  if (response.status === 401) unauthorizedHandler?.();
  return response;
});

export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler;
}

export class ApiError extends Error {
  readonly status: number;
  readonly body?: ErrorModel;

  constructor(status: number, message: string, body?: ErrorModel) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

interface ApiResult {
  data: unknown;
  error: ErrorModel | Error | undefined;
  response?: Response;
}

type ResponseData<Result extends ApiResult> = Extract<Result, { error: undefined }>["data"];

function describeError(body: ErrorModel | undefined, status: number): string {
  if (body?.errors?.length) {
    const details = body.errors
      .map((error) =>
        error.location ? `${error.location}: ${error.message ?? ""}` : (error.message ?? ""),
      )
      .filter(Boolean)
      .join("; ");
    if (details) return details;
  }
  if (body?.detail) return body.detail;
  if (body?.title) return body.title;
  return `request failed (${status})`;
}

export function unwrap<Result extends ApiResult>(
  pending: Promise<Result>,
): Promise<ResponseData<Result>>;
export async function unwrap(pending: Promise<ApiResult>): Promise<unknown> {
  const result = await pending;
  if (result.error instanceof Error) throw result.error;
  if (result.error !== undefined || !result.response?.ok) {
    const status = result.response?.status ?? 0;
    throw new ApiError(status, describeError(result.error, status), result.error);
  }
  return result.data;
}
