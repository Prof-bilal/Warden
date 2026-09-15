"use client";

import { useState } from "react";
import {
  Apple,
  CheckCircle2,
  ChevronDown,
  Clock3,
  LayoutGrid,
  ShieldX,
  Terminal,
} from "lucide-react";
import Diagram from "@/components/Diagram";

const BACKENDS = [
  {
    platform: "Linux",
    icon: Terminal,
    mechanism: "bubblewrap",
    detail: "unprivileged user namespaces",
    status: "verified" as const,
    statusLabel: "Verified on real hardware",
    image: "/diagrams/linux-backend.jpeg",
    imageAlt: "Linux backend architecture with bubblewrap sandboxing",
  },
  {
    platform: "macOS",
    icon: Apple,
    mechanism: "sandbox-exec",
    detail: "with Docker fallback",
    status: "pending" as const,
    statusLabel: "CI-verified · hardware run pending",
    image: "/diagrams/macos-backend.jpeg",
    imageAlt: "macOS security architecture with Seatbelt sandboxing",
  },
  {
    platform: "Windows",
    icon: LayoutGrid,
    mechanism: "AppContainer + WFP",
    detail: "Job Objects · ETW audit",
    status: "verified" as const,
    statusLabel: "CI-verified escape tests",
    image: "/diagrams/windows-backend.jpeg",
    imageAlt: "Windows backend architecture with AppContainer, WFP and ETW",
  },
];

export default function Backends() {
  const [showFootnote, setShowFootnote] = useState(false);

  return (
    <section id="backends" className="border-t border-ink-800 bg-ink-950">
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        {/* header */}
        <div className="mx-auto max-w-[640px] text-center">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Sandbox backends
          </p>
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2.25rem]">
            One policy, three sandboxing backends.
          </h2>
          <p className="mt-4 font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            Warden picks the strongest backend your OS supports automaticallythe same policy
            file works everywhere.
          </p>
        </div>

        {/* card grid */}
        <div className="mx-auto mt-10 grid max-w-5xl gap-4 md:grid-cols-3">
          {BACKENDS.map((b) => {
            const Icon = b.icon;
            const verified = b.status === "verified";
            return (
              <article
                key={b.platform}
                className="group flex flex-col overflow-hidden rounded-2xl border border-ink-700 bg-ink-900 transition-colors hover:border-ink-500"
              >
                <div className="flex items-center justify-between p-5 pb-0">
                  <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-ink-600 bg-ink-800 text-muted transition-colors group-hover:text-paper">
                    <Icon size={17} />
                  </span>
                  <span
                    className={
                      "flex items-center gap-1.5 rounded-full border px-2.5 py-1 font-hero text-[11px] font-bold " +
                      (verified
                        ? "border-grant/30 bg-grant-subtle text-grant"
                        : "border-progress/30 bg-progress-subtle text-progress")
                    }
                  >
                    {verified ? <CheckCircle2 size={11} /> : <Clock3 size={11} />}
                    {verified ? "Verified" : "In progress"}
                  </span>
                </div>

                <div className="px-5 pt-4">
                  <h3 className="font-hero text-[17px] font-bold text-paper">{b.platform}</h3>
                  <p className="mt-1.5 inline-block rounded-md border border-ink-700 bg-ink-950 px-2 py-0.5 font-hero text-[12.5px] text-paper/90">
                    {b.mechanism}
                  </p>
                  <p className="mt-1.5 font-hero text-[12.5px] leading-[1.55] text-muted">
                    {b.detail}
                  </p>
                </div>

                <div className="mx-5 mt-4 overflow-hidden rounded-xl border border-ink-800">
                  <div className="transition-transform duration-300 group-hover:scale-[1.02]">
                    <Diagram src={b.image} alt={b.imageAlt} />
                  </div>
                </div>

                <p className="mt-auto flex items-center gap-1.5 px-5 py-4 font-hero text-[11.5px] text-muted">
                  <span
                    className={
                      "h-1.5 w-1.5 rounded-full " + (verified ? "bg-grant" : "bg-progress")
                    }
                  />
                  {b.statusLabel}
                </p>
              </article>
            );
          })}
        </div>

        {/* fail-closed callout */}
        <div className="mx-auto mt-4 flex max-w-5xl items-start gap-3 rounded-2xl border border-deny/30 bg-deny-subtle px-5 py-4">
          <ShieldX size={17} className="mt-0.5 shrink-0 text-deny" />
          <p className="font-hero text-[13px] leading-[1.65] text-paper/90 md:text-[14px]">
            Fail-closed by default.{" "}
            <span className="text-muted">
              If no backend can be applied correctly, Warden refuses to run unsandboxed rather
              than falling back silently.
            </span>
          </p>
        </div>

        {/* verification footnote */}
        <div className="mx-auto mt-4 max-w-5xl overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
          <button
            onClick={() => setShowFootnote((v) => !v)}
            className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left"
            aria-expanded={showFootnote}
          >
            <span className="font-hero text-[13px] text-muted">
              <span className="font-bold text-paper">Verification is machine-checked, not claimed.</span>{" "}
              What CI proves, per platform.
            </span>
            <ChevronDown
              size={16}
              className={
                "shrink-0 text-muted transition-transform duration-200 " +
                (showFootnote ? "rotate-180" : "")
              }
            />
          </button>
          {showFootnote && (
            <div className="border-t border-ink-700 px-5 pb-5 pt-4">
              <ul className="space-y-2.5 font-hero text-[12.5px] leading-[1.65] text-muted">
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/90">Linux —</span> 7 escape tests pass on real
                    hardware (Arch x86_64): grants readable, unlisted paths invisible, writes
                    outside grants denied, env filtered, fail-closed without bwrap.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <Clock3 size={14} className="mt-0.5 shrink-0 text-progress" />
                  <span>
                    <span className="text-paper/90">macOS —</span> green in CI (GitHub runners
                    provide sandbox-exec); a real end-user machine run of the proof harness is
                    still pending.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/90">Windows —</span> escape tests run on GitHub
                    Windows runners; using WFP + ETW locally requires an elevated
                    (Administrator) shell.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/90">CI hard-fails</span> if escape tests skip or
                    never prove the sandboxed target actually started. Warden fails closed with a
                    clear message rather than running unaudited.
                  </span>
                </li>
              </ul>
              <p className="mt-4 font-hero text-[12.5px] text-muted">
                See the{" "}
                <a className="underline decoration-muted/40 hover:text-paper" href="/docs/install">
                  install docs
                </a>{" "}
                and the{" "}
                <a
                  className="underline decoration-muted/40 hover:text-paper"
                  href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/REMAINING_WORK.md"
                >
                  REMAINING_WORK
                </a>{" "}
                tracker for the exact CI verification state.
              </p>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
