"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { VIDEO_STREAM_PATH } from "@/lib/backend";
import { hidHex, kc, queueKeyLine } from "@/lib/kvmHid";
import type { Mode } from "@/lib/types";

type VideoPaneProps = {
  mode: Mode;
  piConnected: boolean;
  focused: boolean;
  onFocus: () => void;
  onBlur: () => void;
  useStream?: boolean;
  noUplink?: boolean;
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
}: VideoPaneProps) {
  const manualActive = mode === "manual" && !noUplink;
  const keyboardReady = manualActive;
  const badgeLabel = noUplink
    ? "NO UPLINK"
    : piConnected
      ? "LIVE"
      : "PI DISCONNECTED";
  const liveRef = useRef<HTMLInputElement>(null);
  const heldRef = useRef<Set<number>>(new Set());
  const [hidStatus, setHidStatus] = useState("");

  useEffect(() => {
    if (keyboardReady) {
      liveRef.current?.focus();
    } else {
      heldRef.current.clear();
    }
  }, [keyboardReady]);

  const send = useCallback((line: string) => {
    queueKeyLine(line, setHidStatus);
  }, []);

  const onLiveKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      e.preventDefault();
      if (!keyboardReady || e.repeat) return;
      const k = kc(e.code);
      if (k === null) {
        setHidStatus("unmapped key: " + e.code);
        return;
      }
      heldRef.current.add(k);
      send("p" + hidHex(k));
    },
    [keyboardReady, send],
  );

  const onLiveKeyUp = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      e.preventDefault();
      const k = kc(e.code);
      if (k === null || !heldRef.current.has(k)) return;
      heldRef.current.delete(k);
      send("r" + hidHex(k));
    },
    [send],
  );

  const onLiveBlur = useCallback(() => {
    heldRef.current.clear();
    if (keyboardReady) send("a");
    onBlur();
  }, [keyboardReady, send, onBlur]);

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

      <div className="relative min-h-0 w-full flex-1 bg-black">
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
      </div>

      {manualActive && (
        <div className="flex shrink-0 flex-col gap-1 border-t border-[var(--rule)] bg-[var(--background)] p-2">
          <input
            ref={liveRef}
            type="text"
            autoComplete="off"
            disabled={!keyboardReady}
            placeholder="Click here and type: every key is sent live"
            onFocus={onFocus}
            onBlur={onLiveBlur}
            onKeyDown={onLiveKeyDown}
            onKeyUp={onLiveKeyUp}
            className="w-full border border-[var(--rule)] bg-transparent px-2 py-2 text-sm placeholder:opacity-40 focus:outline focus:outline-1 focus:outline-[var(--rule)] disabled:opacity-50"
            aria-label="Live keyboard to target"
          />
          <p className="font-mono text-[10px] opacity-60">
            {hidStatus || "Keys go through the backend to Pi /key (same lines as /ui)"}
          </p>
          {focused && keyboardReady && (
            <p className="font-mono text-[10px] uppercase tracking-wide">
              Manual control active
            </p>
          )}
        </div>
      )}
    </div>
  );
}
