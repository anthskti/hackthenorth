# Lenovo ThinkPad T14s Gen 6 — UEFI BIOS (F1 Setup) Reference

**Machine:** Lenovo ThinkPad T14s Gen 6 (AMD variant)
**Model:** 21QJ00CNUS
**CPU:** AMD Ryzen AI 7 PRO 350 w/ Radeon 860M
**BIOS entry key:** `F1` (repeatedly, right after power-on, before the logo screen finishes)
**Boot device menu key:** `F12`
**BIOS at time of writing:** UEFI BIOS R2XET39W (1.19), dated 2026-03-19

> Source: Lenovo's official T14s Gen 6 User Guide (Chapter 5: "Configure advanced settings"), Lenovo BIOS Security white papers, AMD's AIM-T enablement documentation, and general ThinkPad UEFI BIOS conventions (menu contents can vary slightly by firmware revision — always verify live values in Setup before relying on this for automation).

This BIOS uses a modern GUI-style Setup (mouse + arrow-key navigable), organized into six top-level menu items in the left nav bar: **Main, Config, Date/Time, Security, Startup, Restart**.

---

## Top-level nav bar

| Menu | Purpose |
|---|---|
| **Main** | Read-only system identification/info screen. No configurable settings — just reference data. |
| **Config** | The largest section. Hardware- and platform-level configuration: networking, ports, input devices, display, power/thermal behavior, alerts, and AMD's remote-manageability stack. |
| **Date/Time** | Sets the system real-time clock (RTC) date and time used by the OS and BIOS event log. |
| **Security** | All password, biometric, TPM/firmware-security, and Secure Boot configuration. This is the tab with the most impact on OS installation/dual-boot. |
| **Startup** | Boot device priority and boot-mode (UEFI vs. Legacy/CSM) settings. |
| **Restart** | Exit menu — save changes, discard changes, or reset to factory/setup defaults. |

---

## Main

Purely informational — no editable fields, just system identity used for asset tracking, support calls, and driver/BIOS-update lookups.

- System Firmware Version / UEFI BIOS Version
- UEFI BIOS Date (Year-Month-Day)
- Embedded Controller (EC) Version
- Machine Type Model (e.g., `21QJ00CNUS`)
- System-unit serial number
- System board serial number
- Asset Tag
- CPU Type / CPU Speed
- Installed memory
- UUID
- MAC Address (Internal LAN)
- Preinstalled OS License
- UEFI Secure Boot (status indicator)
- BIOS Event Log (accessible from here on some firmware revisions)

**Agent-relevant note:** this is where to read the machine type/model, serial numbers, and current firmware version to verify identity or check for pending BIOS updates.

---

## Config

Per Lenovo's documentation: *"enables you to update configurations relating to system settings such as network, USB, keyboard, display, CPU, and power."*

Confirmed submenus (from this unit's Config screen) — each is its own page with an arrow (`→`) indicating a submenu:

| Submenu | Contents |
|---|---|
| **Network** | Wake on LAN, PXE/network boot, wireless radio enable/disable (WLAN/WWAN/Bluetooth), UEFI network stack options. |
| **USB** | USB port enable/disable, USB Always On (charging when powered off), USB 3.x vs. 2.0 mode, Always On USB Charging in Battery mode. |
| **Keyboard/Mouse** | TrackPoint/trackpad enable-disable, Fn/Ctrl key swap, F1–F12 as primary function keys, keyboard backlight behavior at boot. |
| **Display** | Boot display device, panel self-refresh, eDP/graphics output options, "Boot Display Device" selection for docks/external monitors. |
| **Power** | Intelligent Cooling Boost (on by default — controls fan/thermal aggressiveness), Cool and Quiet on Lap, sleep/wake behavior (Wake on AC attach, Wake on Lid Open), AC/battery power management, disable built-in battery (for CRU/hardware service). |
| **Beep and Alarm** | Password beep, keyboard beep, low-battery alarm audio cues. |
| **AIM-T** | **AMD Integrated Management Technology** — AMD's out-of-band remote manageability stack for AMD PRO platforms (the AMD equivalent of Intel AMT/vPro). Lets IT admins remotely monitor/manage/diagnose the system even when the OS is down. Present because this unit uses an AMD Ryzen PRO ("AI 7 PRO 350") chip. Enabling requires supported network hardware and is typically an enterprise/IT-managed feature, not consumer-relevant unless deploying fleet management.

**Agent-relevant note:** if a task involves docking station/external display issues, USB charging, fan noise, or battery life, the fix usually lives under **Config → Power**, **Config → Display**, or **Config → USB**.

---

## Date/Time

Single simple screen:
- **System Date** (Month/Day/Year)
- **System Time** (Hour/Minute/Second, 24-hr or 12-hr depending on locale)

Used by the RTC coin-cell-backed clock; drifts if the CMOS battery dies. No sub-tabs.

---

## Security

Per Lenovo: *"enables you to configure security settings related to such as password, fingerprint, and I/O accessibility."* This tab has the most consequential options for OS installs, encryption, and lockdown.

Typical submenus/items on this BIOS generation:

| Item | Contents |
|---|---|
| **Password** | Supervisor Password (locks BIOS Setup itself), Power-On Password, NVMe/Hard Disk Password (drive-level, stays with the physical drive), Lock UEFI BIOS Settings, Set Minimum Password Length, Password at Unattended Boot, Password at Restart. |
| **Fingerprint** | Enroll/manage fingerprint templates for the built-in reader (if equipped); reset fingerprint data. |
| **Security Chip (TPM)** | Enable/disable the discrete or firmware TPM (fTPM on AMD platforms), clear TPM, TPM state reporting — relevant for BitLocker. |
| **UEFI BIOS Update Option** | Controls whether flashing BIOS updates requires the Supervisor Password; can restrict/allow Flash BIOS updating from the OS. |
| **Memory Protection** | Data Execution Prevention (DEP/NX bit) toggle. |
| **Virtualization** | AMD-V (SVM Mode) enable/disable — must be **Enabled** for Hyper-V, WSL2, VirtualBox/VMware, and most container/VM workloads. |
| **I/O Port Access** | Per-device enable/disable: Ethernet LAN, Wireless LAN, Bluetooth, Camera, Microphone, Fingerprint Reader, Memory Card Slot, USB ports, Optical/other. Useful for hardening or for OSes lacking driver support for a given device. |
| **Internal Device Access** | Similar granular control over internally-attached devices (WWAN card, etc.), sometimes merged with I/O Port Access on newer firmware. |
| **Anti-Theft / Absolute Persistence Module** | Enable/disable the UEFI hooks used by Absolute (Computrace)-style anti-theft/asset-tracking services. |
| **Secure Boot** | Enable/disable UEFI Secure Boot, manage/reset PK/KEK/db/dbx keys, switch between Standard Mode and Custom Mode. **Must often be disabled (or set to allow self-signed keys) to dual-boot most Linux distributions**, unless using a shim that's Microsoft-signed. |
| **Device Guard / Memory Protection extras** | Windows-specific virtualization-based security readiness toggles (present on some SKUs). |
| **Intelligent Security / User Presence Sensing** | Human-presence sensor features (lock-on-leave, wake-on-approach) if the unit has the sensor hardware. |

**Agent-relevant note:** Secure Boot + Supervisor Password + I/O Port Access are the three places most "why won't my USB boot / why can't I install Linux / why is my webcam missing" support issues trace back to.

---

## Startup

Per Lenovo: *"enables you to manage settings relevant to booting up."*

Typical contents:
- **Boot** (Boot Priority Order) — drag/reorder list of bootable devices (NVMe SSD, USB HDD/FDD, PXE/Network, etc.)
- **Network Boot** — PXE boot enable/disable and Legacy vs. UEFI PXE agent selection
- **UEFI/Legacy Boot** — choose UEFI Only, Legacy Only, or Both (with sub-choice of which is tried first). Modern ThinkPads increasingly ship UEFI-only.
- **CSM (Compatibility Support Module)** — legacy BIOS emulation layer, only present/relevant if Legacy/Both boot mode is available.
- **Boot Mode** (Quick/Diagnostics) — controls whether POST runs full self-test diagnostics or a fast boot path.
- **Option Key Display** — shows/hides the "press F1/F12" prompt at boot.

**Agent-relevant note:** boot-order changes and UEFI vs. Legacy mode live here — this is the tab to check for "won't boot from USB installer" or dual-boot/OS-swap tasks.

---

## Restart

Exit menu — the final tab, no further sub-navigation beyond its own options list:

- **Exit Saving Changes** — commit changes and reboot (same as pressing `F10` from anywhere in Setup)
- **Exit Discarding Changes** — reboot without saving
- **Load Setup Defaults** — reset all settings to factory defaults (same as `F9`)
- **Discard Changes** — revert unsaved edits without leaving Setup
- **Save Changes** — commit without exiting

---

## Quick key reference (applies throughout Setup)

| Key | Action |
|---|---|
| `F1` | General Help |
| `F9` | Load Setup Defaults |
| `Esc` | Back / up one level |
| `F10` | Save and Exit |
| Arrow keys | Navigate fields/menus |
| `+ / -` | Change a highlighted value |
| `Enter` | Open submenu / confirm selection |

---

## Notes for automated/agent use

1. **Menu contents vary by firmware revision.** Lenovo explicitly states the UEFI BIOS menu "might vary depending on system configurations" — always confirm live option names against the actual screen before scripting an action against them.
2. This is an **AMD** T14s Gen 6 (Ryzen AI 7 PRO 350). Intel-only items seen on other ThinkPad docs (Intel AMT, Intel SGX, Intel VT-d naming) do **not** apply here — the AMD equivalents are **AIM-T** (Config tab) and **AMD-V/SVM Mode** (Security tab, under Virtualization).
3. For programmatic BIOS configuration at scale, Lenovo supports config via **WMI** (Windows) or the **Think-LMI** driver (Linux, exposes settings under `/sys/class/firmware-attributes/`) instead of manual F1 navigation — relevant if the "agent" in question needs to read/write settings without a human at the keyboard.
4. Full authoritative per-model detail (every field, every valid value) is published per-model on Lenovo's **BIOS Simulator Center**: https://download.lenovo.com/bsco/index.html — search "T14s Gen 6" there for a live, clickable simulation of this exact BIOS if any field above needs verification.
