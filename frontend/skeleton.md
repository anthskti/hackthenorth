# BOS Dashboard — Frontend Skeleton

UI contract for the prototype dashboard. Single route: `/`. Backend contract: [backend/design.md](../backend/design.md).

**Dev setup (prototype):** Next.js on `:3000`, Go on `:8080`. [next.config.ts](next.config.ts) rewrites `/api`, `/ws`, `/video`, and `/ping` to the backend (`BACKEND_URL` env, default `http://127.0.0.1:8080`). Use helpers in [src/lib/backend.ts](src/lib/backend.ts) for URLs. Do not embed static files in Go until demo packaging.

---

## Demo story (~60s)

1. Live target screen in the video pane.
2. AI mode: enter a goal → Run → watch `thinking` / `acting` and the action log.
3. Pause (or Stop) the agent.
4. Switch to Manual → click/type on the same video to drive the target (plain IP-KVM).

---

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ BOS    [ AI | Manual ]     Pi ●   WS ●     [ idle | thinking … ] │  ← header
├──────────────────────────────────────┬──────────────────────────┤
│                                      │  AI: goal + Run          │
│         VIDEO (16:9, letterbox)      │       Pause Resume Stop  │
│         LIVE badge overlay           │  Manual: hint text       │
│         (focusable in manual)        ├──────────────────────────┤
│                                      │  Action log              │
│                                      │  (auto-scroll, ~100 rows)│
├──────────────────────────────────────┴──────────────────────────┤
│ optional: last ack — e.g. key ENTER ✓                            │
└─────────────────────────────────────────────────────────────────┘
```

| Region | Contents |
|--------|----------|
| **Header** | Product name **BOS**, mode segmented control `AI \| Manual`, connection indicators (Pi link, dashboard WebSocket), agent state pill in AI mode only |
| **Main (left)** | MJPEG video pane; in manual mode, focusable and captures keyboard + mouse |
| **Right rail** | Mode-specific controls (see matrix below) + action log |
| **Footer (optional)** | Last command acknowledgment from agent activity |

**Video pane**

- Use a plain `<img src="/video/stream" />` (or full backend URL in dev). **Do not** use `next/image` — it cannot consume multipart MJPEG.
- Aspect: 16:9, letterboxed inside the pane.
- Overlay: `LIVE` or `Pi disconnected` badge.
- Manual mode: `tabIndex={0}`, show focus ring; optional crosshair on hover.

---

## Application state

Mirror the API. Do not add client-only modes or states.

```ts
type Mode = "ai" | "manual";

type AgentState = "idle" | "thinking" | "acting" | "paused";

type LogEntry = {
  text: string;
  ts: number;
};

type AppState = {
  /** Top-level mode from backend */
  mode: Mode;
  /** Agent loop state; null when mode === "manual" */
  state: AgentState | null;
  logs: LogEntry[];
  /** Dashboard WebSocket open */
  wsConnected: boolean;
  /** Pi reachable (from status or WS); drives video badge */
  piConnected: boolean;
  /** Controlled input for the goal field (local until Run) */
  goal: string;
};
```

**Hydration:** On mount and after each WebSocket reconnect, call `GET /api/status` once, then rely on `WS /ws/dashboard` for updates. No polling.

**WebSocket lifecycle:** Reconnect with exponential backoff. On `open`, always re-fetch `/api/status`.

---

## Control visibility matrix

| UI element | Visible / enabled when |
|------------|-------------------------|
| Mode toggle `AI \| Manual` | Always |
| Video pane | Always |
| Agent state pill (`idle`, `thinking`, …) | `mode === "ai"` |
| Action log | Always |
| Goal field + **Run** | `mode === "ai"` && `state === "idle"` |
| **Pause** | `mode === "ai"` && (`state === "thinking"` \|\| `state === "acting"`) |
| **Resume** | `mode === "ai"` && `state === "paused"` |
| **Stop** | `mode === "ai"` && (`thinking` \|\| `acting` \|\| `paused`) |
| Manual hint (“Click the video…”) | `mode === "manual"` |
| Prompt + Pause / Resume / Stop | Hidden in manual |
| Manual input listeners (key / mouse) | `mode === "manual"` && video pane **focused** |

**Semantics (match backend):**

- **Pause** → `paused` after current action completes.
- **Resume** → leaves `paused`, loop continues.
- **Stop** → clears goal, `state` → `idle` (not `paused`).
- Switching **AI → Manual** implicitly stops the agent on the server; clear any “running” UI and enable manual listeners on the video pane.
- No confirmation dialog when switching modes (prototype).

---

## Endpoint map

Base URL in dev: proxied paths on the Next origin, or `http://localhost:8080` if calling directly.

### Dashboard-facing (frontend uses these)

| Method | Path | When to call | Request | Response / effect |
|--------|------|--------------|---------|-------------------|
| **GET** | `/video/stream` | Always (img `src`) | — | MJPEG multipart stream |
| **WS** | `/ws/dashboard` | On mount | See messages below | Live log + status; manual input outbound |
| **GET** | `/api/status` | Mount + WS reconnect | — | Snapshot: `mode`, `state` (if AI), `logs[]`, Pi connectivity if exposed |
| **POST** | `/api/mode` | User toggles mode | `{ "mode": "ai" \| "manual" }` | Mode change; WS `status` broadcast |
| **POST** | `/api/prompt` | User clicks Run | `{ "goal": string }` | AI only; expect 409 in manual; then `thinking` via WS |
| **POST** | `/api/control` | Pause / Resume / Stop | `{ "action": "pause" \| "resume" \| "stop" }` | AI only; expect 409 in manual |

### Not used by the dashboard

| Method | Path | Notes |
|--------|------|--------|
| **WS** | `/agent/link` | Pi only: frames up, commands down |
| **GET** | `/ping` | Optional health check while wiring Go; no UI |

---

## `GET /api/status` (expected shape)

Align with [backend/design.md](../backend/design.md); extend only if backend adds fields.

```ts
type StatusResponse = {
  mode: Mode;
  state?: AgentState; // present when mode === "ai"
  logs: LogEntry[];
  pi_connected?: boolean;
};
```

If `mode === "manual"`, treat `state` as absent → `AppState.state = null`.

---

## `WS /ws/dashboard` messages

### Backend → frontend

```json
{ "type": "log", "text": "Detected BIOS boot menu, pressing F2", "ts": 0 }
```

```json
{ "type": "status", "mode": "ai", "state": "acting" }
```

```json
{ "type": "status", "mode": "manual" }
```

- Append `log` messages to `logs` (cap display ~100 rows; backend may ring-buffer ~200).
- On `status`, update `mode` and `state` (`null` when `mode === "manual"`).

### Frontend → backend (manual mode only)

Send only while `mode === "manual"` and video pane is focused. Backend drops these in AI mode; client should not send them anyway.

```json
{ "type": "manual_input", "action": "key", "value": "a" }
```

```json
{ "type": "manual_input", "action": "mouse_move", "x": 120, "y": 340 }
```

```json
{ "type": "manual_input", "action": "mouse_click", "button": "left" }
```

**Mouse move:** throttle client-side (~30ms or `requestAnimationFrame`) to avoid flooding the socket.

**Coordinates:** `mouse_move` / `mouse_click` should use coordinates **relative to the video element** (same space the Pi expects after any letterboxing — document scaling in implementation).

---

## REST error handling (prototype)

| Case | HTTP | UI behavior |
|------|------|-------------|
| Prompt/control in manual | 409 | Should not happen if matrix is respected |
| Empty goal on Run | 400 (if backend validates) | Disable Run when `goal.trim()` is empty |
| Backend down | network error | Show disconnected badges; log line optional |

---

## Suggested file structure (implementation)

```
frontend/src/
  app/
    page.tsx              # dashboard shell
    layout.tsx            # fonts, metadata → title "BOS"
  components/             # optional split
    Header.tsx
    VideoPane.tsx
    ControlRail.tsx
    ActionLog.tsx
  lib/
    types.ts              # Mode, AgentState, AppState, WS payloads
    api.ts                # fetch wrappers for REST
    useDashboardWs.ts     # WS + reconnect + status hydrate
```

Start with everything in `page.tsx` + fake state (`dashboard-shell` todo); extract when wiring live data.

---

## Build order (frontend)

1. **skeleton.md** (this file) — UI contract.
2. **dashboard-shell** — static layout + fake `AppState`.
3. **next-proxy** — rewrites to `:8080`.
4. **wire-live** — status, WS, MJPEG `<img>`.
5. **wire-controls** — mode, prompt, control, then manual input on video.

---

## Out of scope for skeleton UI

- Auth, multi-user, settings pages, routing beyond `/`.
- Logging every keystroke in manual mode in the action log (backend may omit; UI should not expect it).
- Serving the built Next app from Go (last-mile demo packaging).
