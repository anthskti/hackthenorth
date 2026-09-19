"use client";

import { useCallback, useEffect, useRef } from "react";
import { VIDEO_STREAM_PATH } from "@/lib/backend";
import type { ManualInputPayload, Mode } from "@/lib/types";

type VideoPaneProps = {
  mode: Mode;
  piConnected: boolean;
  focused: boolean;
  onFocus: () => void;
  onBlur: () => void;
  useStream?: boolean;
  onManualInput?: (payload: ManualInputPayload) => void;
};

export function VideoPane({
  mode,
  piConnected,
  focused,
  onFocus,
  onBlur,
  useStream = false,
  onManualInput,
}: VideoPaneProps) {
  const manualActive = mode === "manual";
  const badgeLabel = piConnected ? "LIVE" : "Pi disconnected";
  const paneRef = useRef<HTMLDivElement>(null);
  const pendingMoveRef = useRef<{ x: number; y: number } | null>(null);
  const rafRef = useRef<number | null>(null);

  const videoCoords = useCallback((clientX: number, clientY: number) => {
    const el = paneRef.current;
    if (!el) return { x: 0, y: 0 };
    const rect = el.getBoundingClientRect();
    const x = Math.round(clientX - rect.left);
    const y = Math.round(clientY - rect.top);
    return {
      x: Math.max(0, Math.min(x, Math.round(rect.width))),
      y: Math.max(0, Math.min(y, Math.round(rect.height))),
    };
  }, []);

  const flushMove = useCallback(() => {
    rafRef.current = null;
    if (!manualActive || !onManualInput || !pendingMoveRef.current) return;
    const { x, y } = pendingMoveRef.current;
    pendingMoveRef.current = null;
    onManualInput({ action: "mouse_move", x, y });
  }, [manualActive, onManualInput]);

  const onMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!manualActive || !focused || !onManualInput) return;
      pendingMoveRef.current = videoCoords(e.clientX, e.clientY);
      if (rafRef.current === null) {
        rafRef.current = requestAnimationFrame(flushMove);
      }
    },
    [manualActive, focused, onManualInput, videoCoords, flushMove],
  );

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!manualActive || !onManualInput) return;
      e.preventDefault();
      const { x, y } = videoCoords(e.clientX, e.clientY);
      const button =
        e.button === 2 ? "right" : e.button === 1 ? "middle" : "left";
      onManualInput({ action: "mouse_click", button, x, y });
    },
    [manualActive, onManualInput, videoCoords],
  );

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (!manualActive || !onManualInput) return;
      e.preventDefault();
      if (e.key === "Tab" || e.key.length !== 1) {
        const special = keyToValue(e);
        if (special) {
          onManualInput({ action: "key", value: special });
        }
        return;
      }
      onManualInput({ action: "key", value: e.key });
    },
    [manualActive, onManualInput],
  );

  useEffect(() => {
    return () => {
      if (rafRef.current !== null) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  return (
    <div
      className={`relative flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-zinc-200 bg-zinc-900 dark:border-zinc-800 ${
        manualActive
          ? "outline-none ring-2 ring-transparent focus-within:ring-emerald-500/60"
          : ""
      }`}
    >
      <div
        ref={paneRef}
        className={`relative aspect-video w-full bg-zinc-950 ${
          manualActive ? "cursor-crosshair" : ""
        }`}
        tabIndex={manualActive ? 0 : -1}
        onFocus={onFocus}
        onBlur={onBlur}
        onMouseMove={onMouseMove}
        onMouseDown={onMouseDown}
        onKeyDown={onKeyDown}
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
            src={VIDEO_STREAM_PATH}
            alt="Target machine live feed"
            className="pointer-events-none absolute inset-0 h-full w-full object-contain"
          />
        ) : (
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center text-zinc-500">
            <p className="text-sm font-medium text-zinc-400">Video placeholder</p>
            <p className="max-w-sm text-xs">
              Stream loads from{" "}
              <code className="rounded bg-zinc-800 px-1 py-0.5 text-zinc-300">
                /video/stream
              </code>{" "}
              when the backend is up
            </p>
          </div>
        )}

        <div
          className={`pointer-events-none absolute left-3 top-3 rounded px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${
            piConnected
              ? "bg-red-600/90 text-white"
              : "bg-zinc-700/90 text-zinc-200"
          }`}
        >
          {badgeLabel}
        </div>

        {manualActive && focused && (
          <div className="pointer-events-none absolute bottom-3 left-3 rounded bg-emerald-600/90 px-2 py-1 text-[10px] font-medium text-white">
            Manual control active
          </div>
        )}
      </div>
    </div>
  );
}

function keyToValue(e: React.KeyboardEvent): string | null {
  switch (e.key) {
    case "Enter":
      return "ENTER";
    case "Backspace":
      return "BACKSPACE";
    case "Escape":
      return "ESC";
    case "ArrowUp":
      return "UP";
    case "ArrowDown":
      return "DOWN";
    case "ArrowLeft":
      return "LEFT";
    case "ArrowRight":
      return "RIGHT";
    case " ":
      return "SPACE";
    default:
      if (e.key.startsWith("F") && e.key.length <= 3) {
        return e.key.toUpperCase();
      }
      return null;
  }
}
