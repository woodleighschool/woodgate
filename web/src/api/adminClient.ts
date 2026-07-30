import { getCookie, XSRF_COOKIE_NAME, XSRF_HEADER_NAME } from "@/api/cookies";
import type { components, paths } from "@/api/openapi";
import type {
  APIKey,
  APIKeyListResponse,
  Asset,
  AssetListResponse,
  Checkin,
  CheckinListResponse,
  CreateAPIKeyData,
  Group,
  GroupListResponse,
  GroupMembership,
  GroupMembershipListResponse,
  Location,
  LocationListResponse,
  User,
  UserListResponse,
} from "@/api/types";
import createClient from "openapi-fetch";
import { HttpError } from "react-admin";

interface ApiResult<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

type Problem = components["schemas"]["Problem"];
type QueryParameters = Record<string, string | number | boolean | undefined>;
type Compacted<T extends QueryParameters> = { [K in keyof T]?: NonNullable<T[K]> };

export interface AssetCreateRequest {
  file: File;
  name?: string;
}

export interface AssetUpdateRequest {
  file?: File;
  name?: string;
}

const withXsrfHeaders = (headers?: HeadersInit): Headers => {
  const result = new Headers(headers);
  const xsrfToken = getCookie(XSRF_COOKIE_NAME);

  if (xsrfToken) {
    result.set(XSRF_HEADER_NAME, xsrfToken);
  }

  return result;
};

const client = createClient<paths>({
  baseUrl: "/api/v1",
  fetch: (request): Promise<Response> =>
    fetch(
      new Request(request, {
        credentials: "include",
        headers: withXsrfHeaders(request.headers),
      }),
    ),
});

const isProblem = (value: unknown): value is Problem =>
  typeof value === "object" &&
  value !== null &&
  "detail" in value &&
  typeof (value as { detail?: unknown }).detail === "string" &&
  "status" in value &&
  typeof (value as { status?: unknown }).status === "number";

const problemToBody = (problem: Problem): Record<string, unknown> => {
  const errors = Object.fromEntries(
    (problem.field_errors ?? []).map((fieldError): [string, string] => [fieldError.field, fieldError.message]),
  );

  return {
    ...problem,
    ...(Object.keys(errors).length > 0 ? { errors } : {}),
  };
};

const toHttpError = (error: unknown, response: Response): HttpError => {
  const message =
    isProblem(error) && error.detail.trim() !== "" ? error.detail : response.statusText || "Request failed";

  return new HttpError(message, response.status, isProblem(error) ? problemToBody(error) : error);
};

const expectBody = async <T>(resultPromise: Promise<ApiResult<T>>): Promise<T> => {
  const { data, error, response } = await resultPromise;

  if (error !== undefined) {
    throw toHttpError(error, response);
  }

  if (data === undefined) {
    throw new HttpError("Empty response", response.status);
  }

  return data;
};

const expectOk = async (resultPromise: Promise<ApiResult<unknown>>): Promise<void> => {
  const { error, response } = await resultPromise;

  if (error !== undefined) {
    throw toHttpError(error, response);
  }
};

const compactQuery = <T extends QueryParameters>(query: T): Compacted<T> => {
  const result: Compacted<T> = {};

  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined) {
      (result as Record<string, unknown>)[key] = value;
    }
  }

  return result;
};

const withQuery = <T extends QueryParameters>(query: T): { params: { query: Compacted<T> } } => ({
  params: { query: compactQuery(query) },
});

const withPath = <T extends string>(id: T): { params: { path: { id: T } } } => ({
  params: { path: { id } },
});

const list =
  <R>(path: keyof paths) =>
  (query: QueryParameters, signal?: AbortSignal): Promise<R> =>
    expectBody(client.GET(path as never, { ...withQuery(query), signal } as never));

const getOne =
  <R>(path: keyof paths) =>
  (id: string, signal?: AbortSignal): Promise<R> =>
    expectBody(client.GET(path as never, { ...withPath(id), signal } as never));

const createOne =
  <R>(path: keyof paths) =>
  (body: unknown): Promise<R> =>
    expectBody(client.POST(path as never, { body } as never));

const patchOne =
  <R>(path: keyof paths) =>
  (id: string, body: unknown): Promise<R> =>
    expectBody(client.PATCH(path as never, { ...withPath(id), body } as never));

const deleteOne =
  (path: keyof paths) =>
  (id: string): Promise<void> =>
    expectOk(client.DELETE(path as never, withPath(id) as never));

const assetFormData = (body: AssetCreateRequest | AssetUpdateRequest): FormData => {
  const formData = new FormData();
  if (body.name !== undefined) {
    formData.set("name", body.name);
  }
  if (body.file) {
    formData.set("file", body.file);
  }
  return formData;
};

export const usersApi = {
  list: list<UserListResponse>("/users"),
  get: getOne<User>("/users/{id}"),
  patch: patchOne<User>("/users/{id}"),
};

export const groupsApi = {
  list: list<GroupListResponse>("/groups"),
  get: getOne<Group>("/groups/{id}"),
};

export const groupMembershipsApi = {
  list: list<GroupMembershipListResponse>("/group-memberships"),
  get: getOne<GroupMembership>("/group-memberships/{id}"),
};

export const assetsApi = {
  list: list<AssetListResponse>("/assets"),
  get: getOne<Asset>("/assets/{id}"),
  create: (body: AssetCreateRequest): Promise<Asset> =>
    expectBody(client.POST("/assets" as never, { body: assetFormData(body) } as never)),
  patch: (id: string, body: AssetUpdateRequest): Promise<Asset> =>
    expectBody(client.PATCH("/assets/{id}" as never, { ...withPath(id), body: assetFormData(body) } as never)),
  delete: deleteOne("/assets/{id}"),
};

export const locationsApi = {
  list: list<LocationListResponse>("/locations"),
  get: getOne<Location>("/locations/{id}"),
  create: createOne<Location>("/locations"),
  patch: patchOne<Location>("/locations/{id}"),
  delete: deleteOne("/locations/{id}"),
};

export const checkinsApi = {
  list: list<CheckinListResponse>("/checkins"),
  get: getOne<Checkin>("/checkins/{id}"),
  create: createOne<Checkin>("/checkins"),
};

export const apiKeysApi = {
  list: list<APIKeyListResponse>("/api-keys"),
  get: getOne<APIKey>("/api-keys/{id}"),
  patch: patchOne<APIKey>("/api-keys/{id}"),
  create: createOne<CreateAPIKeyData>("/api-keys"),
  delete: deleteOne("/api-keys/{id}"),
};

export {
  type APIKeyAccessWriteRequest,
  type APIKeyCreateRequest,
  type CheckinCreateRequest,
  type LocationWriteRequest,
  type PermissionGrant,
  type UserAccessWriteRequest,
} from "@/api/types";
