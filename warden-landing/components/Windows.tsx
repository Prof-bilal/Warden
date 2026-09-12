import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import { WinAppContainer, WinWFP, WinJobObject, WinETW } from "@/lib/images";

const LAYERS: StepperItem[] = [
  {
    title: "AppContainer Token",
    summary: "LowBox token denies all.",
    body: "Every sandboxed process runs under a LowBox token that denies all filesystem, network, and environment access by default. The token is the hard boundary — no syscall can cross it.",
    image: <WinAppContainer />,
  },
  {
    title: "WFP Egress Filters",
    summary: "TCP + DNS blocked.",
    body: "Windows Filtering Platform rules permit only the loopback proxy bridge and block every other outbound connection. DNS is denied by the token; TCP is denied by the filters.",
    image: <WinWFP />,
  },
  {
    title: "Job Object Limits",
    summary: "Timeout + memory + kill.",
    body: "Wall-clock timeout, memory cap, and kill-on-close are enforced by the kernel. A runaway process is terminated with its entire tree — no orphaned children.",
    image: <WinJobObject />,
  },
  {
    title: "ETW Audit Trail",
    summary: "Kernel file I/O trace.",
    body: "A private real-time trace session captures kernel file I/O events scoped to the sandbox tree. Every file access is logged, not guessed — the audit is the product.",
    image: <WinETW />,
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
