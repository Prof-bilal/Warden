"use client";

import { useState } from "react";
import { Check, Clock3, Copy, FlaskConical, ShieldCheck } from "lucide-react";
import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import Diagram from "@/components/Diagram";

const GUARANTEES: StepperItem[] = [
  {
    title: "Fail-closed by default",
    summary: "No valid backend = no run.",
    body: "Warden refuses to start without a valid sandbox backend. If enforcement can't be verified, the server never runs.",
    image: <Diagram src="/diagrams/fail-closed-startup.jpeg" alt="Startup decision flow: no valid backend means the process refuses to run" />,
  },
  {
    title: "Deny-by-default filesystem",
    summary: "Only granted paths visible.",
    body: "Only explicitly granted paths are visible. Everything elseincluding the rest of the filesystem, environment variables, and networkis invisible.",
    image: <Diagram src="/diagrams/deny-by-default-filesystem.jpeg" alt="Deny-by-default filesystem architecture: only granted paths are visible to the sandbox" />,
  },
  {
    title: "Network enforced at the kernel level",
    summary: "DNS blocked before resolution.",
    body: "An in-process egress proxy blocks every hostname not in the policy. DNS is resolved only after the allowlist check. No policy grant, no connection.",
    image: <Diagram src="/diagrams/kernel-network-policy.jpeg" alt="Kernel-level network security policy blocking DNS before resolution" />,
  },
  {
    title: "Environment filtering",
    summary: "Named variables only.",
    body: "Only env.allow names are forwarded. Empty allowlist = empty environment. No secrets leak through ungranted variables.",
    image: <Diagram src="/diagrams/env-filter-boundary.jpeg" alt="Environment variables filtered at the sandbox boundary" />,
  },
  {
    title: "Fail-closed on every platform",
    summary: "Linux, macOS, Windows.",
    body: "Linux (bubblewrap), macOS (Seatbelt), Windows (AppContainer + WFP + ETW), Docker fallbackeach requires its primitives to initialize or the run is refused.",
    image: <Diagram src="/diagrams/cross-platform.jpeg" alt="Cross-platform security architecture covering Linux, macOS and Windows" />,
  },
  {
    title: "Resource limits enforced by the kernel",
    summary: "Timeout + memory + kill.",
    body: "Wall-clock timeout, memory RSS sampling with SIGTERM-then-SIGKILL, and kill-on-close via Job Objects on Windows. A runaway process is terminated with its entire tree.",
    image: <Diagram src="/diagrams/resource-limit-terminated.jpeg" alt="Process terminated at resource limit" />,
  },
];

const ESCAPE_TESTS = [
  "read grants accessible",
  "unlisted paths invisible",
  "write grants writable",
  "writes outside grants denied",
  "exit codes propagated",
  "environment passthrough filtered",
  "fail-closed without bwrap",
];

const REPRO_CMD = "bash warden-starter/warden/testdata/attacks/run-attacks.sh";

export default function Proof() {
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    navigator.clipboard.writeText(REPRO_CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <section id="proof" className="border-t border-ink-800 bg-ink-950">
      <div className="mx-auto max-w-content px-6 py-20 md:py-28">
        {/* header */}
        <div className="mx-auto max-w-[640px] text-center">
          <p className="mb-4 font-hero text-[12px] font-bold uppercase tracking-[0.12em] text-grant">
            Guarantees
          </p>
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2.25rem]">
            It blocks attacks by design, not by accident.
          </h2>
          <p className="mt-4 font-hero text-[15px] leading-[1.7] text-muted md:text-[16px]">
            Six hard invariants, each verified by escape tests that confirm the sandbox actually
            prevents the access it claims to block.
          </p>
        </div>

        <div className="mx-auto mt-10 max-w-5xl">
          <StepperDetail sectionLabel="Guarantees" items={GUARANTEES} />
        </div>

        {/* escape-test results */}
        <div className="mx-auto mt-10 max-w-5xl overflow-hidden rounded-2xl border border-ink-700 bg-ink-900">
          <div className="flex flex-wrap items-center gap-3 border-b border-ink-700 px-5 py-4">
            <span className="flex items-center gap-2 font-hero text-[13px] font-bold text-paper">
              <FlaskConical size={15} className="text-grant" />
              Escape tests
            </span>
            <span className="rounded-full border border-grant/30 bg-grant-subtle px-2.5 py-0.5 font-hero text-[11px] font-bold text-grant">
              7/7 contained
            </span>
            <span className="font-hero text-[11.5px] text-muted">
              Linux · Arch x86_64 · real hardware
            </span>
          </div>
          <ul className="grid gap-px bg-ink-800 sm:grid-cols-2">
            {ESCAPE_TESTS.map((t) => (
              <li
                key={t}
                className="flex items-center gap-2.5 bg-ink-900 px-5 py-2.5 font-hero text-[12.5px] text-paper/85"
              >
                <span className="flex items-center gap-1 rounded bg-grant px-1.5 py-0.5 text-[10px] font-bold tracking-[0.06em] text-ink-950">
                  <Check size={10} strokeWidth={3} />
                  PASS
                </span>
                {t}
              </li>
            ))}
            <li className="flex items-center gap-2.5 bg-ink-900 px-5 py-2.5 font-hero text-[12.5px] text-muted sm:col-span-2">
              <span className="flex items-center gap-1 rounded border border-progress/40 bg-progress-subtle px-1.5 py-0.5 text-[10px] font-bold tracking-[0.06em] text-progress">
                <Clock3 size={10} strokeWidth={3} />
                PENDING
              </span>
              macOS (Seatbelt)unit tests pass; real-machine escape tests require macOS
              hardware
            </li>
          </ul>
          {/* repro */}
          <div className="flex flex-col gap-3 border-t border-ink-700 px-5 py-4 sm:flex-row sm:items-center">
            <p className="flex items-center gap-2 font-hero text-[12px] text-muted">
              <ShieldCheck size={14} className="shrink-0 text-grant" />
              Reproduce itcontrol must land unsandboxed, be contained sandboxed:
            </p>
            <button
              onClick={handleCopy}
              className="group flex min-w-0 items-center justify-between gap-3 overflow-hidden rounded-full border border-ink-600 bg-ink-950 px-4 py-2 text-left font-hero text-[12px] text-paper transition-colors hover:border-grant/50 sm:ml-auto sm:max-w-md sm:flex-1"
              aria-label="Copy attack reproduction command"
            >
              <span className="truncate">
                <span className="text-muted">$ </span>
                {REPRO_CMD}
              </span>
              {copied ? (
                <Check size={14} className="shrink-0 text-grant" />
              ) : (
                <Copy size={14} className="shrink-0 text-muted transition-colors group-hover:text-paper" />
              )}
            </button>
          </div>
        </div>

        <p className="mx-auto mt-6 max-w-5xl font-hero text-[12px] leading-[1.7] text-muted/80">
          We publish only claims backed by committed test fixtures and source code. Every run
          confirms containment host-sidecollector logs, vault hashes, escape probesand
          writes evidence files. See the{" "}
          <a
            className="underline decoration-muted/40 hover:text-paper"
            href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/testdata/attacks/run-attacks.sh"
          >
            attack harness
          </a>
          .
        </p>
      </div>
    </section>
  );
}
