/**
 * Backend URLs for the dashboard. HTTP paths use Next rewrites (next.config.ts)
 * so the browser stays on the Next origin in dev. WebSocket uses the same host.
 */

export function apiUrl(path: string): string {
  const normalized = path.startsWith("/") ? path : `/${path}`;
  return `/api${normalized}`;
}

export const VIDEO_STREAM_PATH = "/video/stream";

/**
 * The dashboard WebSocket.
 *
 * In dev this must NOT go through the Next rewrite: the dev server proxies the
 * upgrade but drops the upstream connection to Go moments later, leaving the
 * browser side open (so the UI still reads as connected) while every message
 * sent afterwards is silently discarded. Point straight at the backend
 * instead. Set NEXT_PUBLIC_BACKEND_ORIGIN to the Go origin in development.
 *
 * In production Go serves the built frontend, so same-origin is correct and
 * the variable is left unset.
 */
export function dashboardWebSocketUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }
  const origin = process.env.NEXT_PUBLIC_BACKEND_ORIGIN?.trim();
  if (origin) {
    return `${origin.replace(/^http/, "ws").replace(/\/$/, "")}/ws/dashboard`;
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}/ws/dashboard`;
}
