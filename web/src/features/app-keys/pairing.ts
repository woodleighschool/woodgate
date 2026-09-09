export function pairingPayload(baseURL: string, apiKey: string): string {
  return JSON.stringify({ base_url: baseURL.replace(/\/+$/, ""), api_key: apiKey });
}
