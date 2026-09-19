export type ChatMessage = {
  id: string;
  role: "user" | "agent";
  text: string;
  ts: number;
};

export type ChatThread = {
  id: string;
  title: string;
  messages: ChatMessage[];
  createdAt: number;
  updatedAt: number;
};

const STORAGE_PREFIX = "btos-chat-";

let idSeq = 0;

function newId(prefix: string): string {
  idSeq += 1;
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${idSeq}-${Math.random().toString(36).slice(2, 9)}`;
}

function storageKey(systemId: string): string {
  return `${STORAGE_PREFIX}${systemId}`;
}

function safeParse(raw: string | null): ChatThread[] {
  if (!raw) return [];
  try {
    const data = JSON.parse(raw) as ChatThread[];
    return Array.isArray(data) ? data : [];
  } catch {
    return [];
  }
}

/** Reassign ids when legacy saves used `m-${Date.now()}` twice in one tick. */
function normalizeThreads(threads: ChatThread[]): {
  threads: ChatThread[];
  repaired: boolean;
} {
  let repaired = false;
  const normalized = threads.map((thread) => {
    const seen = new Set<string>();
    const messages = thread.messages.map((m) => {
      if (!seen.has(m.id)) {
        seen.add(m.id);
        return m;
      }
      repaired = true;
      const id = newId("m");
      seen.add(id);
      return { ...m, id };
    });
    return { ...thread, messages };
  });
  return { threads: normalized, repaired };
}

export function loadThreads(systemId: string): ChatThread[] {
  if (typeof window === "undefined") return [];
  const raw = safeParse(localStorage.getItem(storageKey(systemId)));
  const { threads, repaired } = normalizeThreads(raw);
  if (repaired) {
    saveThreads(systemId, threads);
  }
  return threads;
}

export function saveThreads(systemId: string, threads: ChatThread[]): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(storageKey(systemId), JSON.stringify(threads));
}

export function createThread(systemId: string, title: string): ChatThread {
  const now = Date.now();
  const thread: ChatThread = {
    id: newId("t"),
    title,
    messages: [],
    createdAt: now,
    updatedAt: now,
  };
  const threads = [thread, ...loadThreads(systemId)];
  saveThreads(systemId, threads);
  return thread;
}

export function appendMessage(
  systemId: string,
  threadId: string,
  message: Omit<ChatMessage, "id">,
): ChatThread | null {
  const threads = loadThreads(systemId);
  const idx = threads.findIndex((t) => t.id === threadId);
  if (idx === -1) return null;
  const thread = threads[idx];
  const msg: ChatMessage = { ...message, id: newId("m") };
  return pushMessagesToThread(threads, idx, [msg], systemId);
}

function pushMessagesToThread(
  threads: ChatThread[],
  idx: number,
  newMessages: ChatMessage[],
  systemId: string,
): ChatThread | null {
  const thread = threads[idx];
  if (!thread) return null;
  const firstUser = newMessages.find((m) => m.role === "user");
  const updated: ChatThread = {
    ...thread,
    messages: [...thread.messages, ...newMessages],
    updatedAt: Date.now(),
    title:
      thread.messages.length === 0 && firstUser
        ? firstUser.text.slice(0, 48) +
          (firstUser.text.length > 48 ? "…" : "")
        : thread.title,
  };
  threads[idx] = updated;
  saveThreads(systemId, threads);
  return updated;
}

export function appendMessages(
  systemId: string,
  threadId: string,
  messages: Omit<ChatMessage, "id">[],
): ChatThread | null {
  if (messages.length === 0) return null;
  const threads = loadThreads(systemId);
  const idx = threads.findIndex((t) => t.id === threadId);
  if (idx === -1) return null;
  const withIds: ChatMessage[] = messages.map((m) => ({
    ...m,
    id: newId("m"),
  }));
  return pushMessagesToThread(threads, idx, withIds, systemId);
}

export function updateThreadMessages(
  systemId: string,
  threadId: string,
  messages: ChatMessage[],
): void {
  const threads = loadThreads(systemId);
  const idx = threads.findIndex((t) => t.id === threadId);
  if (idx === -1) return;
  threads[idx] = { ...threads[idx], messages, updatedAt: Date.now() };
  saveThreads(systemId, threads);
}
