import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { UploadRequest } from "@woodleighschool/bloby-client";

import { toast } from "@components/ui/toast";
import { useUpload } from "@hooks/use-upload";
import type {
  ApiError,
  Definition,
  AuthzRole,
  AuthzRoleMutation,
  Checkin,
  CheckinAttachmentObjectView,
  CheckinDirectUploadTarget,
  GroupSummary,
  ListCheckinsData,
  ListLocationGroupsData,
  ListLocationsData,
  Location,
  LocationMutation,
  Page,
} from "@lib/api";
import {
  createLocationBackgroundUpload,
  createLocation,
  createLocationLogoUpload,
  createAuthzRole,
  deleteLocation,
  deleteAuthzRole,
  getAuthzRole,
  getCheckin,
  getCheckinUser,
  listCheckinDepartments,
  listCheckinLocations,
  getLocation,
  listLocationBackgrounds,
  listLocationGroups,
  listLocationLogos,
  listAuthzResources,
  listAuthzRoles,
  listCheckins,
  listLocations,
  updateLocation,
  updateAuthzRole,
  unwrap,
} from "@lib/api";
import { baseListParams, MAX_PAGE_SIZE } from "@lib/pagination";

const keys = {
  locations: ["locations"] as const,
  checkins: ["checkins"] as const,
  locationBackgrounds: ["locations", "backgrounds"] as const,
  locationLogos: ["locations", "logos"] as const,
  resources: ["authz", "resources"] as const,
  roles: ["authz", "roles"] as const,
};

type CheckinListParams = NonNullable<ListCheckinsData["query"]>;
type LocationGroupListParams = NonNullable<ListLocationGroupsData["query"]>;
type LocationListParams = NonNullable<ListLocationsData["query"]>;

function checkinQueryParams(params: CheckinListParams = {}) {
  return {
    ...baseListParams(params),
    location_id: params.location_id,
    user_id: params.user_id,
    direction: params.direction,
    departments: params.departments,
    created_from: params.created_from,
    created_before: params.created_before,
  };
}

function locationQueryParams(params: LocationListParams = {}) {
  return { ...baseListParams(params), enabled: params.enabled };
}

export function useLocations(params: LocationListParams = {}) {
  const query = locationQueryParams(params);
  return useQuery<Page<Location>, ApiError>({
    queryKey: [...keys.locations, "list", query],
    queryFn: ({ signal }) => unwrap(listLocations({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useLocationGroups(params: LocationGroupListParams = {}) {
  const query = baseListParams(params);
  return useQuery<Page<GroupSummary>, ApiError>({
    queryKey: [...keys.locations, "groups", query],
    queryFn: ({ signal }) => unwrap(listLocationGroups({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useLocation(id: number | null) {
  return useQuery<Location, ApiError>({
    queryKey: [...keys.locations, "detail", id],
    queryFn: ({ signal }) => unwrap(getLocation({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useCreateLocation() {
  const queryClient = useQueryClient();
  return useMutation<Location, ApiError, LocationMutation>({
    mutationFn: (body) => unwrap(createLocation({ body })),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.locations }),
        queryClient.invalidateQueries({ queryKey: ["app-keys", "locations"] }),
      ]);
      toast.add({ title: "Location Created", type: "success" });
    },
  });
}

export function useUpdateLocation(id: number) {
  const queryClient = useQueryClient();
  return useMutation<Location, ApiError, LocationMutation>({
    mutationFn: (body) => unwrap(updateLocation({ path: { id }, body })),
    onSuccess: async (location) => {
      queryClient.setQueryData([...keys.locations, "detail", id], location);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.locations }),
        queryClient.invalidateQueries({ queryKey: ["app-keys", "locations"] }),
      ]);
      toast.add({ title: "Location Saved", type: "success" });
    },
  });
}

export function useDeleteLocation() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteLocation({ path: { id } })),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.locations }),
        queryClient.invalidateQueries({ queryKey: ["app-keys", "locations"] }),
      ]);
      toast.add({ title: "Location Deleted", type: "success" });
    },
  });
}

function directUploadRequest({ upload }: CheckinDirectUploadTarget): UploadRequest {
  if (upload.strategy !== "direct-put") {
    throw new Error("Location images require a direct upload.");
  }
  return upload;
}

type LocationImageUploadVariables = { file: File };

export function useLocationBackgrounds(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<Page<CheckinAttachmentObjectView>, ApiError>({
    queryKey: [...keys.locationBackgrounds, "list", query],
    queryFn: ({ signal }) => unwrap(listLocationBackgrounds({ query, signal })),
    enabled,
  });
}

export function useLocationLogos(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<Page<CheckinAttachmentObjectView>, ApiError>({
    queryKey: [...keys.locationLogos, "list", query],
    queryFn: ({ signal }) => unwrap(listLocationLogos({ query, signal })),
    enabled,
  });
}

export function useUploadLocationBackground() {
  return useUploadLocationImage("background");
}

export function useUploadLocationLogo() {
  return useUploadLocationImage("logo");
}

function useUploadLocationImage(kind: "background" | "logo") {
  const createUpload =
    kind === "background" ? createLocationBackgroundUpload : createLocationLogoUpload;
  const label = kind === "background" ? "Background" : "Logo";

  return useUpload<CheckinDirectUploadTarget, number, LocationImageUploadVariables>({
    mutationKey: ["location-image-upload", kind],
    loadingText: `Uploading ${label}`,
    successText: `${label} Uploaded`,
    createIntent: ({ file }, signal) =>
      unwrap(createUpload({ body: { filename: file.name }, signal })),
    uploadRequest: directUploadRequest,
    completeUpload: (intent) => Promise.resolve(intent.object_id),
  });
}

export function useCheckins(params: CheckinListParams = {}) {
  const query = checkinQueryParams(params);
  return useQuery<Page<Checkin>, ApiError>({
    queryKey: [...keys.checkins, "list", query],
    queryFn: ({ signal }) =>
      unwrap(
        listCheckins({
          query,
          signal,
          querySerializer: { array: { style: "form", explode: true } },
        }),
      ),
    placeholderData: keepPreviousData,
  });
}

export function useCheckinDepartments() {
  return useQuery({
    queryKey: [...keys.checkins, "departments"],
    queryFn: ({ signal }) => unwrap(listCheckinDepartments({ signal })).then((data) => data.items),
  });
}

export function useCheckinLocations() {
  return useQuery({
    queryKey: [...keys.checkins, "locations"],
    queryFn: ({ signal }) => unwrap(listCheckinLocations({ signal })).then((data) => data.items),
  });
}

export function useCheckinUser(id: number | null) {
  return useQuery({
    queryKey: [...keys.checkins, "users", id],
    queryFn: ({ signal }) => unwrap(getCheckinUser({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useCheckin(id: number | null) {
  return useQuery<Checkin, ApiError>({
    queryKey: [...keys.checkins, "detail", id],
    queryFn: ({ signal }) => unwrap(getCheckin({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useAuthzResources() {
  return useQuery<Definition[], ApiError>({
    queryKey: keys.resources,
    queryFn: ({ signal }) => unwrap(listAuthzResources({ signal })).then((data) => data.items),
  });
}

export function useAuthzRoles() {
  return useQuery<AuthzRole[], ApiError>({
    queryKey: [...keys.roles, "list"],
    queryFn: ({ signal }) => unwrap(listAuthzRoles({ signal })).then((data) => data.items),
  });
}

export function useAuthzRole(id: number | null) {
  return useQuery<AuthzRole, ApiError>({
    queryKey: [...keys.roles, "detail", id],
    queryFn: ({ signal }) => unwrap(getAuthzRole({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useCreateAuthzRole() {
  const queryClient = useQueryClient();
  return useMutation<AuthzRole, ApiError, AuthzRoleMutation>({
    mutationFn: (body) => unwrap(createAuthzRole({ body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.roles });
      toast.add({ title: "Role Created", type: "success" });
    },
  });
}

export function useUpdateAuthzRole(id: number) {
  const queryClient = useQueryClient();
  return useMutation<AuthzRole, ApiError, AuthzRoleMutation>({
    mutationFn: (body) => unwrap(updateAuthzRole({ path: { id }, body })),
    onSuccess: async (role) => {
      queryClient.setQueryData([...keys.roles, "detail", id], role);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.roles }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["groups"] }),
        queryClient.invalidateQueries({ queryKey: ["auth", "session"] }),
        queryClient.invalidateQueries({ queryKey: ["account"] }),
      ]);
      toast.add({ title: "Role Saved", type: "success" });
    },
  });
}

export function useDeleteAuthzRole() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteAuthzRole({ path: { id } })),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.roles }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["groups"] }),
        queryClient.invalidateQueries({ queryKey: ["auth", "session"] }),
        queryClient.invalidateQueries({ queryKey: ["account"] }),
      ]);
      toast.add({ title: "Role Deleted", type: "success" });
    },
  });
}

function requireID(id: number | null): number {
  if (id === null) throw new Error("detail query ran without an id");
  return id;
}
