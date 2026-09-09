import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import {
  createAppKey,
  deleteAppKey,
  getAppKey,
  listAppKeys,
  listAppKeyLocations,
  updateAppKey,
  unwrap,
} from "@lib/api";
import type {
  AppKeyKey,
  AppKeyMutation,
  ListAppKeysData,
  ListAppKeyLocationsData,
  AppKeyCreatedKey,
} from "@lib/api";
import { baseListParams } from "@lib/pagination";

const keys = ["app-keys"] as const;

export function useAppKeys(params: NonNullable<ListAppKeysData["query"]>) {
  const query = baseListParams(params);
  return useQuery({
    queryKey: [...keys, "list", query],
    queryFn: ({ signal }) => unwrap(listAppKeys({ query, signal })),
    placeholderData: keepPreviousData,
  });
}
export function useAppKey(id: number | null) {
  return useQuery({
    queryKey: [...keys, "detail", id],
    queryFn: ({ signal }) => {
      if (id === null) throw new Error("Invalid app key ID");
      return unwrap(getAppKey({ path: { id }, signal }));
    },
    enabled: id !== null,
  });
}
export function useAppKeyLocations(params: NonNullable<ListAppKeyLocationsData["query"]>) {
  const query = baseListParams(params);
  return useQuery({
    queryKey: [...keys, "locations", query],
    queryFn: ({ signal }) => unwrap(listAppKeyLocations({ query, signal })),
    placeholderData: keepPreviousData,
  });
}
export function useCreateAppKey(onCreated: (key: AppKeyCreatedKey) => void) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (body: AppKeyMutation) => {
      const created = await unwrap(createAppKey({ body }));
      onCreated(created);
      const { api_key: _secret, ...key } = created;
      return key;
    },
    onSuccess: async (key) => {
      queryClient.setQueryData([...keys, "detail", key.id], key);
      await queryClient.invalidateQueries({ queryKey: keys });
      toast.add({ title: "App Key Created", type: "success" });
    },
  });
}
export function useUpdateAppKey(id: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: AppKeyMutation) => unwrap(updateAppKey({ path: { id }, body })),
    onSuccess: async (key: AppKeyKey) => {
      queryClient.setQueryData([...keys, "detail", id], key);
      await queryClient.invalidateQueries({ queryKey: keys });
      toast.add({ title: "App Key Saved", type: "success" });
    },
  });
}
export function useDeleteAppKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => unwrap(deleteAppKey({ path: { id } })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: keys });
      toast.add({ title: "App Key Deleted", type: "success" });
    },
  });
}
