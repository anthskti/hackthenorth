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
    name: "AWS Server 2",
    role: "Embedded System",
    status: "ONLINE",
    live: true,
  },
  {
    id: "boot-lab",
    name: "AWS Server 3",
    role: "BIOS / bootloader bench",
    status: "STANDBY",
    live: false,
  },
  {
    id: "recovery-node",
    name: "AWS Server 4",
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
