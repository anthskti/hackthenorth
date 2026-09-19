"use client";

import { useState } from "react";
import { fetchStatus, postControl, postMode, postPrompt } from "@/lib/api";
import { useDashboardLive } from "@/lib/useDashboardLive";
import type { Mode } from "@/lib/types";
import { ActionLog } from "./ActionLog";
import { ControlRail } from "./ControlRail";
import { Header } from "./Header";
import { VideoPane } from "./VideoPane";

export function Dashboard() {
  const {
    mode,
    state,
    logs,
    wsConnected,
    piConnected,
    backendReachable,
    applySnapshotFromServer,
    sendManualInput,
  } = useDashboardLive();

  const [goal, setGoal] = useState("");
  const [videoFocused, setVideoFocused] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const showStream = backendReachable && piConnected;

  const runAction = async (fn: () => Promise<unknown>) => {
    setActionError(null);
    try {
      const snap = await fn();
      applySnapshotFromServer(snap as Awaited<ReturnType<typeof fetchStatus>>);
    } catch (e) {
      setActionError(e instanceof Error ? e.message : "Request failed");
    }
  };

  const handleModeChange = (next: Mode) => {
    void runAction(() => postMode(next));
  };

  const handleRun = () => {
    const trimmed = goal.trim();
    if (mode !== "ai" || state !== "idle" || !trimmed) return;
    void runAction(async () => {
      const snap = await postPrompt(trimmed);
      setGoal("");
      return snap;
    });
  };

  const handlePause = () => {
    void runAction(() => postControl("pause"));
  };

  const handleResume = () => {
    void runAction(() => postControl("resume"));
  };

  const handleStop = () => {
    void runAction(() => postControl("stop"));
  };

  const manualInput =
    mode === "manual" && videoFocused ? sendManualInput : undefined;

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
          Backend unreachable. Start Go on :8080 and refresh.
        </p>
      )}

      {actionError && (
        <p className="bg-red-50 px-4 py-2 text-center text-sm text-red-800 dark:bg-red-950 dark:text-red-200">
          {actionError}
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
            onManualInput={manualInput}
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
    </div>
  );
}
