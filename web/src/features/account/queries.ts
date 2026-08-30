import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { sessionQueryOptions } from "@features/auth/queries";
import { userKeys } from "@features/directory/users/queries";
import type { Account, AccountMutation, ApiError, SessionBody } from "@lib/api";
import { getAccount, unwrap, updateAccount } from "@lib/api";

export const accountKey = ["account"] as const;

export const accountQueryOptions = queryOptions<Account, ApiError>({
  queryKey: accountKey,
  queryFn: async ({ signal }) => unwrap(getAccount({ signal })),
});

export function useAccount() {
  return useQuery(accountQueryOptions);
}

export function useUpdateAccount() {
  const queryClient = useQueryClient();
  return useMutation<Account, ApiError, AccountMutation>({
    mutationFn: (body) => unwrap(updateAccount({ body })),
    onSuccess: async (account) => {
      queryClient.setQueryData(accountKey, account);
      queryClient.setQueryData(userKeys.detail(account.user.id), account.user);
      queryClient.setQueryData(sessionQueryOptions.queryKey, (session: SessionBody | undefined) => {
        if (!session) return session;
        return { ...session, user: account.user };
      });
      await queryClient.invalidateQueries({ queryKey: userKeys.all });
    },
  });
}
