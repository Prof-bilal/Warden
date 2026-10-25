import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import Diagram from "@/components/Diagram";

const LAYERS: StepperItem[] = [
  {
    title: "AppContainer Token",
    summary: "Restricted process permissions.",
    body: "AppContainer restricts the process, while policy grants control selected filesystem access. Environment filtering is applied separately before launch.",
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
    body: "Job Objects group the sandboxed process tree for memory limits and cleanup. Warden supervises the wall-clock timeout and terminates the job when needed.",
    image: <Diagram src="/diagrams/job-object-tree-kill.jpeg" alt="Process tree termination inside a Windows Job Object" />,
  },
  {
    title: "ETW Audit Trail",
    summary: "Kernel file I/O trace.",
    body: "An elevated ETW session captures supported file I/O events for the sandbox tree. Denied file opens can be rejected before an ETW event is emitted, so not every denial appears in this trace.",
    image: <Diagram src="/diagrams/etw-audit-trail.jpeg" alt="ETW audit trail architecture for kernel file I/O tracing" />,
  },
];

export default function Windows() {
  return (
    <section id="windows" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <div className="flex items-baseline gap-3">
          <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2rem]">
            Windows: four layers, one boundary.
          </h2>
          <span className="rounded-sm bg-grant-subtle px-2.5 py-1 font-hero text-[0.75rem] text-grant">
            Native backend
          </span>
        </div>
        <p className="mt-3 max-w-[36rem] font-hero text-[1rem] leading-[1.65] text-muted">
          The Windows backend combines AppContainer, WFP, ETW, and Job Objects.
          Full network enforcement and ETW require elevation. The opt-in
          non-admin mode blocks network access and does not provide ETW file auditing.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Windows Layers" items={LAYERS} />
        </div>
      </div>
    </section>
  );
}
