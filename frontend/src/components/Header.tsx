"use client";

import Link from "next/link";
import type { AgentState, Mode } from "@/lib/types";

function ConnectionMark({
  label,
  connected,
}: {
  label: string;
  connected: boolean;
}) {
  return (
    <span className="inline-flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-wide">
      <span
        className={`h-2 w-2 border border-[var(--rule)] ${
          connected ? "bg-[var(--foreground)]" : "bg-transparent"
        }`}
        aria-hidden
      />
      {label}
    </span>
  );
}

type HeaderProps = {
  systemName: string;
  mode: Mode;
  state: AgentState | null;
  wsConnected: boolean;
  piConnected: boolean;
  onModeChange: (mode: Mode) => void;
};

export function Header({
  systemName,
  mode,
  state,
  wsConnected,
  piConnected,
  onModeChange,
}: HeaderProps) {
  return (
    <header
      className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-[var(--rule)] px-3 py-2"
    >
      <div className="flex flex-wrap items-center gap-4">
        <Link
          href="/"
          className="font-mono text-sm font-semibold uppercase tracking-widest hover:underline"
        >
          BTOS
        </Link>
        <span className="font-mono text-xs opacity-70">{systemName}</span>
        <div
          className="inline-flex border border-[var(--rule)]"
          role="group"
          aria-label="Control mode"
        >
          {(["ai", "manual"] as const).map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => onModeChange(m)}
              className={`px-3 py-1 font-mono text-[10px] uppercase tracking-widest ${
                mode === m
                  ? "bg-[var(--foreground)] text-[var(--background)]"
                  : "hover:bg-[var(--foreground)]/10"
              }`}
              aria-pressed={mode === m}
            >
              {m === "ai" ? "AI" : "Manual"}
            </button>
          ))}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <ConnectionMark label="Pi" connected={piConnected} />
        <ConnectionMark label="WS" connected={wsConnected} />
        {mode === "ai" && state !== null && (
          <span
            className="border border-[var(--rule)] px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide"
          >
            {state}
          </span>
        )}
      </div>
    </header>
  );
}
