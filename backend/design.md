# Backend Architecture — Autonomous IP-KVM

## 1. Overview

The backend is the "brain" of the system. It does not touch hardware directly —
that's the Pi's job. The backend:

- Receives video frames from the Raspberry Pi 5
- Calls the vision LLM (OpenAI) to decide the next action **(AI mode only)**
- Forwards manual keyboard/mouse input from the user straight to the Pi **(Manual mode only)**
- Sends action commands to the Pi for execution, regardless of source
- Hosts the dashboard webapp (serves frontend + its own API)
- Relays live video, logs, and status to the dashboard
- Accepts user prompts/goals, mode switches, and manual overrides from the dashboard

```
Target Server ── HDMI ──► Capture Dongle ── USB-A ──► Raspberry Pi 5
                                                            │
                                                    WS: frames up
                                                    WS: commands down
                                                            │
                                                            ▼
                                                    ┌───────────────┐
                                                    │  Go Backend    │
                                                    │                │
                                                    │  - Mode switch │
                                                    │  - Agent loop  │◄─ AI mode
                                                    │  - Manual relay│◄─ Manual mode
                                                    │  - OpenAI call │
                                                    │  - WS hub      │
                                                    │  - REST API    │
                                                    │  - Static      │
                                                    │    frontend    │
                                                    └───────────────┘
                                                            │
                                                    WS: log/status/video/manual input
                                                    REST: prompt/control/mode
                                                            │
                                                            ▼
                                                       Dashboard
                                                      (browser/frontend)
```

## 2. Language / stack

- **Go** — chosen for goroutine-based concurrency (frame handling, LLM calls,
  WS fan-out, and REST all running concurrently without callback complexity).
- `gorilla/websocket` (or `nhooyr.io/websocket`) for both WS links.
- Standard `net/http` for REST + static file serving of the built frontend.
- No database. No auth (single-user, local/hackathon-network deployment).
  Action log lives in memory; optionally flushed to a flat JSON file for
  crash recovery.

## 3. Components (internal)

| Component | Responsibility |
|---|---|
| `PiLink` | Owns the WS connection to the Pi. Receives frames, sends commands — regardless of whether they came from the agent or a manual override. |
| `ModeManager` | Tracks current mode (`ai` \| `manual`) and routes input to the right handler. Switching modes stops whichever loop is active. |
| `AgentLoop` | **AI mode only.** Core state machine: idle → thinking → acting → paused. Decides when to grab a frame, call the LLM, and dispatch a command. |
| `ManualRelay` | **Manual mode only.** Takes keyboard/mouse events straight from the dashboard and forwards them to the Pi as commands — no LLM involved. This is effectively "plain IP-KVM" reusing the same command channel. |
| `LLMClient` | Wraps the OpenAI vision API call. Used only by `AgentLoop`. |
| `Hub` | WebSocket fan-out to all connected dashboard clients (log entries, status/mode changes) and, in manual mode, the inbound channel for user input. |
| `VideoRelay` | Re-exposes the Pi's video feed to the dashboard via MJPEG (`/video/stream`) — identical in both modes, since video capture doesn't care who's driving. |
| `API` | REST handlers: prompt submission, pause/resume/stop, mode switch, status snapshot. |

**Key point:** both modes ultimately write to the same `PiLink` command
channel, using the same command message shape (Section 5). The only
difference is *who* produces the command — `AgentLoop` (AI mode) or
`ManualRelay` (manual mode, sourced directly from the user's raw input).
This means the Pi side needs zero changes to support manual mode — it just
executes whatever command arrives, same as always.

## 4. Modes

The system has one top-level **mode**, plus (in AI mode only) an **agent
status**. These are two separate concerns — don't conflate them in the UI or the API.

### 4.1 Mode: `ai` | `manual`

- **AI mode** — `AgentLoop` is active. User submits a goal/prompt; the loop
  captures frames, calls the LLM, and drives the Pi. User can intervene with
  the three override actions below.
- **Manual mode** — a "basic connection without AI." No LLM calls, no agent
  loop running at all. The user directly drives the target machine — this is
  just a regular IP-KVM, reusing the same video feed and command channel.
  Prompt box and pause/resume/stop are irrelevant here (not applicable, not shown).

Switching modes should cleanly tear down whichever loop was active:
switching `ai → manual` should stop the agent loop first (treat it like an
implicit `stop`); switching `manual → ai` should ensure no stray manual input
handlers are left listening.

### 4.2 Agent status (AI mode only): `idle` | `thinking` | `acting` | `paused`

```
        ┌────────┐   prompt received   ┌──────────┐
        │  idle  │ ────────────────────►│ thinking │
        └────────┘                      └────┬─────┘
             ▲                                │ LLM returns action
             │                                ▼
        ┌────┴─────┐   action executed   ┌────────┐
        │  paused  │◄─────stop───────────│ acting │
        └──────────┘                     └────────┘
```

- **idle** — waiting for a prompt/goal from the user.
- **thinking** — frame captured, sent to LLM, awaiting decision.
- **acting** — command dispatched to Pi, awaiting confirmation/next frame.
- **paused** — user hit pause/stop; loop halts until resumed.

The three manual-override actions on this status:
- **pause** — halt the loop after the current action completes; state → `paused`.
- **resume** — pick the loop back up from `paused`; state → `idle` (awaiting next tick) or back into the cycle.
- **stop** — hard-stop the loop entirely; clears any in-flight goal; state → `idle`.

Every mode/status transition emits a log entry + status update over the
dashboard WebSocket.

## 5. Pi ↔ Backend link

**`WS /agent/link`** — the Pi connects once at boot and holds this connection open.
Identical in both modes — the Pi has no concept of AI vs. manual, it just executes commands.

Pi → Backend (upstream):
```json
{ "type": "frame", "data": "<base64 jpeg>", "ts": 1234567890 }
```
```json
{ "type": "ack", "command_id": "abc123", "status": "done" }
```

Backend → Pi (downstream):
```json
{ "type": "command", "command_id": "abc123", "action": "key", "value": "ENTER" }
```
```json
{ "type": "command", "command_id": "abc124", "action": "mouse_click", "x": 340, "y": 220 }
```

- `command_id` lets the backend match acks back to the action it dispatched
  (useful for the action log — "pressed ENTER ✓ confirmed"). In manual mode
  you may choose to skip logging every single keystroke to avoid flooding
  the log — consider only logging connection/mode events, not raw input.
- Keep the frame cadence configurable in AI mode (e.g. every N seconds, or
  on-demand/triggered) — this is your main lever for LLM API cost and
  perceived responsiveness. In manual mode, video should just stream
  continuously at whatever frame rate the capture supports (no LLM in the
  loop to bottleneck on).

## 6. Backend ↔ Dashboard (frontend)

**`WS /ws/dashboard`** — now bidirectional (previously outbound-only):

- **Backend → Frontend:** log entries, status/mode changes (unchanged from before).
- **Frontend → Backend:** manual input events, only meaningful/accepted while `mode == "manual"`.

```json
// backend → frontend
{ "type": "log", "text": "Detected BIOS boot menu, pressing F2", "ts": 0 }
{ "type": "status", "mode": "ai", "state": "acting" }
{ "type": "status", "mode": "manual" }
```
```json
// frontend → backend (manual mode only)
{ "type": "manual_input", "action": "key", "value": "a" }
{ "type": "manual_input", "action": "mouse_move", "x": 120, "y": 340 }
{ "type": "manual_input", "action": "mouse_click", "button": "left" }
```

Backend should silently drop (or reject with a log line) any `manual_input`
message received while `mode == "ai"`, to guard against a stale client.

**`GET /video/stream`** — MJPEG multipart stream, relayed from the Pi's frames.
Identical endpoint in both modes.

**REST:**

| Method | Path | Body | Purpose |
|---|---|---|---|
| POST | `/api/prompt` | `{ "goal": "reinstall the OS" }` | AI mode only — starts the agent loop with a new goal |
| POST | `/api/control` | `{ "action": "pause" \| "resume" \| "stop" }` | AI mode only — manual override of the agent loop |
| POST | `/api/mode` | `{ "mode": "ai" \| "manual" }` | Switches top-level mode |
| GET | `/api/status` | — | Snapshot: current mode + (if AI) agent state + last N log entries |

## 7. Open questions / TODO

- [ ] Frame cadence: fixed interval vs. triggered by Pi detecting screen change? (AI mode only — manual mode should just stream continuously)
- [ ] What happens if the Pi WS connection drops mid-action? (reconnect + resume vs. hard stop) — applies in both modes
- [ ] Do we cap action log size in memory, or flush to disk periodically?
- [ ] Confirm OpenAI model choice (cost vs. latency vs. accuracy trade-off) — see cost notes from earlier discussion.
- [ ] Decide whether `/agent/link` needs any auth token even for hackathon demo (recommend: simple shared secret in a header, cheap insurance).
- [ ] Manual mode input rate: do we throttle mouse_move events client-side before sending, to avoid flooding the WS?
- [ ] Should switching to manual mode require confirmation if the agent is mid-`acting`?