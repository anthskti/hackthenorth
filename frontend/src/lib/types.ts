export type Mode = "ai" | "manual";

export type AgentState = "idle" | "thinking" | "acting" | "paused";

export type LogEntry = {
  text: string;
  ts: number;
};

export type AppState = {
  mode: Mode;
  state: AgentState | null;
  logs: LogEntry[];
  wsConnected: boolean;
  piConnected: boolean;
  goal: string;
};
