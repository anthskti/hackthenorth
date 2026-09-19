"use client";

import { useEffect, useRef } from "react";
import type { LogEntry } from "@/lib/types";

function formatTime(ts: number) {
  return new Date(ts).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

type ActionLogProps = {
  logs: LogEntry[];
};

export function ActionLog({ logs }: ActionLogProps) {
  const endRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  return (
    <div className="flex h-[160px] min-h-[160px] shrink-0 flex-col border-t border-[var(--rule)]">
      <div className="flex items-center justify-between border-b border-[var(--rule)] px-3 py-1">
        <h2 className="font-mono text-[10px] uppercase tracking-[0.25em]">
          Log
        </h2>
        <span className="font-mono text-[10px] tabular-nums opacity-60">
          {logs.length}
        </span>
      </div>
      <div
        className="min-h-0 flex-1 overflow-y-auto p-2 font-mono text-[11px] leading-relaxed"
        role="log"
        aria-live="polite"
      >
        {logs.length === 0 ? (
          <p className="opacity-50">No entries yet.</p>
        ) : (
          <ul className="space-y-1">
            {logs.map((entry, i) => (
              <li key={`${entry.ts}-${i}`}>
                <span className="opacity-50">[{formatTime(entry.ts)}]</span>{" "}
                <span className="whitespace-pre-wrap break-all">{entry.text}</span>
              </li>
            ))}
          </ul>
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}
