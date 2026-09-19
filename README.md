# orcactl (Behind the OS)

AI coding agents today are remarkably capable, but only inside an OS. They can write code, browser the internet, run terminal commands, even debug a stack trace. But the moment something breaks below the OS (a corrupted bootloader, a BIOS setting that needs changing, an OS reinstall) the agent is helpless, because there's no operating system for it to act inside. That's still a human walking to a data center with a monitor and keyboard.

IP-KVM (IP-based keyboard/video/mouse) already solves remote access to this problem, it's the standard tool sysadmins use to see and control a server over the network when SSH isn't an option. But it's still a human doing the clicking. We want to build the next step: an autonomous IP-KVM, where a vision-capable AI agent can watch the and any issues behind the OS.

## Setup
Install dependencies
```
# Install (mac)

# Install (windows)

```

# Run
frontend:
```bash
cd frontend && bun dev
```

backend:
```bash
cd backend && go run .
```
