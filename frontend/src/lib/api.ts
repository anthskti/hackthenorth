import { apiUrl } from "@/lib/backend";
import type { AgentState, LogEntry, Mode } from "@/lib/types";

export type StatusResponse = {
  mode: Mode;
  state?: AgentState;
  logs: LogEntry[];
  pi_connected?: boolean;
  stream_width?: number;
  stream_height?: number;
};

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(apiUrl(path), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error ?? `request failed (${res.status})`);
  }
  return res.json() as Promise<T>;
}

export async function fetchStatus(): Promise<StatusResponse> {
  const res = await fetch(apiUrl("/status"), { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`status ${res.status}`);
  }
  return res.json() as Promise<StatusResponse>;
}

export async function postMode(mode: Mode): Promise<StatusResponse> {
  return postJSON("/mode", { mode });
}

export async function postPrompt(goal: string): Promise<StatusResponse> {
  return postJSON("/prompt", { goal });
}

export async function postControl(
  action: "pause" | "resume" | "stop",
): Promise<StatusResponse> {
  return postJSON("/control", { action });
}
