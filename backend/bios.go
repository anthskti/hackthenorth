package main

import _ "embed"

//go:embed thinkpadbios.md
var t14sGen6BIOSManual string

// Live KVM overlay: this unit's left-nav label and how focus looks on HDMI.
// Lenovo's guide names the boot tab "Startup"; the captured BIOS shows "Setup".
const t14sGen6LiveKVM = `LIVE UNIT (IP-KVM HDMI, already in F1 Setup — not Windows):
Left-nav order top→bottom: Main, Config, Date/Time, Security, Setup, Restart
  Setup == Lenovo "Startup" (boot devices). Do not confuse with Config.
Start: blue fill on Main. From Main: DOWN×1 = Config; DOWN×3 = Security; DOWN×4 = Setup.

FOCUS
- OPEN TAB = filled blue background on the left-nav item. That is the current page. Trust this, not a thin outline.
- HOVER on left nav = thin 1px border only. Arrows move this hover; the page / blue fill does NOT change until ENTER.
- HID LAYER: every UP/DOWN while pane=left_nav is followed by ENTER automatically so the next snapshot shows the new blue-filled tab.
- Right content = often a dotted box. UP/DOWN there do NOT auto-ENTER.
- LEFT/RIGHT jump left-nav ↔ right content.
- ESC = back one level.
- Boot list: F6 raises priority, F5 lowers. +/- also change some values.
- F10 = save and exit (confirm Yes if asked). F9 = load defaults — never press unless asked.

PRIMARY DEMO — switch boot drive to Windows ("windows driver" / Windows Boot Manager):
  1. Blue fill on Main (if on the right pane, LEFT first).
  2. LEFT-NAV DOWN four times (each auto-ENTER) until blue fill is Setup — skip Config, Date/Time, Security.
  3. Setup/Startup page: Boot, Network Boot, UEFI/Legacy, etc.
  4. RIGHT pane: DOWN/UP until Boot (Boot Priority Order) is boxed → ENTER (this ENTER is explicit).
  5. Highlight Windows Boot Manager (or the Windows NVMe / "Windows" disk).
  6. F6 until that entry is first in the list.
  7. F10 → Yes if prompted. Done only after save/exit is sent (or Yes confirmed).
Never open Config for a boot-order / Windows-driver goal.

SECURE BOOT INNER PAGE (critical — ENTER here EXITS/resets if you skip the Downs):
Once you have already ENTERed the Secure Boot submenu (right pane, inside Security → Secure Boot):
  - Do NOT press ENTER on the first/highlighted item. That ENTER backs out and undoes progress.
  - pane=right_content. Send DOWN exactly twice. Then ENTER. Nothing else in between.
  - Sequence: DOWN, DOWN, ENTER. That is the only way to change Secure Boot on this unit.
`

func t14sPlannerContext() string {
	return t14sGen6LiveKVM + "\n\n--- PAGE ENCYCLOPEDIA (use this to diagnose the operator goal) ---\n\n" + t14sGen6BIOSManual
}

func t14sExecutorContext() string {
	return t14sGen6LiveKVM + `

Executor still uses the encyclopedia via the plan's issue + menu_path.
Read on-screen labels; if a name differs (Setup vs Startup), follow the JPEG.

You execute a ThinkPad T14s Gen 6 BIOS plan over IP-KVM. Keyboard only. One HID step per turn.
Follow the agreed diagnosis and HID steps in order.

LEFT NAV (pane=left_nav):
- Open tab = FILLED BLUE BACKGROUND. That is the page you are on.
- Hover = thin 1px border only, easy to miss on JPEG. Ignore it as current tab.
- After you send UP or DOWN on the navbar, the backend will automatically press ENTER so the next JPEG shows the new blue-filled tab. Do not also send ENTER for that move.
- If blue fill is already the goal tab, action done.

RIGHT PANE (pane=right_content): UP/DOWN only, no auto-ENTER. ENTER opens a submenu (Boot, Password, etc).

If blue fill is still Main and the goal is Security, press DOWN (do not ENTER Main).
Boot-swap demo: do not mark done until Windows is first in Boot and F10 (or Yes) has been issued.
Once you enter the Secure Boot submenu: go DOWN exactly twice then ENTER. Never ENTER immediately — that exits.

observation: which left-nav item has the FILLED BLUE background, plus any thin hover border if visible.
pane: left_nav or right_content.
Actions: type | key (ENTER, ESC, F5, F6, F9, F10, UP, DOWN, LEFT, RIGHT, TAB, SPACE, or one letter) | wait | done.
`
}
