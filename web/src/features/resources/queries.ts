import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { useUpload } from "@hooks/use-upload";
import type {
  ApiError,
  AuthzAssignment,
  AuthzAssignmentMutation,
  AuthzResource,
  AuthzRole,
  AuthzRoleMutation,
  Checkin,
  ListCheckinsData,
  ListLocationsData,
  ListStationsData,
  Location,
  LocationMutation,
  Page,
  Station,
  StationKey,
  StationMutation,
} from "@lib/api";
import {
  createLocationBackgroundUpload,
  createLocation,
  createLocationLogoUpload,
  createStation,
  createAuthzRole,
  deleteLocation,
  deleteStation,
  deleteAuthzRole,
  getAuthzRole,
  getCheckin,
  getLocation,
  getStation,
  listLocationBackgrounds,
  listLocationLogos,
  listAuthzAssignments,
  listAuthzResources,
  listAuthzRoles,
  listCheckins,
  listLocations,
  listStations,
  rotateStationKey,
  replaceAuthzAssignments,
  updateLocation,
  updateStation,
  updateAuthzRole,
  setLocationBackground,
  setLocationLogo,
  unwrap,
} from "@lib/api";
import { baseListParams, MAX_PAGE_SIZE } from "@lib/pagination";
import { directUploadRequest, type DirectUploadTarget } from "@lib/upload";

const keys = {
  locations: ["locations"] as const,
  checkins: ["checkins"] as const,
  stations: ["stations"] as const,
  locationBackgrounds: ["locations", "backgrounds"] as const,
  locationLogos: ["locations", "logos"] as const,
  resources: ["authz", "resources"] as const,
  roles: ["authz", "roles"] as const,
  assignments: ["authz", "assignments"] as const,
};

type CheckinListParams = NonNullable<ListCheckinsData["query"]>;
type LocationListParams = NonNullable<ListLocationsData["query"]>;
type StationListParams = NonNullable<ListStationsData["query"]>;

function checkinQueryParams(params: CheckinListParams = {}) {
  return {
    ...baseListParams(params),
    location_id: params.location_id,
    user_id: params.user_id,
    direction: params.direction,
    department: params.department,
    created_from: params.created_from,
    created_before: params.created_before,
  };
}

function locationQueryParams(params: LocationListParams = {}) {
  return { ...baseListParams(params), enabled: params.enabled };
}

function stationQueryParams(params: StationListParams = {}) {
  return {
    ...baseListParams(params),
    location_id: params.location_id,
    enabled: params.enabled,
  };
}

export function useLocations(params: LocationListParams = {}) {
  const query = locationQueryParams(params);
  return useQuery<Page<Location>, ApiError>({
    queryKey: [...keys.locations, "list", query],
    queryFn: ({ signal }) => unwrap(listLocations({ query, signal })),
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
      await queryClient.invalidateQueries({ queryKey: keys.locations });
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
      await queryClient.invalidateQueries({ queryKey: keys.locations });
      toast.add({ title: "Location Saved", type: "success" });
    },
  });
}

export function useDeleteLocation() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteLocation({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.locations });
      toast.add({ title: "Location Deleted", type: "success" });
    },
  });
}

export interface LocationImageObject {
  id: number;
  filename: string;
  content_url?: string;
}

interface LocationUploadTarget extends DirectUploadTarget {
  object_id: number;
}

interface LocationImagePage {
  count: number;
  items: LocationImageObject[];
}

type LocationImageUploadVariables = { locationID: number; file: File };

export function useLocationBackgrounds(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<LocationImagePage, ApiError>({
    queryKey: [...keys.locationBackgrounds, "list", query],
    queryFn: ({ signal }) => unwrap(listLocationBackgrounds({ query, signal })),
    enabled,
  });
}

export function useLocationLogos(enabled = true) {
  const query = baseListParams({}, { defaultPerPage: MAX_PAGE_SIZE });
  return useQuery<LocationImagePage, ApiError>({
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
  const queryClient = useQueryClient();
  const createUpload =
    kind === "background" ? createLocationBackgroundUpload : createLocationLogoUpload;
  const setImage = kind === "background" ? setLocationBackground : setLocationLogo;
  const label = kind === "background" ? "Background" : "Logo";

  return useUpload<LocationUploadTarget, LocationImageObject, LocationImageUploadVariables>({
    mutationKey: ["location-image-upload", kind],
    loadingText: `Uploading ${label}`,
    successText: `${label} Uploaded`,
    createIntent: ({ file }) => unwrap(createUpload({ body: { filename: file.name } })),
    uploadRequest: directUploadRequest,
    completeUpload: (intent, { locationID }, signal) =>
      unwrap(
        setImage({
          path: { id: locationID },
          body: { object_id: intent.object_id },
          signal,
        }),
      ),
    onSuccess: async (_object, { locationID }) => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: keys.locations }),
        queryClient.invalidateQueries({
          queryKey: kind === "background" ? keys.locationBackgrounds : keys.locationLogos,
        }),
        queryClient.invalidateQueries({
          queryKey: [...keys.locations, "detail", locationID],
        }),
      ]);
    },
  });
}

export function useCheckins(params: CheckinListParams = {}) {
  const query = checkinQueryParams(params);
  return useQuery<Page<Checkin>, ApiError>({
    queryKey: [...keys.checkins, "list", query],
    queryFn: ({ signal }) => unwrap(listCheckins({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useCheckin(id: number | null) {
  return useQuery<Checkin, ApiError>({
    queryKey: [...keys.checkins, "detail", id],
    queryFn: ({ signal }) => unwrap(getCheckin({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useStations(params: StationListParams = {}) {
  const query = stationQueryParams(params);
  return useQuery<Page<Station>, ApiError>({
    queryKey: [...keys.stations, "list", query],
    queryFn: ({ signal }) => unwrap(listStations({ query, signal })),
    placeholderData: keepPreviousData,
  });
}

export function useStation(id: number | null) {
  return useQuery<Station, ApiError>({
    queryKey: [...keys.stations, "detail", id],
    queryFn: ({ signal }) => unwrap(getStation({ path: { id: requireID(id) }, signal })),
    enabled: id !== null,
  });
}

export function useCreateStation() {
  const queryClient = useQueryClient();
  return useMutation<StationKey, ApiError, StationMutation>({
    mutationFn: (body) => unwrap(createStation({ body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.stations });
      toast.add({ title: "Station Created", type: "success" });
    },
  });
}

export function useUpdateStation(id: number) {
  const queryClient = useQueryClient();
  return useMutation<Station, ApiError, StationMutation>({
    mutationFn: (body) => unwrap(updateStation({ path: { id }, body })),
    onSuccess: async (station) => {
      queryClient.setQueryData([...keys.stations, "detail", id], station);
      await queryClient.invalidateQueries({ queryKey: keys.stations });
      toast.add({ title: "Station Saved", type: "success" });
    },
  });
}

export function useDeleteStation() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteStation({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.stations });
      toast.add({ title: "Station Deleted", type: "success" });
    },
  });
}

export function useRotateStationKey() {
  const queryClient = useQueryClient();
  return useMutation<StationKey, ApiError, number>({
    mutationFn: (id) => unwrap(rotateStationKey({ path: { id } })),
    onSuccess: async (result, id) => {
      if (result.station) {
        queryClient.setQueryData([...keys.stations, "detail", id], result.station);
      }
      await queryClient.invalidateQueries({ queryKey: keys.stations });
      toast.add({ title: "Station Key Rotated", type: "success" });
    },
  });
}

export function useAuthzResources() {
  return useQuery<AuthzResource[], ApiError>({
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
      await queryClient.invalidateQueries({ queryKey: keys.roles });
      toast.add({ title: "Role Saved", type: "success" });
    },
  });
}

export function useDeleteAuthzRole() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, number>({
    mutationFn: (id) => unwrap(deleteAuthzRole({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.roles });
      await queryClient.invalidateQueries({ queryKey: keys.assignments });
      toast.add({ title: "Role Deleted", type: "success" });
    },
  });
}

export function useAuthzAssignments() {
  return useQuery<AuthzAssignment[], ApiError>({
    queryKey: [...keys.assignments, "list"],
    queryFn: ({ signal }) => unwrap(listAuthzAssignments({ signal })).then((data) => data.items),
  });
}

export function useReplaceAuthzAssignments() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, AuthzAssignmentMutation>({
    mutationFn: (body) => unwrap(replaceAuthzAssignments({ body })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys.assignments });
      toast.add({ title: "Assignments Saved", type: "success" });
    },
  });
}

function requireID(id: number | null): number {
  if (id === null) throw new Error("detail query ran without an id");
  return id;
}
