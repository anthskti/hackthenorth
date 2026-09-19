/**
 * Backend URLs for the dashboard. HTTP paths use Next rewrites (next.config.ts)
 * so the browser stays on the Next origin in dev. WebSocket uses the same host.
 */

export function apiUrl(path: string): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `/api${normalized}`;
}

export const VIDEO_STREAM_PATH = "/video/stream";

export function dashboardWebSocketUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws/dashboard`;
}
