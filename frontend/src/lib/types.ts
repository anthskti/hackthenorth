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

export type ManualInputPayload =
  | { action: "key"; value: string }
  | { action: "mouse_move"; x: number; y: number }
  | {
      action: "mouse_click";
      button: "left" | "right" | "middle";
      x?: number;
      y?: number;
    };

export type ManualInputMessage = ManualInputPayload & { type: "manual_input" };
