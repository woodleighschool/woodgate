function metadata(name: string): string | undefined {
  return document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)?.content || undefined;
}

export const runtime = {
  version: metadata("woodgate-version"),
  serverURL: metadata("woodgate-server-url"),
};
