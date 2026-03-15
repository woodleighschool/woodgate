export function getCookie(name: string): string | undefined {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) {
    return parts.pop()?.split(";").shift();
  }
  return undefined;
}

export const XSRF_COOKIE_NAME = "woodgate_xsrf";
export const XSRF_HEADER_NAME = "X-XSRF-TOKEN";
