import { apiUrl } from "@/lib/backend";
import type { AgentState, LogEntry, Mode } from "@/lib/types";

export type StatusResponse = {
  mode: Mode;
  state?: AgentState;
  logs: LogEntry[];
  pi_connected?: boolean;
};

export async function fetchStatus(): Promise<StatusResponse> {
  const res = await fetch(apiUrl("/status"), { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`status ${res.status}`);
  }
  return res.json() as Promise<StatusResponse>;
}
