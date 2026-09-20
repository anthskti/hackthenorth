import { apiUrl } from "@/lib/backend";

/** Same key mapping as embedded/qnx/kvmd.c /ui. */

const SPECIAL: Record<string, number> = {
  Enter: 0xb0,
  NumpadEnter: 0xb0,
  Escape: 0xb1,
  Backspace: 0xb2,
  Tab: 0xb3,
  Space: 0x20,
  CapsLock: 0xc1,
  Insert: 0xd1,
  Home: 0xd2,
  PageUp: 0xd3,
  Delete: 0xd4,
  End: 0xd5,
  PageDown: 0xd6,
  ArrowRight: 0xd7,
  ArrowLeft: 0xd8,
  ArrowDown: 0xd9,
  ArrowUp: 0xda,
  ControlLeft: 0x80,
  ShiftLeft: 0x81,
  AltLeft: 0x82,
  MetaLeft: 0x83,
  ControlRight: 0x84,
  ShiftRight: 0x85,
  AltRight: 0x86,
  MetaRight: 0x87,
  Minus: 0x2d,
  Equal: 0x3d,
  BracketLeft: 0x5b,
  BracketRight: 0x5d,
  Backslash: 0x5c,
  Semicolon: 0x3b,
  Quote: 0x27,
  Comma: 0x2c,
  Period: 0x2e,
  Slash: 0x2f,
  Backquote: 0x60,
};

export function kc(code: string): number | null {
  if (code in SPECIAL) {
    return SPECIAL[code];
  }
  if (code.length === 4 && code.startsWith("Key")) {
    return code.charCodeAt(3) + 32;
  }
  if (code.length === 6 && code.startsWith("Digit")) {
    return code.charCodeAt(5);
  }
  if (code[0] === "F") {
    const n = parseInt(code.slice(1), 10);
    if (n >= 1 && n <= 12) {
      return 0xc1 + n;
    }
  }
  return null;
}

export function hidHex(n: number): string {
  return n.toString(16).toUpperCase().padStart(2, "0");
}

let sendQueue = Promise.resolve();

/**
 * Browser stays on localhost. Next rewrites /api/key → Go → Pi POST /key.
 * Direct fetch to 172.20.10.x is blocked by Chrome (private-network / CORS).
 */
export function queueKeyLines(
  lines: string[],
  onStatus?: (text: string) => void,
): void {
  if (lines.length === 0) return;
  const body = lines.join("\n") + "\n";
  sendQueue = sendQueue
    .then(async () => {
      const res = await fetch(apiUrl("/key"), {
        method: "POST",
        body,
      });
      if (res.ok) {
        onStatus?.("sent");
        return;
      }
      const err = (await res.json().catch(() => ({}))) as { error?: string };
      onStatus?.(err.error ?? `error ${res.status}`);
    })
    .catch((e: unknown) => {
      const msg = e instanceof Error ? e.message : "error";
      onStatus?.(
        msg === "Failed to fetch"
          ? "error: backend unreachable — restart go run . on :8080"
          : `error: ${msg}`,
      );
    });
}

export function queueKeyLine(
  line: string,
  onStatus?: (text: string) => void,
): void {
  queueKeyLines([line], onStatus);
}
