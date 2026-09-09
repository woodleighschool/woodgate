function metadata(name: string): string | undefined {
  return document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)?.content || undefined;
}

export const runtime = {
  version: metadata("woodgate-version") ?? "0.0.0-dev",
  serverURL: metadata("woodgate-server-url"),
};
