"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import type { AgentState, LogEntry, Mode } from "@/lib/types";
import { useDashboardLive } from "@/lib/useDashboardLive";
import { ActionLog } from "./ActionLog";
import { ControlRail } from "./ControlRail";
import { Header } from "./Header";
import { VideoPane } from "./VideoPane";

function appendLog(logs: LogEntry[], text: string): LogEntry[] {
  return [...logs, { text, ts: Date.now() }];
}

export function Dashboard() {
  const {
    mode,
    state,
    logs,
    wsConnected,
    piConnected,
    backendReachable,
    setMode,
    setState,
    setLogs,
  } = useDashboardLive();

  const [goal, setGoal] = useState("");
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
        appendLog(prev, "Switched to manual (local — wire-controls next)"),
      );
    } else {
      setState("idle");
      setLogs((prev) => appendLog(prev, "Switched to AI (local — wire-controls next)"));
    }
  };

  const handleRun = () => {
    const trimmed = goal.trim();
    if (mode !== "ai" || state !== "idle" || !trimmed) return;
    clearSim();
    setState("thinking");
    setLogs((prev) => appendLog(prev, `Goal: ${trimmed}`));
    setLogs((prev) => appendLog(prev, "Capturing frame, calling vision model… (local mock)"));
    simTimerRef.current = setTimeout(() => {
      setState("acting");
      setLogs((prev) => appendLog(prev, "LLM: press F2 to enter setup (local mock)"));
      setLastAck("key F2 ✓");
      simTimerRef.current = setTimeout(() => {
        setState("idle");
        setLogs((prev) => appendLog(prev, "Action complete — idle (local mock)"));
        simTimerRef.current = null;
      }, 2000);
    }, 1500);
  };

  const handlePause = () => {
    if (state !== "thinking" && state !== "acting") return;
    clearSim();
    setState("paused");
    setLogs((prev) => appendLog(prev, "Paused (local mock)"));
  };

  const handleResume = () => {
    if (state !== "paused") return;
    setState("idle");
    setLogs((prev) => appendLog(prev, "Resumed (local mock)"));
  };

  const handleStop = () => {
    if (state !== "thinking" && state !== "acting" && state !== "paused") return;
    clearSim();
    setState("idle");
    setLogs((prev) => appendLog(prev, "Stopped (local mock)"));
    setLastAck(null);
  };

  const showStream = backendReachable && piConnected;

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-white dark:bg-zinc-950">
      <Header
        mode={mode}
        state={state}
        wsConnected={wsConnected}
        piConnected={piConnected}
        onModeChange={handleModeChange}
      />

      {!backendReachable && (
        <p className="bg-amber-50 px-4 py-2 text-center text-sm text-amber-900 dark:bg-amber-950 dark:text-amber-100">
          Backend unreachable — start Go on :8080 and refresh. Showing last local state.
        </p>
      )}

      <div className="flex min-h-0 flex-1 flex-col gap-4 p-4 lg:flex-row">
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <VideoPane
            mode={mode}
            piConnected={piConnected}
            focused={videoFocused}
            onFocus={() => setVideoFocused(true)}
            onBlur={() => setVideoFocused(false)}
            useStream={showStream}
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
          <span className="ml-2 text-zinc-400">(local mock)</span>
        </footer>
      )}
    </div>
  );
}
