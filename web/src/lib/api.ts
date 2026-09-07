import { client } from "@lib/api-client/client.gen";
import { sessionTransport } from "@lib/session";

export * from "@lib/api-client";

export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;
  constructor(status: number, body: unknown) {
    const detail = body && typeof body === "object" && "detail" in body ? body.detail : undefined;
    super(typeof detail === "string" && detail ? detail : `Request failed (${status})`);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

export function xsrfHeaders(headers?: HeadersInit) {
  const result = new Headers(headers);
  const token = document.cookie
    .split("; ")
    .find((part) => part.startsWith("woodgate_xsrf="))
    ?.slice("woodgate_xsrf=".length);
  if (token) result.set("X-XSRF-TOKEN", token);
  return result;
}

client.setConfig({ baseUrl: "/api/v1", credentials: "include", fetch: sessionTransport.fetch });
client.interceptors.request.use((request) => {
  for (const [name, value] of xsrfHeaders()) request.headers.set(name, value);
  return request;
});

export function unwrap<T>(
  promise: Promise<{ data?: T; error?: unknown; response?: Response }>,
): Promise<T>;
export async function unwrap(
  promise: Promise<{ data?: unknown; error?: unknown; response?: Response }>,
): Promise<unknown> {
  const result = await promise;
  if (!result.response)
    throw result.error instanceof Error ? result.error : new Error("Unable to reach the server");
  if (!result.response.ok) throw new ApiError(result.response.status, result.error);
  return result.data;
}
