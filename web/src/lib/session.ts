export function createSessionTransport(fetcher: typeof fetch) {
  let generation = 0;
  let onExpired: (() => void) | undefined;
  return {
    renew() {
      generation++;
    },
    onExpired(handler: () => void) {
      onExpired = handler;
    },
    fetch: async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const requestGeneration = generation;
      const response = await fetcher(input, init);
      const url = new URL(
        input instanceof Request ? input.url : String(input),
        "https://local.invalid",
      );
      const method = init?.method ?? (input instanceof Request ? input.method : "GET");
      const isLogin = url.pathname === "/api/session" && method.toUpperCase() === "POST";
      if (!isLogin && response.status === 401 && requestGeneration === generation) {
        generation++;
        onExpired?.();
      }
      return response;
    },
  };
}

export const sessionTransport = createSessionTransport((input, init) => fetch(input, init));
