"use client";

import type { AgentState, Mode } from "@/lib/types";

function ConnectionDot({ label, connected }: { label: string; connected: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-zinc-500 dark:text-zinc-400">
      <span
        className={`h-2 w-2 rounded-full ${
          connected ? "bg-emerald-500" : "bg-zinc-400 dark:bg-zinc-600"
        }`}
        aria-hidden
      />
      {label}
    </span>
  );
}

const stateStyles: Record<AgentState, string> = {
  idle: "bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300",
  thinking: "bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-200",
  acting: "bg-emerald-100 text-emerald-900 dark:bg-emerald-950 dark:text-emerald-200",
  paused: "bg-orange-100 text-orange-900 dark:bg-orange-950 dark:text-orange-200",
};

type HeaderProps = {
  mode: Mode;
  state: AgentState | null;
  wsConnected: boolean;
  piConnected: boolean;
  onModeChange: (mode: Mode) => void;
};

export function Header({
  mode,
  state,
  wsConnected,
  piConnected,
  onModeChange,
}: HeaderProps) {
  return (
    <header
      className="flex shrink-0 flex-wrap items-center justify-between gap-4 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800"
    >
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="text-lg font-semibold tracking-tight">BtOS</h1>
        <div
          className="inline-flex rounded-lg border border-zinc-200 p-0.5 dark:border-zinc-700"
          role="group"
          aria-label="Control mode"
        >
          {(["ai", "manual"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => onModeChange(m)}
              className={`rounded-md px-3 py-1.5 text-sm font-medium capitalize transition-colors ${
                mode === m
                  ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                  : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
              }`}
              aria-pressed={mode === m}
            >
              {m === "ai" ? "AI" : "Manual"}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <ConnectionDot label="Pi" connected={piConnected} />
        <ConnectionDot label="WS" connected={wsConnected} />
        {mode === "ai" && state !== null && (
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${stateStyles[state]}`}
          >
            {state}
          </span>
        )}
      </div>
    </header>
  );
}
