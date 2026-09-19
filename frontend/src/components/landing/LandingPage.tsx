import Link from "next/link";
import { DEFAULT_SYSTEM_ID, SYSTEMS } from "@/lib/systems";

const ISSUE_DATE = "SEP 2026";

export function LandingPage() {
  return (
    <div className="min-h-full bg-paper text-ink">
      <header className="border-b-4 border-ink px-6 py-4 md:px-10">
        <div className="mx-auto flex max-w-6xl flex-wrap items-end justify-between gap-4">
          <div>
            <p className="font-mono text-[10px] uppercase tracking-[0.35em]">
              Behind the OS
            </p>
            <h1 className="font-display text-4xl tracking-tight md:text-5xl">
              orcactl
            </h1>
          </div>
          <div className="text-right font-mono text-[10px] uppercase tracking-widest">
            <p>VOL. 01</p>
            <p>{ISSUE_DATE}</p>
          </div>
        </div>
        <p className="mx-auto mt-4 max-w-6xl border-t border-ink pt-3 font-mono text-[11px] uppercase tracking-wide">
          Autonomous IP-KVM. When SSH is not an option.
        </p>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-12 md:px-10 md:py-16">
        <section className="grid gap-10 border-b border-ink pb-16 md:grid-cols-12 md:gap-8">
          <div className="md:col-span-8">
            <h2 className="font-display text-4xl leading-[1.1] md:text-6xl lg:text-7xl">
              The agent that works when the OS doesn&apos;t.
            </h2>
          </div>
          <div className="flex flex-col justify-between gap-6 md:col-span-4">
            <p className="text-sm leading-relaxed md:text-base">
              AI agents excel inside an operating system. Below the
              OS (bootloader, BIOS, bare metal), they stop. orcactl bridges HDMI to an
              agent that can see and drive the machine remotely.
            </p>
            <div className="border border-ink p-4 font-mono text-[10px] uppercase tracking-widest">
              <p>Project:</p>
              <p className="mt-2 text-lg normal-case tracking-normal">
                Hack the North
              </p>
            </div>
          </div>
        </section>

        <section className="border-b border-ink py-16">
          <h3 className="mb-8 font-mono text-[11px] uppercase tracking-[0.3em]">
            Systems index
          </h3>
          <ul className="divide-y divide-ink border-t border-ink">
            {SYSTEMS.map((sys, i) => (
              <li key={sys.id}>
                <Link
                  href={`/${sys.id}`}
                  className="group grid grid-cols-[auto_1fr_auto] items-center gap-4 px-2 py-5 transition-colors hover:bg-ink hover:text-paper md:grid-cols-[3rem_1fr_8rem_6rem_auto]"
                >
                  <span className="font-mono text-sm tabular-nums">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <div>
                    <p className="font-medium">{sys.name}</p>
                    <p className="text-sm opacity-70">{sys.role}</p>
                  </div>
                  <span className="hidden font-mono text-[10px] uppercase tracking-widest md:block">
                    {sys.status}
                  </span>
                  <span className="hidden font-mono text-[10px] md:block">
                    {sys.live ? "UPLINK" : "—"}
                  </span>
                  <span className="font-mono text-lg" aria-hidden>
                    →
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        </section>

        <section className="grid gap-8 border-b border-ink py-16 md:grid-cols-3">
          <div className="border border-ink p-6">
            <p className="font-mono text-[10px] uppercase tracking-widest">
              01
            </p>
            <h4 className="mt-4 font-display text-2xl">HDMI</h4>
            <p className="mt-3 text-sm leading-relaxed">
              Frames from the target machine, relayed over the network like
              classic IP-KVM.
            </p>
          </div>
          <div className="border border-ink p-6">
            <p className="font-mono text-[10px] uppercase tracking-widest">
              02
            </p>
            <h4 className="mt-4 font-display text-2xl">Agent</h4>
            <p className="mt-3 text-sm leading-relaxed">
              A vision model interprets the screen and issues keyboard and mouse
              commands.
            </p>
          </div>
          <div className="border border-ink p-6">
            <p className="font-mono text-[10px] uppercase tracking-widest">
              03
            </p>
            <h4 className="mt-4 font-display text-2xl">KVM</h4>
            <p className="mt-3 text-sm leading-relaxed">
              Same pipe in manual mode—you drive when the agent should step
              aside.
            </p>
          </div>
        </section>

        <footer className="flex flex-wrap items-center justify-between gap-4 py-12 font-mono text-[10px] uppercase tracking-widest">
          <p>Hack the North 2026</p>
          <Link
            href={`/${DEFAULT_SYSTEM_ID}`}
            className="border border-ink bg-ink px-6 py-3 text-paper shadow-[4px_4px_0_0_#0a0a0a] transition-transform hover:translate-x-[2px] hover:translate-y-[2px] hover:shadow-[2px_2px_0_0_#0a0a0a]"
          >
            Enter workspace →
          </Link>
        </footer>
      </main>
    </div>
  );
}
