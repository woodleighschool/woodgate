import {
  keepPreviousData,
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import type { ApiError, Group, GroupMutation, PageGroup } from "@lib/api";
import { getGroup, listGroups, unwrap, updateGroup } from "@lib/api";
import type { ListGroupsData } from "@lib/api";
import { baseListParams } from "@lib/pagination";

export type GroupListParams = NonNullable<ListGroupsData["query"]>;

type QueryParams = Record<string, unknown>;

export const groupKeys = {
  all: ["groups"] as const,
  list: (params?: QueryParams) => ["groups", "list", params ?? {}] as const,
  detail: (id: number) => ["groups", "detail", id] as const,
};

function groupQueryParams(params: GroupListParams = {}) {
  return {
    ...baseListParams(params),
    values: params.values && params.values.length > 0 ? params.values : undefined,
  };
}

export function useGroups(params: GroupListParams = {}, options: { enabled?: boolean } = {}) {
  const queryParams = groupQueryParams(params);
  return useQuery<PageGroup, ApiError>({
    queryKey: groupKeys.list(queryParams),
    queryFn: ({ signal }) =>
      unwrap(
        listGroups({
          query: queryParams,
          signal,
        }),
      ),
    placeholderData: keepPreviousData,
    enabled: options.enabled,
  });
}

export function useGroup(id: number) {
  return useQuery(groupQueryOptions(id));
}

export function groupQueryOptions(id: number) {
  return queryOptions<Group, ApiError>({
    queryKey: groupKeys.detail(id),
    queryFn: ({ signal }) => unwrap(getGroup({ path: { id }, signal })),
  });
}

export function useUpdateGroup(id: number) {
  const queryClient = useQueryClient();
  return useMutation<Group, ApiError, GroupMutation>({
    mutationFn: (body) => unwrap(updateGroup({ path: { id }, body })),
    onSuccess: async (group) => {
      queryClient.setQueryData(groupKeys.detail(id), group);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: groupKeys.all }),
        queryClient.invalidateQueries({ queryKey: ["users"] }),
        queryClient.invalidateQueries({ queryKey: ["auth", "session"] }),
        queryClient.invalidateQueries({ queryKey: ["account"] }),
      ]);
      toast.add({ title: "Group Saved", type: "success" });
    },
  });
}
