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
      if (response.status === 401 && requestGeneration === generation) {
        generation++;
        onExpired?.();
      }
      return response;
    },
  };
}

export const sessionTransport = createSessionTransport((input, init) => fetch(input, init));
