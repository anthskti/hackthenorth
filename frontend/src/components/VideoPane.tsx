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
  noUplink?: boolean;
  onManualInput?: (payload: ManualInputPayload) => void;
  /** MJPEG frame size from kvmd (default 16:9 @ 480p → 854×480). */
  streamWidth?: number;
  streamHeight?: number;
};

export function VideoPane({
  mode,
  piConnected,
  focused,
  onFocus,
  onBlur,
  useStream = false,
  noUplink = false,
  onManualInput,
  streamWidth = 854,
  streamHeight = 480,
}: VideoPaneProps) {
  const streamAspect = streamWidth / streamHeight;
  const manualActive = mode === "manual" && !noUplink;
  const badgeLabel = noUplink
    ? "NO UPLINK"
    : piConnected
      ? "LIVE"
      : "PI DISCONNECTED";
  const paneRef = useRef<HTMLDivElement>(null);
  const pendingMoveRef = useRef<{ x: number; y: number } | null>(null);
  const rafRef = useRef<number | null>(null);

  const videoCoords = useCallback((clientX: number, clientY: number) => {
    const el = paneRef.current;
    if (!el) return null;
    const rect = el.getBoundingClientRect();
    const paneW = rect.width;
    const paneH = rect.height;
    if (paneW <= 0 || paneH <= 0) return null;

    const paneAspect = paneW / paneH;
    let videoW: number;
    let videoH: number;
    let offsetX: number;
    let offsetY: number;

    if (paneAspect > streamAspect) {
      videoH = paneH;
      videoW = paneH * streamAspect;
      offsetX = (paneW - videoW) / 2;
      offsetY = 0;
    } else {
      videoW = paneW;
      videoH = paneW / streamAspect;
      offsetX = 0;
      offsetY = (paneH - videoH) / 2;
    }

    const localX = clientX - rect.left - offsetX;
    const localY = clientY - rect.top - offsetY;
    if (localX < 0 || localY < 0 || localX > videoW || localY > videoH) {
      return null;
    }

    const x = Math.round((localX / videoW) * streamWidth);
    const y = Math.round((localY / videoH) * streamHeight);
    return {
      x: Math.max(0, Math.min(streamWidth, x)),
      y: Math.max(0, Math.min(streamHeight, y)),
    };
  }, [streamWidth, streamHeight, streamAspect]);

  const flushMove = useCallback(() => {
    rafRef.current = null;
    if (!manualActive || !onManualInput || !pendingMoveRef.current) return;
    const { x, y } = pendingMoveRef.current;
    pendingMoveRef.current = null;
    onManualInput({ action: "mouse_move", x, y });
  }, [manualActive, onManualInput]);

  const onMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!manualActive || !onManualInput) return;
      const coords = videoCoords(e.clientX, e.clientY);
      if (!coords) return;
      pendingMoveRef.current = coords;
      if (rafRef.current === null) {
        rafRef.current = requestAnimationFrame(flushMove);
      }
    },
    [manualActive, onManualInput, videoCoords, flushMove],
  );

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!manualActive || !onManualInput) return;
      paneRef.current?.focus();
      const coords = videoCoords(e.clientX, e.clientY);
      if (!coords) return;
      e.preventDefault();
      const button =
        e.button === 2 ? "right" : e.button === 1 ? "middle" : "left";
      onManualInput({
        action: "mouse_click",
        button,
        x: coords.x,
        y: coords.y,
      });
    },
    [manualActive, onManualInput, videoCoords],
  );

  const onContextMenu = useCallback(
    (e: React.MouseEvent) => {
      if (manualActive) e.preventDefault();
    },
    [manualActive],
  );

  const onWheel = useCallback(
    (e: React.WheelEvent) => {
      if (!manualActive || !onManualInput) return;
      e.preventDefault();
      const delta = Math.round(-e.deltaY / 40);
      if (delta === 0) return;
      onManualInput({ action: "mouse_wheel", delta });
    },
    [manualActive, onManualInput],
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

  const showLiveBadge = !noUplink && piConnected;

  return (
    <div
      className={`relative flex min-h-0 flex-1 flex-col overflow-hidden border border-[var(--rule)] ${
        manualActive
          ? "outline-none focus-within:outline focus-within:outline-1 focus-within:outline-[var(--rule)]"
          : ""
      }`}
    >
      <span
        className="pointer-events-none absolute left-0 top-0 z-10 h-4 w-4 border-l border-t border-[var(--rule)]"
        aria-hidden
      />
      <span
        className="pointer-events-none absolute right-0 top-0 z-10 h-4 w-4 border-r border-t border-[var(--rule)]"
        aria-hidden
      />
      <span
        className="pointer-events-none absolute bottom-0 left-0 z-10 h-4 w-4 border-b border-l border-[var(--rule)]"
        aria-hidden
      />
      <span
        className="pointer-events-none absolute bottom-0 right-0 z-10 h-4 w-4 border-b border-r border-[var(--rule)]"
        aria-hidden
      />

      <div
        ref={paneRef}
        className={`relative min-h-0 w-full flex-1 bg-black ${
          manualActive ? "cursor-crosshair" : ""
        }`}
        tabIndex={manualActive ? 0 : -1}
        onFocus={onFocus}
        onBlur={onBlur}
        onMouseMove={onMouseMove}
        onMouseDown={onMouseDown}
        onContextMenu={onContextMenu}
        onWheel={onWheel}
        onKeyDown={onKeyDown}
        role={manualActive ? "application" : undefined}
        aria-label={
          manualActive
            ? "Remote screen — move mouse to hover, click to send pointer; click then type for keys"
            : "Remote screen"
        }
      >
        {noUplink ? (
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center text-sm opacity-60">
            <p className="font-mono text-xs uppercase tracking-widest">
              No uplink
            </p>
            <p className="max-w-sm text-xs">
              This system is not connected to the live backend. Open the live
              system to control hardware.
            </p>
          </div>
        ) : useStream ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={VIDEO_STREAM_PATH}
            alt="Target machine live feed"
            className="pointer-events-none absolute inset-0 h-full w-full object-contain"
          />
        ) : (
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-2 px-6 text-center opacity-60">
            <p className="font-mono text-xs uppercase tracking-widest">
              Video placeholder
            </p>
            <p className="max-w-sm text-xs">
              Stream loads from{" "}
              <code className="border border-[var(--rule)] px-1">/video/stream</code>{" "}
              when the backend is up
            </p>
          </div>
        )}

        <div
          className={`pointer-events-none absolute left-3 top-3 px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider ${
            showLiveBadge
              ? "bg-accent text-white"
              : "border border-[var(--rule)] bg-[var(--background)]"
          }`}
        >
          {badgeLabel}
        </div>

        {manualActive && focused && (
          <div className="pointer-events-none absolute bottom-3 left-3 border border-[var(--rule)] bg-[var(--foreground)] px-2 py-1 font-mono text-[10px] uppercase tracking-wide text-[var(--background)]">
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
