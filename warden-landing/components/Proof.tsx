import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import {
  ShieldFailClosed,
  DenyFilesystem,
  NetworkEnforced,
  EnvFilter,
  PlatformFailClosed,
  ResourceLimits,
} from "@/lib/images";

const GUARANTEES: StepperItem[] = [
  {
    title: "Fail-closed by default",
    summary: "No valid backend = no run.",
    body: "Warden refuses to start without a valid sandbox backend. If enforcement can't be verified, the server never runs.",
    image: <ShieldFailClosed />,
  },
  {
    title: "Deny-by-default filesystem",
    summary: "Only granted paths visible.",
    body: "Only explicitly granted paths are visible. Everything else — including the rest of the filesystem, environment variables, and network — is invisible.",
    image: <DenyFilesystem />,
  },
  {
    title: "Network enforced at the kernel level",
    summary: "DNS blocked before resolution.",
    body: "An in-process egress proxy blocks every hostname not in the policy. DNS is resolved only after the allowlist check. No policy grant, no connection.",
    image: <NetworkEnforced />,
  },
  {
    title: "Environment filtering",
    summary: "Named variables only.",
    body: "Only env.allow names are forwarded. Empty allowlist = empty environment. No secrets leak through ungranted variables.",
    image: <EnvFilter />,
  },
  {
    title: "Fail-closed on every platform",
    summary: "Linux, macOS, Windows.",
    body: "Linux (bubblewrap), macOS (Seatbelt), Windows (AppContainer + WFP + ETW), Docker fallback — each requires its primitives to initialize or the run is refused.",
    image: <PlatformFailClosed />,
  },
  {
    title: "Resource limits enforced by the kernel",
    summary: "Timeout + memory + kill.",
    body: "Wall-clock timeout, memory RSS sampling with SIGTERM-then-SIGKILL, and kill-on-close via Job Objects on Windows. A runaway process is terminated with its entire tree.",
    image: <ResourceLimits />,
  },
];

export default function Proof() {
  return (
    <section id="proof" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          It blocks attacks by design, not by accident.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Warden enforces six hard invariants. Every one of them is verified by
          the test suite — including escape tests that confirm the sandbox
          actually prevents the access it claims to block.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Guarantees" items={GUARANTEES} />
        </div>

        {/* What's been verified */}
        <div className="mt-10 border-y border-ink-800 py-6">
          <p className="text-[0.8125rem] leading-[1.6] text-muted">
            <span className="text-grant">Verified on Linux (Arch x86_64):</span>{" "}
            7 escape tests pass — read grants accessible, unlisted paths invisible,
            write grants writable, writes outside grants denied, exit codes
            propagated, environment passthrough filtered, fail-closed without bwrap.
          </p>
          <p className="mt-3 text-[0.8125rem] leading-[1.6] text-muted">
            <span className="text-progress">Code-complete, verification pending:</span>{" "}
            macOS (Seatbelt) — unit tests pass; real-machine escape tests require
            macOS hardware.
          </p>
        </div>

        <p className="mt-6 text-[0.8125rem] leading-[1.6] text-muted/80">
          We publish only claims backed by committed test fixtures and source
          code. Attack-simulation benchmarks are not included until a
          reproducible harness and fixtures are committed to the repo. See{" "}
          <a
            className="underline decoration-muted/40 hover:text-paper"
            href="https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/ROADMAP.md"
          >
            ROADMAP.md
          </a>{" "}
          for what&apos;s still in progress.
        </p>
      </div>
    </section>
  );
}
