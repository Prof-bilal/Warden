import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import Diagram from "@/components/Diagram";

const LAYERS: StepperItem[] = [
  {
    title: "AppContainer Token",
    summary: "LowBox token denies all.",
    body: "Every sandboxed process runs under a LowBox token that denies all filesystem, network, and environment access by default. The token is the hard boundary — no syscall can cross it.",
    image: <Diagram src="/diagrams/appcontainer-token.jpeg" alt="Windows AppContainer token security boundary" />,
  },
  {
    title: "WFP Egress Filters",
    summary: "TCP + DNS blocked.",
    body: "Windows Filtering Platform rules permit only the loopback proxy bridge and block every other outbound connection. DNS is denied by the token; TCP is denied by the filters.",
    image: <Diagram src="/diagrams/wfp-egress-filters.jpeg" alt="WFP egress filters network diagram" />,
  },
  {
    title: "Job Object Limits",
    summary: "Timeout + memory + kill.",
    body: "Wall-clock timeout, memory cap, and kill-on-close are enforced by the kernel. A runaway process is terminated with its entire tree — no orphaned children.",
    image: <Diagram src="/diagrams/job-object-tree-kill.jpeg" alt="Process tree termination inside a Windows Job Object" />,
  },
  {
    title: "ETW Audit Trail",
    summary: "Kernel file I/O trace.",
    body: "A private real-time trace session captures kernel file I/O events scoped to the sandbox tree. Every file access is logged, not guessed — the audit is the product.",
    image: <Diagram src="/diagrams/etw-audit-trail.jpeg" alt="ETW audit trail architecture for kernel file I/O tracing" />,
  },
];

export default function Windows() {
  return (
    <section id="windows" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex items-baseline gap-3">
          <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
            Windows: four layers, one boundary.
          </h2>
          <span className="rounded-sm bg-grant-subtle px-2.5 py-1 text-[0.75rem] text-grant">
            v0.1.6
          </span>
        </div>
        <p className="mt-3 max-w-[36rem] text-[1rem] leading-[1.65] text-muted">
          The Windows backend stacks four OS-native primitives so each
          sandboxed MCP server gets exactly the access its policy allows —
          no more, no fallback, no silent escalation.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Windows Layers" items={LAYERS} />
        </div>
      </div>
    </section>
  );
}
