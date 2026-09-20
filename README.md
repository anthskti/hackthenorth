<div align="center">

<img src="docs/img/logo.png" alt="orcactl" width="160">

# orcactl

**An AI agent with eyes and hands on a machine that has no operating system.**

[Demo video](https://youtu.be/W0umiOq4xsg) · [Hack the North 2026](https://hackthenorth.com)

</div>

---

## Inspiration

During a major data-center outage, tens to hundreds of servers can end up bricked or unreachable by every remote tool that lives at the OS level. A bad firmware or kernel update is enough. Traditionally, troubleshooting that means technicians on site — often many of them.

AI coding agents hit the same ceiling. They can write code, run commands, read a stack trace — but only inside an operating system. The moment the failure is *below* the OS, a corrupted bootloader, a BIOS setting, a reinstall, the agent has nothing to act inside.

IP-KVM already solves the remote half of this: see and control a machine over the network when SSH is not an option. But a human is still doing the clicking. orcactl is the next step.

## What it does

orcactl connects to each machine through its **hardware**: reading directly from HDMI and emulating keyboard and mouse input as a USB peripheral.

That means it works on any machine **out of the box, with no software installed** — one that has never been configured, is mid-reinstall, or is sitting at a BIOS prompt with no OS at all. Switching between servers is just re-plugging.

On request, orcactl autonomously iterates to solve an issue, capturing the display and sending input until it is done. This **scales to diagnosing and interpreting several connected machines at once**.

<div align="center">
  <img src="docs/img/dashboard.png" alt="The orcactl dashboard: live screen, agent chat, action log" width="820">
  <br><em>Live target screen, agent reasoning, and the action log</em>
</div>

## How we built it

<div align="center">
  <img src="docs/img/setup.jpg" alt="Full bench: operator laptop, target ThinkPad, and the orcactl controller" width="820">
  <br><em>Operator laptop (left), target machine (centre), orcactl controller (right)</em>
</div>

```
                HDMI                         USB (keyboard + mouse)
  target  ───────────────►  capture card          ▲
     ▲                           │                │
     └───────────────────────────┼────────────────┘
                                 ▼                │
                    ┌──────────────────────────────────────┐
                    │  Raspberry Pi 5 · QNX                │
                    │  kvmd (C): frames ─► MJPEG over HTTP │
                    │            /key   ─► UART            │
                    └───────────────┬──────────────────────┘
                          GPIO UART │  (Pi TX → Leonardo RX + GND)
                    ┌───────────────▼──────────────────────┐
                    │  Arduino Leonardo · C++ firmware     │
                    │  HID-Project, boot protocol          │
                    └──────────────────────────────────────┘
                                 ▲
                    HTTP         │
  Go + Gin backend ──────────────┘         Next.js dashboard
  OpenAI vision agent                      MJPEG + WebSocket
```

orcactl uses a **Raspberry Pi 5** running QNX, connected over GPIO pins on UART to an **Arduino Leonardo** (Pi TX to Leonardo RX, plus ground). The QNX Pi acts as the middle layer between our agent system and the machine, handling both input and output.

Because the Pi 5 does not support acting as a USB peripheral, input commands are forwarded to the Leonardo, which maps and sends the keystrokes and mouse coordinates.

**Video.** We wrote a C daemon on QNX that converts HDMI frame data from a Guermok capture card sensor into images and streams them over HTTP to our webserver.

**Input.** Commands go over the Pi's GPIO UART to the Arduino, which runs our C++ firmware to map them to USB as keyboard and mouse. We used NicoHood's HID-Project library so the board also emulates a **boot-protocol** device — regular USB devices are generally not supported in BIOS and UEFI menus.

<div align="center">
  <img src="docs/img/controller.jpg" alt="Inside the controller: Arduino Leonardo above, Raspberry Pi 5 below" width="440">
  <br><em>Inside the controller — Leonardo on top, Pi 5 underneath, joined by three jumper wires</em>
</div>

**The web layer.** A Next.js client hosts the dashboard, while the backend runs on Go's Gin framework. We use the OpenAI API, which continuously receives screenshots and gathers context on what is on the display.

**The agent.** In autonomous mode, **a master agent drafts a plan and a vision model iterates over the steps**, confirming each one was executed correctly before moving on.

## Challenges we ran into

- **QNX quirks** — using libraries, and compiling and linking code.
- **The Pi 5 does not support device-mode USB**, so after a lot of investigation into supported drivers we pivoted to adding an Arduino input controller.
- **Networking** — configuring and troubleshooting connections between the devices, the webapp and our laptops.
- **Prompting the agent** to navigate BIOS menus correctly.

## Accomplishments that we're proud of

Making an embedded system, and being able to watch our hardware agent run remotely on all of our laptops.

## What we learned

The QNX development environment, AI agent design, and USB protocols.

## What's next for orcactl

Running on more devices concurrently :)

---

## Repository

| Path | What |
|---|---|
| `embedded/qnx/kvmd.c` | QNX daemon: capture card → MJPEG over HTTP, and `POST /key` → UART |
| `embedded/qnx/kvm_setup.sh` | Brings up the GPIO UART (`/dev/ser1`) and the capture card sensor |
| `embedded/kvm_keyboard/` | Arduino Leonardo firmware: serial commands → USB keyboard and mouse |
| `backend/` | Go + Gin: video relay, dashboard WebSocket, agent loop, OpenAI client |
| `frontend/` | Next.js dashboard: live screen, chat, action log, manual control |

## Hardware

- Raspberry Pi 5 running QNX
- Arduino Leonardo (any ATmega32U4 board with native USB)
- HDMI capture card (UVC)
- 3 jumper wires: **Pi pin 8 (GPIO14 TX) → Leonardo RX**, **Pi pin 6 (GND) → Leonardo GND**

> Do not wire the Leonardo's 5 V TX back to the Pi — Pi 5 GPIO is not 5 V tolerant.

The Leonardo's USB goes to the target machine; the capture card takes the target's HDMI.

## Running it

**1. Flash the Leonardo.** Install the **HID-Project** library (Library Manager → "HID-Project"), then upload `embedded/kvm_keyboard/kvm_keyboard.ino` with the board set to *Arduino Leonardo*.

**2. Start the Pi.** Over ssh:

```bash
sudo ./kvm_setup.sh          # UART + capture card
./kvmd -w 720 -h 480 -p 8080 -q 80 -s /dev/ser1
```

Check it: `http://<pi>:8080/stream` should show the target's screen, and `http://<pi>:8080/ui` is a built-in control page, handy for testing without the dashboard.

**3. Backend.**

```bash
cd backend && cp .env.example .env
```

Set `OPENAI_API_KEY` and point `QNX_BASE_URL` at the Pi, then:

```bash
cd backend && go run .
```

**4. Dashboard.**

```bash
cd frontend && bun install && bun dev
```

Open <http://localhost:3000>.

> In development the dashboard WebSocket connects straight to Go rather than through the Next rewrite, which drops the upgrade. Put `NEXT_PUBLIC_BACKEND_ORIGIN=http://localhost:8080` in `frontend/.env.local`.

## Serial protocol

`POST /key` takes one command per line. The QNX daemon validates each one before it reaches the Leonardo.

| Command | Meaning |
|---|---|
| `k<hex>` | tap a key — `kB0` Enter, `kC3` F2, `k61` `a` |
| `p<hex>` / `r<hex>` | press and hold / release |
| `t<text>` | type a string |
| `a` | release every key |
| `m<dx>,<dy>[,<wheel>]` | move the mouse, relative |
| `m<XXXX><YYYY>` | move the mouse, absolute (8 hex digits, each axis `0000`–`7FFF`) |
| `b<L\|R\|M><d\|u\|c>` | button down / up / click — `bLc` is a left click |
| `w<n>` | scroll wheel, signed |
| `z` | release every mouse button |

Key codes are USB HID bytes: `B0` Enter, `B1` Esc, `B2` Backspace, `B3` Tab, `20` Space, `C2`–`CD` F1–F12, `D7`/`D8`/`D9`/`DA` arrows, `80`–`83` left Ctrl/Shift/Alt/GUI. Printable characters use ASCII.

## Built with

QNX · Raspberry Pi 5 · Arduino · C · C++ · Go · Gin · Next.js · React · TypeScript · Tailwind · OpenAI API · HID-Project

Built at Hack the North 2026.
