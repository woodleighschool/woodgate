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
  APIKeyAccessWriteRequest,
  APIKeyCreateRequest,
  LocationWriteRequest,
  UserAccessWriteRequest,
} from "@/api/types";
import type {
  CreateParams,
  CreateResult,
  DataProvider,
  DeleteManyParams,
  DeleteManyResult,
  DeleteParams,
  DeleteResult,
  GetListParams,
  GetListResult,
  GetManyParams,
  GetManyReferenceParams,
  GetManyReferenceResult,
  GetManyResult,
  GetOneParams,
  GetOneResult,
  Identifier,
  RaRecord,
  UpdateManyParams,
  UpdateManyResult,
  UpdateParams,
  UpdateResult,
} from "react-admin";

type RecordShape = Record<string, unknown>;

interface ListResult {
  data: RaRecord[];
  total: number;
}

const asRecord = (value: unknown): RecordShape =>
  typeof value === "object" && value !== null ? (value as RecordShape) : {};

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
    // eslint-disable-next-line unicorn/no-null
    return null;
  }

  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  // eslint-disable-next-line unicorn/no-null
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
    const rawFile = (value as { rawFile?: unknown }).rawFile;
    return isFile(rawFile) ? rawFile : undefined;
  }

  return undefined;
};

const getSearch = (filter?: RecordShape): string | undefined => getOptionalString(filter?.search);

const getSort = (parameters: GetListParams | GetManyReferenceParams): string | undefined => {
  const field = getOptionalString(parameters.sort?.field);
  return field;
};

const getOrder = (parameters: GetListParams | GetManyReferenceParams): string | undefined =>
  typeof parameters.sort?.order === "string" ? parameters.sort.order.toLowerCase() : undefined;

const asListQuery = (
  parameters: GetListParams | GetManyReferenceParams,
  extra?: Record<string, string | number | boolean | undefined>,
): Record<string, string | number | boolean | undefined> => {
  const filter = asRecord(parameters.filter);
  const page = parameters.pagination?.page;
  const perPage = parameters.pagination?.perPage;

  return {
    limit: typeof perPage === "number" ? perPage : undefined,
    offset: typeof page === "number" && typeof perPage === "number" ? (page - 1) * perPage : undefined,
    search: getSearch(filter),
    sort: getSort(parameters),
    order: getOrder(parameters),
    ...extra,
  };
};

const toListResult = (payload: { rows: unknown[]; total: number }): ListResult => ({
  data: payload.rows as RaRecord[],
  total: payload.total,
});

const unsupported = (operation: string, resource: string): never => {
  throw new Error(`${operation} not supported for resource: ${resource}`);
};

type ListHandler = (parameters: GetListParams | GetManyReferenceParams, signal?: AbortSignal) => Promise<ListResult>;
type GetOneHandler = (id: Identifier, signal?: AbortSignal) => Promise<RaRecord>;
type CreateHandler = (data: RecordShape) => Promise<RaRecord>;
type UpdateHandler = (id: Identifier, data: RecordShape) => Promise<RaRecord>;
type DeleteHandler = (id: Identifier) => Promise<void>;

type ResourceName = "users" | "groups" | "group-memberships" | "assets" | "locations" | "checkins" | "api-keys";

const listHandlers: Record<ResourceName, ListHandler> = {
  users: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);

    return toListResult(
      await usersApi.list(
        asListQuery(parameters, {
          location_id: getOptionalString(filter.location_id),
        }),
        signal,
      ),
    );
  },

  groups: async (parameters, signal): Promise<ListResult> =>
    toListResult(await groupsApi.list(asListQuery(parameters), signal)),

  "group-memberships": async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);

    return toListResult(
      await groupMembershipsApi.list(
        asListQuery(parameters, {
          group_id: getOptionalString(filter.group_id),
          user_id: getOptionalString(filter.user_id),
        }),
        signal,
      ),
    );
  },

  assets: async (parameters, signal): Promise<ListResult> =>
    toListResult(
      await assetsApi.list(
        asListQuery(parameters, {
          type: getOptionalString(asRecord(parameters.filter).type),
        }),
        signal,
      ),
    ),

  locations: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);

    return toListResult(
      await locationsApi.list(
        asListQuery(parameters, {
          enabled: typeof filter.enabled === "boolean" ? filter.enabled : undefined,
        }),
        signal,
      ),
    );
  },

  checkins: async (parameters, signal): Promise<ListResult> => {
    const filter = asRecord(parameters.filter);

    return toListResult(
      await checkinsApi.list(
        asListQuery(parameters, {
          location_id: getOptionalString(filter.location_id),
          user_id: getOptionalString(filter.user_id),
          direction: getOptionalString(filter.direction),
          created_from: getOptionalString(filter.created_from),
          created_to: getOptionalString(filter.created_to),
        }),
        signal,
      ),
    );
  },
  "api-keys": async (parameters, signal): Promise<ListResult> =>
    toListResult(await apiKeysApi.list(asListQuery(parameters), signal)),
};

const getOneHandlers: Record<ResourceName, GetOneHandler> = {
  users: (id, signal): Promise<RaRecord> => usersApi.get(String(id), signal) as Promise<RaRecord>,
  groups: (id, signal): Promise<RaRecord> => groupsApi.get(String(id), signal) as Promise<RaRecord>,
  "group-memberships": (id, signal): Promise<RaRecord> =>
    groupMembershipsApi.get(String(id), signal) as Promise<RaRecord>,
  assets: (id, signal): Promise<RaRecord> => assetsApi.get(String(id), signal) as Promise<RaRecord>,
  locations: (id, signal): Promise<RaRecord> => locationsApi.get(String(id), signal) as Promise<RaRecord>,
  checkins: (id, signal): Promise<RaRecord> => checkinsApi.get(String(id), signal) as Promise<RaRecord>,
  "api-keys": (id, signal): Promise<RaRecord> => apiKeysApi.get(String(id), signal) as Promise<RaRecord>,
};

const createHandlers: Partial<Record<ResourceName, CreateHandler>> = {
  assets: (data): Promise<RaRecord> => assetsApi.create(toAssetCreateRequest(data)) as Promise<RaRecord>,
  locations: (data): Promise<RaRecord> => locationsApi.create(toLocationWriteRequest(data)) as Promise<RaRecord>,
  "api-keys": (data): Promise<RaRecord> => apiKeysApi.create(data as APIKeyCreateRequest) as Promise<RaRecord>,
};

const updateHandlers: Partial<Record<ResourceName, UpdateHandler>> = {
  users: (id, data): Promise<RaRecord> =>
    usersApi.patch(String(id), data as UserAccessWriteRequest) as Promise<RaRecord>,
  assets: (id, data): Promise<RaRecord> => assetsApi.patch(String(id), toAssetUpdateRequest(data)) as Promise<RaRecord>,
  locations: (id, data): Promise<RaRecord> =>
    locationsApi.patch(String(id), toLocationWriteRequest(data)) as Promise<RaRecord>,
  "api-keys": (id, data): Promise<RaRecord> =>
    apiKeysApi.patch(String(id), data as APIKeyAccessWriteRequest) as Promise<RaRecord>,
};

const deleteHandlers: Partial<Record<ResourceName, DeleteHandler>> = {
  assets: (id): Promise<void> => assetsApi.delete(String(id)),
  locations: (id): Promise<void> => locationsApi.delete(String(id)),
  "api-keys": (id): Promise<void> => apiKeysApi.delete(String(id)),
};

const isResourceName = (value: string): value is ResourceName => value in listHandlers;

const assertResourceName = (operation: string, resource: string): ResourceName => {
  if (!isResourceName(resource)) {
    unsupported(operation, resource);
  }

  return resource as ResourceName;
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
  async getList<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: GetListParams,
  ): Promise<GetListResult<RecordType>> {
    const resourceName = assertResourceName("List", resource);
    const result = await listHandlers[resourceName](parameters, parameters.signal);
    return result as GetListResult<RecordType>;
  },

  async getOne<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: GetOneParams,
  ): Promise<GetOneResult<RecordType>> {
    const resourceName = assertResourceName("GetOne", resource);
    const data = await getOneHandlers[resourceName](toIdentifier(parameters.id), parameters.signal);
    return { data: data as RecordType };
  },

  async getMany<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: GetManyParams,
  ): Promise<GetManyResult<RecordType>> {
    const resourceName = assertResourceName("GetMany", resource);
    const data = await Promise.all(
      parameters.ids.map((id): Promise<RaRecord> => getOneHandlers[resourceName](toIdentifier(id), parameters.signal)),
    );
    return { data: data as RecordType[] };
  },

  async getManyReference<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: GetManyReferenceParams,
  ): Promise<GetManyReferenceResult<RecordType>> {
    const resourceName = assertResourceName("GetManyReference", resource);
    const filter = asRecord(parameters.filter);
    return (await listHandlers[resourceName](
      {
        ...parameters,
        filter: { ...filter, [parameters.target]: toIdentifier(parameters.id) },
      },
      parameters.signal,
    )) as GetManyReferenceResult<RecordType>;
  },

  async create<ResultRecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: CreateParams,
  ): Promise<CreateResult<ResultRecordType>> {
    const resourceName = assertResourceName("Create", resource);
    const handler = getCreateHandler(resourceName);
    const data = await handler(parameters.data as RecordShape);
    return { data: data as ResultRecordType };
  },

  async update<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: UpdateParams,
  ): Promise<UpdateResult<RecordType>> {
    const resourceName = assertResourceName("Update", resource);
    const handler = getUpdateHandler(resourceName);
    const data = await handler(toIdentifier(parameters.id), parameters.data as RecordShape);
    return { data: data as RecordType };
  },

  async delete<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: DeleteParams,
  ): Promise<DeleteResult<RecordType>> {
    const resourceName = assertResourceName("Delete", resource);
    const handler = getDeleteHandler(resourceName);
    await handler(toIdentifier(parameters.id));
    return { data: parameters.previousData as RecordType };
  },

  async deleteMany<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: DeleteManyParams,
  ): Promise<DeleteManyResult<RecordType>> {
    const resourceName = assertResourceName("DeleteMany", resource);
    const handler = getDeleteHandler(resourceName);
    const ids = parameters.ids.map((id): Identifier => toIdentifier(id));
    await Promise.all(ids.map((id): Promise<void> => handler(id)));
    return { data: ids };
  },

  async updateMany<RecordType extends RaRecord = RaRecord>(
    resource: string,
    parameters: UpdateManyParams,
  ): Promise<UpdateManyResult<RecordType>> {
    const resourceName = assertResourceName("UpdateMany", resource);
    const handler = getUpdateHandler(resourceName);
    const ids = parameters.ids.map((id): Identifier => toIdentifier(id));
    await Promise.all(ids.map((id): Promise<RaRecord> => handler(id, parameters.data as RecordShape)));
    return { data: ids };
  },
};
