import { MutationCache, QueryClient } from "@tanstack/react-query";

import { toast } from "@components/ui/toast";
import { ApiError } from "@lib/api";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnMount: true,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
      retry: (count, error) =>
        !(error instanceof ApiError && [401, 403].includes(error.status)) && count < 2,
      retryOnMount: false,
    },
  },
  mutationCache: new MutationCache({
    onError: (error, _variables, _context, mutation) => {
      if (mutation.meta?.inlineError || mutation.options.onError) return;
      toast.add({
        title: error instanceof Error ? error.message : "Request Failed",
        type: "error",
      });
    },
  }),
});
