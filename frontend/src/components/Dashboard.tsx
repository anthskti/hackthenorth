"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { AgentState, LogEntry, Mode } from "@/lib/types";
import { ActionLog } from "./ActionLog";
import { ControlRail } from "./ControlRail";
import { Header } from "./Header";
import { VideoPane } from "./VideoPane";

const FAKE_INITIAL_LOGS: LogEntry[] = [
  { text: "Dashboard connected (fake state)", ts: Date.now() - 60_000 },
  { text: "Mode: AI / waiting for a goal", ts: Date.now() - 30_000 },
];

function appendLog(logs: LogEntry[], text: string): LogEntry[] {
  return [...logs, { text, ts: Date.now() }];
}

export function Dashboard() {
  const [mode, setMode] = useState<Mode>("ai");
  const [state, setState] = useState<AgentState | null>("idle");
  const [logs, setLogs] = useState<LogEntry[]>(FAKE_INITIAL_LOGS);
  const [goal, setGoal] = useState("");
  const [wsConnected] = useState(true);
  const [piConnected] = useState(true);
  const [videoFocused, setVideoFocused] = useState(false);
  const [lastAck, setLastAck] = useState<string | null>(null);

  const simTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearSim = useCallback(() => {
    if (simTimerRef.current) {
      clearTimeout(simTimerRef.current);
      simTimerRef.current = null;
    }
  }, []);

  useEffect(() => () => clearSim(), [clearSim]);

  const handleModeChange = (next: Mode) => {
    setMode(next);
    clearSim();
    if (next === "manual") {
      setState(null);
      setLogs((prev) =>
        appendLog(prev, "Switched to manual — agent stopped (fake)"),
      );
    } else {
      setState("idle");
      setLogs((prev) => appendLog(prev, "Switched to AI mode (fake)"));
    }
  };

  const handleRun = () => {
    const trimmed = goal.trim();
    if (mode !== "ai" || state !== "idle" || !trimmed) return;
    clearSim();
    setState("thinking");
    setLogs((prev) => appendLog(prev, `Goal: ${trimmed}`));
    setLogs((prev) => appendLog(prev, "Capturing frame, calling vision model…"));
    simTimerRef.current = setTimeout(() => {
      setState("acting");
      setLogs((prev) => appendLog(prev, "LLM: press F2 to enter setup"));
      setLastAck("key F2 ✓");
      simTimerRef.current = setTimeout(() => {
        setState("idle");
        setLogs((prev) => appendLog(prev, "Action complete — idle"));
        simTimerRef.current = null;
      }, 2000);
    }, 1500);
  };

  const handlePause = () => {
    if (state !== "thinking" && state !== "acting") return;
    clearSim();
    setState("paused");
    setLogs((prev) => appendLog(prev, "Paused after current action (fake)"));
  };

  const handleResume = () => {
    if (state !== "paused") return;
    setState("idle");
    setLogs((prev) => appendLog(prev, "Resumed — awaiting next tick (fake)"));
  };

  const handleStop = () => {
    if (state !== "thinking" && state !== "acting" && state !== "paused") return;
    clearSim();
    setState("idle");
    setLogs((prev) => appendLog(prev, "Stopped — goal cleared (fake)"));
    setLastAck(null);
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-white dark:bg-zinc-950">
      <Header
        mode={mode}
        state={state}
        wsConnected={wsConnected}
        piConnected={piConnected}
        onModeChange={handleModeChange}
      />

      <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 lg:flex-row">
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <VideoPane
            mode={mode}
            piConnected={piConnected}
            focused={videoFocused}
            onFocus={() => setVideoFocused(true)}
            onBlur={() => setVideoFocused(false)}
            useStream={false}
          />
        </div>

        <aside className="flex w-full shrink-0 flex-col gap-4 lg:h-[min(100%,calc(100vh-8rem))] lg:w-80 xl:w-96">
          <ControlRail
            mode={mode}
            state={state}
            goal={goal}
            onGoalChange={setGoal}
            onRun={handleRun}
            onPause={handlePause}
            onResume={handleResume}
            onStop={handleStop}
          />
          <ActionLog logs={logs} />
        </aside>
      </div>

      {lastAck && mode === "ai" && (
        <footer className="shrink-0 border-t border-zinc-200 px-4 py-2 text-xs text-zinc-500 dark:border-zinc-800">
          Last ack:{" "}
          <span className="font-mono text-zinc-700 dark:text-zinc-300">{lastAck}</span>
          <span className="ml-2 text-zinc-400">(fake)</span>
        </footer>
      )}
    </div>
  );
}
