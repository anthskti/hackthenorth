"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { dashboardWebSocketUrl } from "@/lib/backend";
import { fetchStatus } from "@/lib/api";
import type { AgentState, LogEntry, ManualInputPayload, Mode } from "@/lib/types";

type StatusMessage = {
  type: "status";
  mode: Mode;
  state?: AgentState;
};

type LogMessage = {
  type: "log";
  text: string;
  ts: number;
};

function applyStatus(
  msg: StatusMessage,
  setMode: (m: Mode) => void,
  setState: (s: AgentState | null) => void,
) {
  setMode(msg.mode);
  setState(msg.mode === "ai" ? (msg.state ?? "idle") : null);
}

function applySnapshot(
  snap: Awaited<ReturnType<typeof fetchStatus>>,
  setMode: (m: Mode) => void,
  setState: (s: AgentState | null) => void,
  setLogs: (l: LogEntry[]) => void,
  setPiConnected: (p: boolean) => void,
) {
  setMode(snap.mode);
  setState(snap.mode === "ai" ? (snap.state ?? "idle") : null);
  setLogs(snap.logs);
  setPiConnected(snap.pi_connected ?? false);
}

type UseDashboardLiveOptions = {
  enabled?: boolean;
};

export function useDashboardLive(options: UseDashboardLiveOptions = {}) {
  const enabled = options.enabled ?? true;
  const [mode, setMode] = useState<Mode>("ai");
  const [state, setState] = useState<AgentState | null>("idle");
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [wsConnected, setWsConnected] = useState(false);
  const [piConnected, setPiConnected] = useState(false);
  const [backendReachable, setBackendReachable] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const backoffRef = useRef(1000);

  const hydrate = useCallback(async () => {
    try {
      const snap = await fetchStatus();
      setBackendReachable(true);
      applySnapshot(snap, setMode, setState, setLogs, setPiConnected);
    } catch {
      setBackendReachable(false);
    }
  }, []);

  const applySnapshotFromServer = useCallback((snap: Awaited<ReturnType<typeof fetchStatus>>) => {
    setBackendReachable(true);
    applySnapshot(snap, setMode, setState, setLogs, setPiConnected);
  }, []);

  useEffect(() => {
    if (!enabled) {
      setWsConnected(false);
      setBackendReachable(false);
      return;
    }

    let cancelled = false;

    const connect = () => {
      if (cancelled) return;
      const url = dashboardWebSocketUrl();
      if (!url) return;

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setWsConnected(true);
        backoffRef.current = 1000;
        void hydrate();
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data as string) as StatusMessage | LogMessage;
          if (msg.type === "status") {
            applyStatus(msg, setMode, setState);
          } else if (msg.type === "log") {
            setLogs((prev) => {
              if (prev.some((e) => e.ts === msg.ts && e.text === msg.text)) {
                return prev;
              }
              return [...prev, { text: msg.text, ts: msg.ts }];
            });
          }
        } catch {
          // ignore malformed frames
        }
      };

      ws.onclose = () => {
        setWsConnected(false);
        wsRef.current = null;
        if (cancelled) return;
        const delay = backoffRef.current;
        backoffRef.current = Math.min(delay * 2, 30_000);
        reconnectTimerRef.current = setTimeout(connect, delay);
      };

      ws.onerror = () => {
        ws.close();
      };
    };

    void hydrate();
    connect();

    return () => {
      cancelled = true;
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      wsRef.current?.close();
    };
  }, [hydrate, enabled]);

  const sendManualInput = useCallback((payload: ManualInputPayload) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return;
    }
    ws.send(JSON.stringify({ type: "manual_input", ...payload }));
  }, []);

  return {
    mode,
    state,
    logs,
    wsConnected,
    piConnected,
    backendReachable,
    applySnapshotFromServer,
    sendManualInput,
  };
}
