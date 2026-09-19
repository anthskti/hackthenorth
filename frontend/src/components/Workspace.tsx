"use client";

import { useState } from "react";
import { fetchStatus, postControl, postMode, postPrompt } from "@/lib/api";
import { useDashboardLive } from "@/lib/useDashboardLive";
import { isLiveSystem } from "@/lib/systems";
import type { System } from "@/lib/systems";
import type { Mode } from "@/lib/types";
import { ActionLog } from "./ActionLog";
import { ChatPanel } from "./ChatPanel";
import { Header } from "./Header";
import { SystemsRail } from "./SystemsRail";
import { VideoPane } from "./VideoPane";

type WorkspaceProps = {
  system: System;
};

export function Workspace({ system }: WorkspaceProps) {
  const live = isLiveSystem(system.id);
  const {
    mode,
    state,
    logs,
    wsConnected,
    piConnected,
    backendReachable,
    applySnapshotFromServer,
    sendManualInput,
  } = useDashboardLive({ enabled: live });

  const [videoFocused, setVideoFocused] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const showStream = live && backendReachable && piConnected;

  const runAction = async (fn: () => Promise<unknown>) => {
    if (!live) return;
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

  const handleRun = (goal: string) => {
    const trimmed = goal.trim();
    if (!live || mode !== "ai" || state !== "idle" || !trimmed) return;
    void runAction(() => postPrompt(trimmed));
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
    live && mode === "manual" && videoFocused ? sendManualInput : undefined;

  const displayMode = live ? mode : "ai";
  const displayState = live ? state : null;
  const displayLogs = live ? logs : [];
  const displayWs = live && wsConnected;
  const displayPi = live && piConnected;

  return (
    <div className="workspace-shell flex h-dvh min-h-0 flex-col bg-background text-foreground">
      <Header
        systemName={system.name}
        mode={displayMode}
        state={displayState}
        wsConnected={displayWs}
        piConnected={displayPi}
        onModeChange={live ? handleModeChange : () => {}}
      />

      {live && !backendReachable && (
        <p className="border-b border-[var(--rule)] px-3 py-2 text-center text-xs">
          Backend unreachable. Start Go on :8080 and refresh.
        </p>
      )}

      {actionError && (
        <p className="border-b border-accent bg-accent/10 px-3 py-2 text-center text-xs text-accent">
          {actionError}
        </p>
      )}

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <SystemsRail activeId={system.id} />

        <ChatPanel
          systemId={system.id}
          mode={displayMode}
          state={displayState}
          onRun={handleRun}
          onPause={handlePause}
          onResume={handleResume}
          onStop={handleStop}
        />

        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1 p-2">
            <VideoPane
              mode={displayMode}
              piConnected={displayPi}
              focused={videoFocused}
              onFocus={() => setVideoFocused(true)}
              onBlur={() => setVideoFocused(false)}
              useStream={showStream}
              noUplink={!live}
              onManualInput={manualInput}
            />
          </div>
          <ActionLog logs={displayLogs} />
        </div>
      </div>
    </div>
  );
}
