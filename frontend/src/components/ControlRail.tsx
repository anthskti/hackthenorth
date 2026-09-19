"use client";

import type { AgentState, Mode } from "@/lib/types";

type ControlRailProps = {
  mode: Mode;
  state: AgentState | null;
  goal: string;
  onGoalChange: (goal: string) => void;
  onRun: () => void;
  onPause: () => void;
  onResume: () => void;
  onStop: () => void;
};

function btnBase(disabled: boolean) {
  return `rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
    disabled
      ? "cursor-not-allowed bg-zinc-100 text-zinc-400 dark:bg-zinc-900 dark:text-zinc-600"
      : "bg-zinc-900 text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
  }`;
}

function btnSecondary(disabled: boolean) {
  return `rounded-lg border px-3 py-2 text-sm font-medium transition-colors ${
    disabled
      ? "cursor-not-allowed border-zinc-200 text-zinc-400 dark:border-zinc-800"
      : "border-zinc-300 text-zinc-800 hover:bg-zinc-50 dark:border-zinc-600 dark:text-zinc-200 dark:hover:bg-zinc-800"
  }`;
}

export function ControlRail({
  mode,
  state,
  goal,
  onGoalChange,
  onRun,
  onPause,
  onResume,
  onStop,
}: ControlRailProps) {
  const isAi = mode === "ai";
  const canRun = isAi && state === "idle" && goal.trim().length > 0;
  const canPause = isAi && (state === "thinking" || state === "acting");
  const canResume = isAi && state === "paused";
  const canStop =
    isAi && (state === "thinking" || state === "acting" || state === "paused");

  return (
    <div className="flex shrink-0 flex-col gap-4">
      {mode === "manual" ? (
        <p className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 px-3 py-3 text-sm text-zinc-600 dark:border-zinc-700 dark:bg-zinc-900/50 dark:text-zinc-400">
          Click the video pane and use your keyboard and mouse to control the
          target. Input is sent only while the pane is focused.
        </p>
      ) : (
        <>
          <div className="flex flex-col gap-2">
            <label htmlFor="goal" className="text-xs font-medium text-zinc-500">
              Goal
            </label>
            <textarea
              id="goal"
              rows={3}
              value={goal}
              onChange={(e) => onGoalChange(e.target.value)}
              placeholder="e.g. Open BIOS and enable virtualization"
              disabled={state !== "idle"}
              className="resize-none rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 placeholder:text-zinc-400 focus:border-zinc-400 focus:outline-none focus:ring-1 focus:ring-zinc-400 disabled:opacity-60 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
            />
            <button
              type="button"
              disabled={!canRun}
              onClick={onRun}
              className={btnBase(!canRun)}
            >
              Run
            </button>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={!canPause}
              onClick={onPause}
              className={btnSecondary(!canPause)}
            >
              Pause
            </button>
            <button
              type="button"
              disabled={!canResume}
              onClick={onResume}
              className={btnSecondary(!canResume)}
            >
              Resume
            </button>
            <button
              type="button"
              disabled={!canStop}
              onClick={onStop}
              className={btnSecondary(!canStop)}
            >
              Stop
            </button>
          </div>
        </>
      )}
    </div>
  );
}
