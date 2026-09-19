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
    <div className="flex min-h-0 flex-1 flex-col">
      <h2 className="mb-2 text-xs font-medium uppercase tracking-wide text-zinc-500">
        Action log
      </h2>
      <div
        className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-zinc-200 bg-zinc-50 p-3 font-mono text-xs leading-relaxed dark:border-zinc-800 dark:bg-zinc-950"
        role="log"
        aria-live="polite"
      >
        {logs.length === 0 ? (
          <p className="text-zinc-400">No entries yet.</p>
        ) : (
          <ul className="space-y-2">
            {logs.map((entry, i) => (
              <li key={`${entry.ts}-${i}`} className="text-zinc-700 dark:text-zinc-300">
                <span className="text-zinc-400 dark:text-zinc-500">
                  [{formatTime(entry.ts)}]
                </span>{" "}
                {entry.text}
              </li>
            ))}
          </ul>
        )}
        <div ref={endRef} />
      </div>
    </div>
  );
}
