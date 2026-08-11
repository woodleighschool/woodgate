import type {
  CreateParams,
  DataProvider,
  DeleteManyParams,
  DeleteParams,
  GetListParams,
  GetManyParams,
  GetManyReferenceParams,
  GetOneParams,
  Identifier,
  UpdateManyParams,
  UpdateParams,
} from "react-admin";
import { z } from "zod";

import {
  apiKeysApi,
  assetsApi,
  checkinsApi,
  groupMembershipsApi,
  groupsApi,
  locationsApi,
  usersApi,
  type AssetCreateRequest,
  type AssetUpdateRequest,
} from "@/api/adminClient";
import type {
  APIKeyCreateRequest,
  LocationWriteRequest,
  UserAccessWriteRequest,
} from "@/api/types";

type RecordShape = Record<string, unknown>;

interface ListResult {
  data: any[];
  total: number;
}

interface BaseListQuery {
  limit?: number;
  offset?: number;
  search?: string;
  sort?: string;
  order?: "asc" | "desc";
}

const permissionGrantSchema = z.object({
  resource: z.enum(["users", "groups", "locations", "checkins", "assets", "api_keys"]),
  action: z.enum(["read", "create", "write", "delete"]),
  location_id: z.string().nullable().optional(),
  asset_type: z.enum(["asset", "photo"]).nullable().optional(),
});

const accessWriteRequestSchema = z.object({
  admin: z.boolean(),
  access: z.array(permissionGrantSchema),
});

const apiKeyCreateRequestSchema = z.object({
  name: z.string(),
  expires_at: z.string().nullable().optional(),
});

const toAccessWriteRequest = (data: RecordShape): UserAccessWriteRequest => {
  const parsed = accessWriteRequestSchema.parse(data);
  return {
    admin: parsed.admin,
    access: parsed.access.map((grant) => ({
      resource: grant.resource,
      action: grant.action,
      ...(grant.location_id === undefined ? {} : { location_id: grant.location_id }),
      ...(grant.asset_type === undefined ? {} : { asset_type: grant.asset_type }),
    })),
  };
};

const toAPIKeyCreateRequest = (data: RecordShape): APIKeyCreateRequest => {
  const parsed = apiKeyCreateRequestSchema.parse(data);
  return {
    name: parsed.name,
    ...(parsed.expires_at === undefined ? {} : { expires_at: parsed.expires_at }),
  };
};

const asRecord = (value: unknown): RecordShape =>
  typeof value === "object" && value !== null ? Object.fromEntries(Object.entries(value)) : {};

const toIdentifier = String;

const getOptionalString = (value: unknown): string | undefined => {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
};

const getNullableString = (value: unknown): string | null | undefined => {
  if (value === null) {
    return null;
  }

  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
};
const isFile = (value: unknown): value is File => value instanceof File;

const getAssetFile = (value: unknown): File | undefined => {
  if (Array.isArray(value)) {
    return getAssetFile(value[0]);
  }
  if (isFile(value)) {
    return value;
  }
  if (typeof value === "object" && value !== null && "rawFile" in value) {
    const rawFile = Reflect.get(value, "rawFile");
    return isFile(rawFile) ? rawFile : undefined;
  }

  return undefined;
};

const getSearch = (filter?: RecordShape): string | undefined => getOptionalString(filter?.search);

const getSort = (parameters: GetListParams | GetManyReferenceParams): string | undefined => {
  const field = getOptionalString(parameters.sort?.field);
  return field;
};

const getOrder = (
  parameters: GetListParams | GetManyReferenceParams,
): "asc" | "desc" | undefined => {
  const order = parameters.sort?.order.toLowerCase();
  return order === "asc" || order === "desc" ? order : undefined;
};

const asListQuery = <Extra extends object>(
  parameters: GetListParams | GetManyReferenceParams,
  extra: Extra,
): BaseListQuery & Extra => {
  const filter = asRecord(parameters.filter);
  const page = parameters.pagination?.page;
  const perPage = parameters.pagination?.perPage;
  const search = getSearch(filter);
  const sort = getSort(parameters);
  const order = getOrder(parameters);

  return {
    ...(typeof perPage === "number" ? { limit: perPage } : {}),
    ...(typeof page === "number" && typeof perPage === "number"
      ? { offset: (page - 1) * perPage }
      : {}),
    ...(search === undefined ? {} : { search }),
    ...(sort === undefined ? {} : { sort }),
    ...(order === undefined ? {} : { order }),
    ...extra,
  };
};

const toListResult = (payload: { rows: any[]; total: number }): ListResult => ({
  data: payload.rows,
  total: payload.total,
});

const unsupported = (operation: string, resource: string): never => {
  throw new Error(`${operation} not supported for resource: ${resource}`);
};

type ListHandler = (
  parameters: GetListParams | GetManyReferenceParams,
  signal?: AbortSignal,
) => Promise<ListResult>;
type GetOneHandler = (id: Identifier, signal?: AbortSignal) => Promise<any>;
type CreateHandler = (data: RecordShape) => Promise<any>;
type UpdateHandler = (id: Identifier, data: RecordShape) => Promise<any>;
type DeleteHandler = (id: Identifier) => Promise<void>;

type ResourceName =
  | "users"
  | "groups"
  | "group-memberships"
  | "assets"
  | "locations"
  | "checkins"
  | "api-keys";

type ListResourceName = ResourceName | "checkin-departments";

const listHandlers: Record<ListResourceName, ListHandler> = {
  users: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);
    const locationId = getOptionalString(filter.location_id);

    return toListResult(
      await usersApi.list(
        asListQuery(parameters, locationId === undefined ? {} : { location_id: locationId }),
        signal,
      ),
    );
  },

  groups: async (parameters, signal): Promise<ListResult> =>
    toListResult(await groupsApi.list(asListQuery(parameters, {}), signal)),

  "group-memberships": async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);
    const groupId = getOptionalString(filter.group_id);
    const userId = getOptionalString(filter.user_id);

    return toListResult(
      await groupMembershipsApi.list(
        asListQuery(parameters, {
          ...(groupId === undefined ? {} : { group_id: groupId }),
          ...(userId === undefined ? {} : { user_id: userId }),
        }),
        signal,
      ),
    );
  },

  assets: async (parameters, signal): Promise<ListResult> => {
    const assetType = getOptionalString(asRecord(parameters.filter).type);

    return toListResult(
      await assetsApi.list(
        asListQuery(
          parameters,
          assetType === "asset" || assetType === "photo" ? { type: assetType } : {},
        ),
        signal,
      ),
    );
  },

  locations: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);
    const enabled = typeof filter.enabled === "boolean" ? filter.enabled : undefined;

    return toListResult(
      await locationsApi.list(
        asListQuery(parameters, enabled === undefined ? {} : { enabled }),
        signal,
      ),
    );
  },

  checkins: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);
    const locationId = getOptionalString(filter.location_id);
    const userId = getOptionalString(filter.user_id);
    const direction = getOptionalString(filter.direction);
    const department = getOptionalString(filter.department);
    const createdFrom = getOptionalString(filter.created_from);
    const createdTo = getOptionalString(filter.created_to);

    return toListResult(
      await checkinsApi.list(
        asListQuery(parameters, {
          ...(locationId === undefined ? {} : { location_id: locationId }),
          ...(userId === undefined ? {} : { user_id: userId }),
          ...(direction === "check_in" || direction === "check_out" ? { direction } : {}),
          ...(department === undefined ? {} : { department }),
          ...(createdFrom === undefined ? {} : { created_from: createdFrom }),
          ...(createdTo === undefined ? {} : { created_to: createdTo }),
        }),
        signal,
      ),
    );
  },
  "checkin-departments": async (_parameters, signal): Promise<ListResult> =>
    toListResult(await checkinsApi.listDepartments(signal)),
  "api-keys": async (parameters, signal): Promise<ListResult> =>
    toListResult(await apiKeysApi.list(asListQuery(parameters, {}), signal)),
};

const getOneHandlers: Record<ResourceName, GetOneHandler> = {
  users: (id, signal): Promise<any> => usersApi.get(String(id), signal),
  groups: (id, signal): Promise<any> => groupsApi.get(String(id), signal),
  "group-memberships": (id, signal): Promise<any> => groupMembershipsApi.get(String(id), signal),
  assets: (id, signal): Promise<any> => assetsApi.get(String(id), signal),
  locations: (id, signal): Promise<any> => locationsApi.get(String(id), signal),
  checkins: (id, signal): Promise<any> => checkinsApi.get(String(id), signal),
  "api-keys": (id, signal): Promise<any> => apiKeysApi.get(String(id), signal),
};

const createHandlers: Partial<Record<ResourceName, CreateHandler>> = {
  assets: (data): Promise<any> => assetsApi.create(toAssetCreateRequest(data)),
  locations: (data): Promise<any> => locationsApi.create(toLocationWriteRequest(data)),
  "api-keys": (data): Promise<any> => apiKeysApi.create(toAPIKeyCreateRequest(data)),
};

const updateHandlers: Partial<Record<ResourceName, UpdateHandler>> = {
  users: (id, data): Promise<any> => usersApi.patch(String(id), toAccessWriteRequest(data)),
  assets: (id, data): Promise<any> => assetsApi.patch(String(id), toAssetUpdateRequest(data)),
  locations: (id, data): Promise<any> =>
    locationsApi.patch(String(id), toLocationWriteRequest(data)),
  "api-keys": (id, data): Promise<any> => apiKeysApi.patch(String(id), toAccessWriteRequest(data)),
};

const deleteHandlers: Partial<Record<ResourceName, DeleteHandler>> = {
  assets: (id): Promise<void> => assetsApi.delete(String(id)),
  locations: (id): Promise<void> => locationsApi.delete(String(id)),
  "api-keys": (id): Promise<void> => apiKeysApi.delete(String(id)),
};

const isListResourceName = (value: string): value is ListResourceName => value in listHandlers;

const assertListResourceName = (operation: string, resource: string): ListResourceName => {
  if (isListResourceName(resource)) {
    return resource;
  }

  return unsupported(operation, resource);
};

const isResourceName = (value: string): value is ResourceName => value in getOneHandlers;

const assertResourceName = (operation: string, resource: string): ResourceName => {
  if (isResourceName(resource)) {
    return resource;
  }

  return unsupported(operation, resource);
};

const getCreateHandler = (resourceName: ResourceName): CreateHandler => {
  const handler = createHandlers[resourceName];
  if (!handler) {
    return unsupported("Create", resourceName);
  }

  return handler;
};

const getUpdateHandler = (resourceName: ResourceName): UpdateHandler => {
  const handler = updateHandlers[resourceName];
  if (!handler) {
    return unsupported("Update", resourceName);
  }

  return handler;
};

const getDeleteHandler = (resourceName: ResourceName): DeleteHandler => {
  const handler = deleteHandlers[resourceName];
  if (!handler) {
    return unsupported("Delete", resourceName);
  }

  return handler;
};

const toAssetCreateRequest = (data: RecordShape): AssetCreateRequest => {
  const file = getAssetFile(data.file);
  const name = typeof data.name === "string" ? data.name.trim() : undefined;

  if (!file) {
    throw new Error("Asset file is required.");
  }

  return {
    file,
    ...(name === undefined ? {} : { name }),
  };
};

const toAssetUpdateRequest = (data: RecordShape): AssetUpdateRequest => {
  const name = typeof data.name === "string" ? data.name.trim() : undefined;
  const file = getAssetFile(data.file);

  return {
    ...(file ? { file } : {}),
    ...(name === undefined ? {} : { name }),
  };
};

const toLocationWriteRequest = (data: RecordShape): LocationWriteRequest => {
  const backgroundAssetId = getNullableString(data.background_asset_id);
  const logoAssetId = getNullableString(data.logo_asset_id);

  return {
    name: typeof data.name === "string" ? data.name.trim() : "",
    description: typeof data.description === "string" ? data.description : "",
    enabled: data.enabled === true,
    notes: data.notes === true,
    photo: data.photo === true,
    group_ids: Array.isArray(data.group_ids) ? data.group_ids.map(String) : [],
    ...(backgroundAssetId === undefined ? {} : { background_asset_id: backgroundAssetId }),
    ...(logoAssetId === undefined ? {} : { logo_asset_id: logoAssetId }),
  };
};

export const dataProvider: DataProvider = {
  async getList(resource: string, parameters: GetListParams) {
    const resourceName = assertListResourceName("List", resource);
    return listHandlers[resourceName](parameters, parameters.signal);
  },

  async getOne(resource: string, parameters: GetOneParams) {
    const resourceName = assertResourceName("GetOne", resource);
    const data = await getOneHandlers[resourceName](toIdentifier(parameters.id), parameters.signal);
    return { data };
  },

  async getMany(resource: string, parameters: GetManyParams) {
    const resourceName = assertResourceName("GetMany", resource);
    const data = await Promise.all(
      parameters.ids.map((id): Promise<any> =>
        getOneHandlers[resourceName](toIdentifier(id), parameters.signal),
      ),
    );
    return { data };
  },

  async getManyReference(resource: string, parameters: GetManyReferenceParams) {
    const resourceName = assertResourceName("GetManyReference", resource);
    const filter = asRecord(parameters.filter);
    return listHandlers[resourceName](
      {
        ...parameters,
        filter: { ...filter, [parameters.target]: toIdentifier(parameters.id) },
      },
      parameters.signal,
    );
  },

  async create(resource: string, parameters: CreateParams) {
    const resourceName = assertResourceName("Create", resource);
    const handler = getCreateHandler(resourceName);
    const data = await handler(asRecord(parameters.data));
    return { data };
  },

  async update(resource: string, parameters: UpdateParams) {
    const resourceName = assertResourceName("Update", resource);
    const handler = getUpdateHandler(resourceName);
    const data = await handler(toIdentifier(parameters.id), asRecord(parameters.data));
    return { data };
  },

  async delete(resource: string, parameters: DeleteParams) {
    const resourceName = assertResourceName("Delete", resource);
    const handler = getDeleteHandler(resourceName);
    await handler(toIdentifier(parameters.id));
    return { data: parameters.previousData ?? { id: parameters.id } };
  },

  async deleteMany(resource: string, parameters: DeleteManyParams) {
    const resourceName = assertResourceName("DeleteMany", resource);
    const handler = getDeleteHandler(resourceName);
    const ids = parameters.ids.map((id): Identifier => toIdentifier(id));
    await Promise.all(ids.map((id): Promise<void> => handler(id)));
    return { data: ids };
  },

  async updateMany(resource: string, parameters: UpdateManyParams) {
    const resourceName = assertResourceName("UpdateMany", resource);
    const handler = getUpdateHandler(resourceName);
    const ids = parameters.ids.map((id): Identifier => toIdentifier(id));
    await Promise.all(ids.map((id): Promise<any> => handler(id, asRecord(parameters.data))));
    return { data: ids };
  },
};
