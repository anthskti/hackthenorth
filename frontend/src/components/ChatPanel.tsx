"use client";

import { useCallback, useEffect, useState } from "react";
import type { AgentState, Mode } from "@/lib/types";
import type { ChatThread } from "@/lib/chatHistory";
import {
  appendMessage,
  appendMessages,
  createThread,
  loadThreads,
} from "@/lib/chatHistory";

type ChatPanelProps = {
  systemId: string;
  mode: Mode;
  state: AgentState | null;
  onRun: (goal: string) => void;
  onPause: () => void;
  onResume: () => void;
  onStop: () => void;
};

function stateLine(state: AgentState | null): string | null {
  if (!state || state === "idle") return null;
  const labels: Record<AgentState, string> = {
    idle: "",
    thinking: "Agent is thinking…",
    acting: "Agent is acting on the target…",
    paused: "Agent paused.",
  };
  return labels[state];
}

export function ChatPanel({
  systemId,
  mode,
  state,
  onRun,
  onPause,
  onResume,
  onStop,
}: ChatPanelProps) {
  const [threads, setThreads] = useState<ChatThread[]>([]);
  const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");

  const refreshThreads = useCallback(() => {
    setThreads(loadThreads(systemId));
  }, [systemId]);

  useEffect(() => {
    refreshThreads();
    const existing = loadThreads(systemId);
    if (existing.length > 0) {
      setActiveThreadId(existing[0].id);
    }
  }, [systemId, refreshThreads]);

  const activeThread = threads.find((t) => t.id === activeThreadId) ?? null;

  const isAi = mode === "ai";
  const canRun = isAi && state === "idle" && draft.trim().length > 0;
  const canPause = isAi && (state === "thinking" || state === "acting");
  const canResume = isAi && state === "paused";
  const canStop =
    isAi && (state === "thinking" || state === "acting" || state === "paused");

  const ensureThread = (): string => {
    if (activeThreadId) return activeThreadId;
    const t = createThread(systemId, "New session");
    refreshThreads();
    setActiveThreadId(t.id);
    return t.id;
  };

  const handleNew = () => {
    const t = createThread(systemId, "New session");
    refreshThreads();
    setActiveThreadId(t.id);
    setDraft("");
  };

  const handleSend = () => {
    const trimmed = draft.trim();
    if (!canRun || !trimmed) return;
    const tid = ensureThread();
    const now = Date.now();
    appendMessages(systemId, tid, [
      { role: "user", text: trimmed, ts: now },
      {
        role: "agent",
        text: "Run started. Watch the log and video for progress.",
        ts: now,
      },
    ]);
    refreshThreads();
    setDraft("");
    onRun(trimmed);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      handleSend();
    }
  };

  const statusMsg = stateLine(state);

  useEffect(() => {
    if (!activeThreadId || !statusMsg) return;
    const thread = loadThreads(systemId).find((t) => t.id === activeThreadId);
    if (!thread) return;
    const last = thread.messages[thread.messages.length - 1];
    if (last?.role === "agent" && last.text === statusMsg) return;
    const updated = appendMessage(systemId, activeThreadId, {
      role: "agent",
      text: statusMsg,
      ts: Date.now(),
    });
    if (updated) refreshThreads();
  }, [systemId, activeThreadId, statusMsg]);

  const selectThread = (id: string) => {
    setActiveThreadId(id);
    const t = loadThreads(systemId).find((x) => x.id === id);
    if (t) {
      const lastUser = [...t.messages].reverse().find((m) => m.role === "user");
      setDraft(lastUser?.text ?? "");
    }
  };

  return (
    <div className="flex h-full min-h-0 w-full max-w-[360px] shrink-0 flex-col border-r border-[var(--rule)]">
      <div className="flex items-center justify-between border-b border-[var(--rule)] px-3 py-2">
        <p className="font-mono text-[10px] uppercase tracking-[0.25em]">
          History
        </p>
        <button
          type="button"
          onClick={handleNew}
          className="border border-[var(--rule)] px-2 py-0.5 font-mono text-[10px] uppercase tracking-wide hover:bg-[var(--foreground)] hover:text-[var(--background)]"
        >
          + New
        </button>
      </div>

      {threads.length > 0 && (
        <ul className="max-h-28 shrink-0 overflow-y-auto border-b border-[var(--rule)]">
          {threads.map((t) => (
            <li key={t.id}>
              <button
                type="button"
                onClick={() => selectThread(t.id)}
                className={`w-full truncate px-3 py-2 text-left text-xs ${
                  t.id === activeThreadId
                    ? "bg-[var(--foreground)] text-[var(--background)]"
                    : "hover:bg-[var(--foreground)]/10"
                }`}
              >
                {t.title}
              </button>
            </li>
          ))}
        </ul>
      )}

      <div
        className="min-h-0 flex-1 overflow-y-auto p-3 font-mono text-xs leading-relaxed"
        role="log"
      >
        {!activeThread || activeThread.messages.length === 0 ? (
          <p className="opacity-50">No messages. Send a goal to start.</p>
        ) : (
          <ul className="space-y-3">
            {activeThread.messages.map((m) => (
              <li
                key={m.id}
                className={`border border-[var(--rule)] p-2 ${
                  m.role === "user"
                    ? "bg-[var(--foreground)]/5"
                    : "border-dashed opacity-90"
                }`}
              >
                <p className="mb-1 text-[10px] uppercase tracking-widest opacity-60">
                  {m.role === "user" ? "You" : "Agent"}
                </p>
                {m.text}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="shrink-0 border-t border-[var(--rule)] p-3">
        {mode === "manual" ? (
          <p className="border border-dashed border-[var(--rule)] p-3 text-xs leading-relaxed opacity-80">
            Focus the video to drive the target. Keyboard and mouse are sent only
            while the pane is focused.
          </p>
        ) : (
          <>
            <div className="mb-2 flex flex-wrap gap-1">
              <button
                type="button"
                disabled={!canPause}
                onClick={onPause}
                className={controlBtn(!canPause)}
              >
                Pause
              </button>
              <button
                type="button"
                disabled={!canResume}
                onClick={onResume}
                className={controlBtn(!canResume)}
              >
                Resume
              </button>
              <button
                type="button"
                disabled={!canStop}
                onClick={onStop}
                className={controlBtn(!canStop)}
              >
                Stop
              </button>
            </div>
            <textarea
              rows={3}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleKeyDown}
              disabled={state !== "idle"}
              placeholder="Goal — e.g. Open BIOS and enable virtualization"
              className="w-full resize-none border border-[var(--rule)] bg-transparent px-2 py-2 text-sm placeholder:opacity-40 focus:outline focus:outline-1 focus:outline-[var(--rule)] disabled:opacity-50"
            />
            <button
              type="button"
              disabled={!canRun}
              onClick={handleSend}
              className={`mt-2 w-full border border-[var(--rule)] py-2 text-xs font-medium uppercase tracking-widest ${
                canRun
                  ? "bg-[var(--foreground)] text-[var(--background)] hover:opacity-90"
                  : "cursor-not-allowed opacity-40"
              }`}
            >
              Run
            </button>
            <p className="mt-1 text-[10px] opacity-50">⌘/Ctrl + Enter</p>
          </>
        )}
      </div>
    </div>
  );
}

function controlBtn(disabled: boolean) {
  return `border border-[var(--rule)] px-2 py-1 text-[10px] uppercase tracking-wide ${
    disabled
      ? "cursor-not-allowed opacity-40"
      : "hover:bg-[var(--foreground)] hover:text-[var(--background)]"
  }`;
}
