export type SystemStatus = "ONLINE" | "STANDBY";

export type System = {
  id: string;
  name: string;
  role: string;
  status: SystemStatus;
  live: boolean;
};

export const SYSTEMS: System[] = [
  {
    id: "qnx-target",
    name: "QNX Raspberry Pi 5 Target",
    role: "Embedded System",
    status: "ONLINE",
    live: true,
  },
  {
    id: "boot-lab",
    name: "Boot Lab",
    role: "BIOS / bootloader bench",
    status: "STANDBY",
    live: false,
  },
  {
    id: "recovery-node",
    name: "Recovery Node",
    role: "OS update pipeline",
    status: "STANDBY",
    live: false,
  },
];

export const DEFAULT_SYSTEM_ID = "qnx-target";

export function getSystemById(id: string): System | undefined {
  return SYSTEMS.find((s) => s.id === id);
}

export function isLiveSystem(id: string): boolean {
  return getSystemById(id)?.live === true;
}
