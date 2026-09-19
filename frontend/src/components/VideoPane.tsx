"use client";

import type { Mode } from "@/lib/types";

type VideoPaneProps = {
  mode: Mode;
  piConnected: boolean;
  focused: boolean;
  onFocus: () => void;
  onBlur: () => void;
  /** When false, show placeholder instead of stream img (shell / offline dev). */
  useStream?: boolean;
};

export function VideoPane({
  mode,
  piConnected,
  focused,
  onFocus,
  onBlur,
  useStream = false,
}: VideoPaneProps) {
  const manualActive = mode === "manual";
  const badgeLabel = piConnected ? "LIVE" : "Pi disconnected";

  return (
    <div
      className={`relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-zinc-900 dark:border-zinc-800 ${
        manualActive
          ? "outline-none ring-2 ring-transparent focus-within:ring-emerald-500/60"
          : ""
      }`}
    >
      <div
        className={`relative aspect-video w-full bg-zinc-950 ${
          manualActive ? "cursor-crosshair" : ""
        }`}
        tabIndex={manualActive ? 0 : -1}
        onFocus={onFocus}
        onBlur={onBlur}
        role={manualActive ? "application" : undefined}
        aria-label={
          manualActive
            ? "Remote screen — click here to send keyboard and mouse input"
            : "Remote screen"
        }
      >
        {useStream ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src="/video/stream"
            alt="Target machine live feed"
            className="absolute inset-0 h-full w-full object-contain"
          />
        ) : (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center text-zinc-500">
            <p className="text-sm font-medium text-zinc-400">Video placeholder</p>
            <p className="max-w-sm text-xs">
              Stream will load from{" "}
              <code className="rounded bg-zinc-800 px-1 py-0.5 text-zinc-300">
                /video/stream
              </code>{" "}
              after wire-live
            </p>
          </div>
        )}

        <div
          className={`absolute left-3 top-3 rounded px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${
            piConnected
              ? "bg-red-600/90 text-white"
              : "bg-zinc-700/90 text-zinc-200"
          }`}
        >
          {badgeLabel}
        </div>

        {manualActive && focused && (
          <div className="absolute bottom-3 left-3 rounded bg-emerald-600/90 px-2 py-1 text-[10px] font-medium text-white">
            Manual control active
          </div>
        )}
      </div>
    </div>
  );
}
