"use client";

import { useState } from "react";
import { CheckCircle2, Clock3, ChevronDown, Apple, Terminal, LayoutGrid } from "lucide-react";
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
    <section id="backends" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          One policy, three sandboxing backends.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Warden picks the strongest backend your OS supports automatically.
          If none of them can be applied correctly, it refuses to run
          unsandboxed rather than falling back silently.
        </p>

        {/* Card grid */}
        <div className="mt-10 grid gap-4 md:grid-cols-3">
          {BACKENDS.map((b) => {
            const Icon = b.icon;
            const verified = b.status === "verified";
            return (
              <div
                key={b.platform}
                className="group rounded-[12px] border border-ink-700 bg-ink-900 p-5 transition-colors hover:border-ink-600"
              >
                <div className="flex items-center justify-between">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg border border-ink-600 bg-ink-800 text-muted transition-colors group-hover:text-paper">
                    <Icon size={16} />
                  </span>
                  <span
                    className={
                      "flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[0.6875rem] font-medium " +
                      (verified
                        ? "border-grant/30 bg-grant-subtle text-grant"
                        : "border-progress/30 bg-progress-subtle text-progress")
                    }
                  >
                    {verified ? <CheckCircle2 size={11} /> : <Clock3 size={11} />}
                    {verified ? "Verified" : "In progress"}
                  </span>
                </div>

                <h3 className="mt-4 text-[1.0625rem] font-medium text-paper">
                  {b.platform}
                </h3>
                <p className="mt-1.5 font-mono text-[0.8125rem] leading-[1.5] text-paper/85">
                  {b.mechanism}
                </p>
                <p className="mt-0.5 text-[0.8125rem] leading-[1.5] text-muted">
                  {b.detail}
                </p>

                <p className="mt-4 border-t border-ink-800 pt-3 text-[0.75rem] text-muted">
                  {b.statusLabel}
                </p>

                <div className="mt-4 overflow-hidden rounded-[8px] border border-ink-800">
                  <Diagram src={b.image} alt={b.imageAlt} />
                </div>
              </div>
            );
          })}
        </div>

        {/* Collapsible verification footnote */}
        <div className="mt-8 rounded-[12px] border border-ink-800 bg-ink-950/60">
          <button
            onClick={() => setShowFootnote((v) => !v)}
            className="flex w-full items-center justify-between px-5 py-3.5 text-left"
            aria-expanded={showFootnote}
          >
            <span className="text-[0.8125rem] text-muted">
              <span className="text-paper">Verification is machine-checked, not claimed.</span>{" "}
              What CI proves, per platform.
            </span>
            <ChevronDown
              size={15}
              className={
                "shrink-0 text-muted transition-transform duration-200 " +
                (showFootnote ? "rotate-180" : "")
              }
            />
          </button>
          {showFootnote && (
            <div className="border-t border-ink-800 px-5 pb-5 pt-4">
              <ul className="space-y-2.5 text-[0.8125rem] leading-[1.6] text-muted">
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/85">Linux —</span> 7 escape tests
                    pass on real hardware (Arch x86_64): grants readable,
                    unlisted paths invisible, writes outside grants denied,
                    env filtered, fail-closed without bwrap.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <Clock3 size={14} className="mt-0.5 shrink-0 text-progress" />
                  <span>
                    <span className="text-paper/85">macOS —</span> green in CI
                    (GitHub runners provide sandbox-exec); a real end-user
                    machine run of the proof harness is still pending.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/85">Windows —</span> escape
                    tests run on GitHub Windows runners; using WFP + ETW
                    locally requires an elevated (Administrator) shell.
                  </span>
                </li>
                <li className="flex gap-2.5">
                  <CheckCircle2 size={14} className="mt-0.5 shrink-0 text-grant" />
                  <span>
                    <span className="text-paper/85">CI hard-fails</span> if
                    escape tests skip or never prove the sandboxed target
                    actually started. Warden fails closed with a clear message
                    rather than running unaudited.
                  </span>
                </li>
              </ul>
              <p className="mt-4 text-[0.8125rem] text-muted">
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
