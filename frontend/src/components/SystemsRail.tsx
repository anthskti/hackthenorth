"use client";

import Link from "next/link";
import { SYSTEMS } from "@/lib/systems";

type SystemsRailProps = {
  activeId: string;
};

export function SystemsRail({ activeId }: SystemsRailProps) {
  return (
    <nav
      className="flex h-full min-h-0 w-[220px] shrink-0 flex-col border-r border-[var(--rule)]"
      aria-label="Systems"
    >
      <p className="border-b border-[var(--rule)] px-3 py-2 font-mono text-[10px] uppercase tracking-[0.25em]">
        Systems
      </p>
      <ul className="min-h-0 flex-1 overflow-y-auto">
        {SYSTEMS.map((sys) => {
          const active = sys.id === activeId;
          return (
            <li key={sys.id}>
              <Link
                href={`/${sys.id}`}
                className={`flex items-start gap-2 border-b border-[var(--rule)] px-3 py-3 text-sm transition-colors ${
                  active
                    ? "bg-[var(--foreground)] text-[var(--background)]"
                    : "hover:bg-[var(--foreground)]/10"
                }`}
              >
                <span
                  className={`mt-1.5 h-1.5 w-1.5 shrink-0 border border-[var(--rule)] ${
                    sys.status === "ONLINE"
                      ? "bg-[var(--foreground)]"
                      : "bg-transparent"
                  }`}
                  aria-hidden
                />
                <div className="min-w-0">
                  <p className="truncate font-medium">{sys.name}</p>
                  <p
                    className={`truncate font-mono text-[10px] uppercase tracking-wide ${
                      active ? "opacity-80" : "opacity-60"
                    }`}
                  >
                    {sys.status}
                  </p>
                </div>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
