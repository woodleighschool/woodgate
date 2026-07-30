import { createPathBasedClient } from "openapi-fetch";
import { HttpError } from "react-admin";

import { getCookie, XSRF_COOKIE_NAME, XSRF_HEADER_NAME } from "@/api/cookies";
import type { components, operations, paths } from "@/api/openapi";
import type {
  APIKey,
  APIKeyListResponse,
  Asset,
  AssetListResponse,
  Checkin,
  CheckinListResponse,
  CreateAPIKeyData,
  DepartmentOptionListResponse,
  Group,
  GroupListResponse,
  GroupMembership,
  GroupMembershipListResponse,
  Location,
  LocationListResponse,
  User,
  UserListResponse,
} from "@/api/types";

interface ApiResult<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

type Problem = components["schemas"]["Problem"];
export type UsersQuery = NonNullable<operations["listUsers"]["parameters"]["query"]>;
export type GroupsQuery = NonNullable<operations["listGroups"]["parameters"]["query"]>;
export type GroupMembershipsQuery = NonNullable<
  operations["listGroupMemberships"]["parameters"]["query"]
>;
export type AssetsQuery = NonNullable<operations["listAssets"]["parameters"]["query"]>;
export type LocationsQuery = NonNullable<operations["listLocations"]["parameters"]["query"]>;
export type CheckinsQuery = NonNullable<operations["listCheckins"]["parameters"]["query"]>;
export type APIKeysQuery = NonNullable<operations["listAPIKeys"]["parameters"]["query"]>;

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

const client = createPathBasedClient<paths>({
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
  typeof Reflect.get(value, "detail") === "string" &&
  typeof Reflect.get(value, "status") === "number";

const problemToBody = (problem: Problem): Record<string, unknown> => {
  const errors = Object.fromEntries(
    (problem.field_errors ?? []).map((fieldError): [string, string] => [
      fieldError.field,
      fieldError.message,
    ]),
  );

  return {
    ...problem,
    ...(Object.keys(errors).length > 0 ? { errors } : {}),
  };
};

const toHttpError = (error: unknown, response: Response): HttpError => {
  const message =
    isProblem(error) && error.detail.trim() !== ""
      ? error.detail
      : response.statusText || "Request failed";

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

const withPath = <T extends string>(id: T): { params: { path: { id: T } } } => ({
  params: { path: { id } },
});

const withSignal = (signal?: AbortSignal): { signal?: AbortSignal } => (signal ? { signal } : {});

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
  list: (query: UsersQuery, signal?: AbortSignal): Promise<UserListResponse> =>
    expectBody(client["/users"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<User> =>
    expectBody(client["/users/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
  patch: (id: string, body: components["schemas"]["UserAccessWriteRequest"]): Promise<User> =>
    expectBody(client["/users/{id}"].PATCH({ ...withPath(id), body })),
};

export const groupsApi = {
  list: (query: GroupsQuery, signal?: AbortSignal): Promise<GroupListResponse> =>
    expectBody(client["/groups"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<Group> =>
    expectBody(client["/groups/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
};

export const groupMembershipsApi = {
  list: (
    query: GroupMembershipsQuery,
    signal?: AbortSignal,
  ): Promise<GroupMembershipListResponse> =>
    expectBody(client["/group-memberships"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<GroupMembership> =>
    expectBody(client["/group-memberships/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
};

export const assetsApi = {
  list: (query: AssetsQuery, signal?: AbortSignal): Promise<AssetListResponse> =>
    expectBody(client["/assets"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<Asset> =>
    expectBody(client["/assets/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
  create: (body: AssetCreateRequest): Promise<Asset> =>
    expectBody(
      client["/assets"].POST({
        body: { file: body.file.name, ...(body.name === undefined ? {} : { name: body.name }) },
        bodySerializer: () => assetFormData(body),
      }),
    ),
  patch: (id: string, body: AssetUpdateRequest): Promise<Asset> =>
    expectBody(
      client["/assets/{id}"].PATCH({
        ...withPath(id),
        body: {
          ...(body.file === undefined ? {} : { file: body.file.name }),
          ...(body.name === undefined ? {} : { name: body.name }),
        },
        bodySerializer: () => assetFormData(body),
      }),
    ),
  delete: (id: string): Promise<void> => expectOk(client["/assets/{id}"].DELETE(withPath(id))),
};

export const locationsApi = {
  list: (query: LocationsQuery, signal?: AbortSignal): Promise<LocationListResponse> =>
    expectBody(client["/locations"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<Location> =>
    expectBody(client["/locations/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
  create: (body: components["schemas"]["LocationWriteRequest"]): Promise<Location> =>
    expectBody(client["/locations"].POST({ body })),
  patch: (id: string, body: components["schemas"]["LocationWriteRequest"]): Promise<Location> =>
    expectBody(client["/locations/{id}"].PATCH({ ...withPath(id), body })),
  delete: (id: string): Promise<void> => expectOk(client["/locations/{id}"].DELETE(withPath(id))),
};

export const checkinsApi = {
  list: (query: CheckinsQuery, signal?: AbortSignal): Promise<CheckinListResponse> =>
    expectBody(client["/checkins"].GET({ params: { query }, ...withSignal(signal) })),
  listDepartments: (signal?: AbortSignal): Promise<DepartmentOptionListResponse> =>
    expectBody(client["/checkins/departments"].GET(withSignal(signal))),
  get: (id: string, signal?: AbortSignal): Promise<Checkin> =>
    expectBody(client["/checkins/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
  create: (body: components["schemas"]["CheckinCreateRequest"]): Promise<Checkin> =>
    expectBody(client["/checkins"].POST({ body })),
};

export const apiKeysApi = {
  list: (query: APIKeysQuery, signal?: AbortSignal): Promise<APIKeyListResponse> =>
    expectBody(client["/api-keys"].GET({ params: { query }, ...withSignal(signal) })),
  get: (id: string, signal?: AbortSignal): Promise<APIKey> =>
    expectBody(client["/api-keys/{id}"].GET({ ...withPath(id), ...withSignal(signal) })),
  patch: (id: string, body: components["schemas"]["APIKeyAccessWriteRequest"]): Promise<APIKey> =>
    expectBody(client["/api-keys/{id}"].PATCH({ ...withPath(id), body })),
  create: (body: components["schemas"]["APIKeyCreateRequest"]): Promise<CreateAPIKeyData> =>
    expectBody(client["/api-keys"].POST({ body })),
  delete: (id: string): Promise<void> => expectOk(client["/api-keys/{id}"].DELETE(withPath(id))),
};
