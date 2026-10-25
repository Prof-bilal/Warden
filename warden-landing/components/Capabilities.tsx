import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import Diagram from "@/components/Diagram";

const GROUPS: StepperItem[] = [
  {
    title: "Files",
    summary: "Deny-by-default read/write.",
    body: "Grant read or write access to selected paths. Keep credential folders and unrelated projects outside those grants. Runtime support varies by backend.",
    image: <Diagram src="/diagrams/filesystem-capabilities.jpeg" alt="Filesystem capabilities architecture with deny-by-default grants" />,
  },
  {
    title: "Network",
    summary: "Hostname allowlist.",
    body: "Choose which hosts and APIs a tool may contact. The enforced proxy path checks the allowlist before upstream DNS resolution.",
    image: <Diagram src="/diagrams/network-access.jpeg" alt="Network access security architecture with hostname allowlist" />,
  },
  {
    title: "Environment",
    summary: "Named variables only.",
    body: "Only named variables pass through. Empty allowlist means an empty environment.",
    image: <Diagram src="/diagrams/env-filtering-process.jpeg" alt="Environment filtering as the server process starts inside the sandbox" />,
  },
  {
    title: "Limits",
    summary: "Timeout + memory caps.",
    body: "Set supported wall-clock and memory limits. Enforcement varies by backend; sampled memory can briefly overshoot. Warden does not throttle CPU usage.",
    image: <Diagram src="/diagrams/process-termination-limits.jpeg" alt="Process termination on resource limit breach" />,
  },
  {
    title: "Audit",
    summary: "JSONL decision log.",
    body: "Inspect JSONL audit events with `warden logs`. Proxy decisions are recorded; file-denial coverage differs by platform.",
    image: <Diagram src="/diagrams/audit-trail.jpeg" alt="Security audit trail architecture logging every allow and deny decision" />,
  },
  {
    title: "Policy tooling",
    summary: "init → trace → doctor.",
    body: "`warden trace` runs trusted code unsandboxed to observe accesses. `warden init` drafts a policy for review, and `warden doctor` checks readiness.",
    image: <Diagram src="/diagrams/policy-flow.jpeg" alt="Security policy architecture flow from creation to enforcement" />,
  },
  {
    title: "MCP client proxy",
    summary: "JSON-RPC filtering.",
    body: "`warden proxy` filters JSON-RPC in both directionstool allowlists, secret deny patterns, payload capsfor local stdio and remote HTTPS upstreams.",
    image: <Diagram src="/diagrams/mcp-client-proxy.jpeg" alt="MCP client proxy security architecture filtering JSON-RPC" />,
  },
  {
    title: "CI and deployment",
    summary: "GitHub Action + K8s.",
    body: "A GitHub Action wraps CI jobs in the same policy; `warden k8s render` emits hardened Deployment + NetworkPolicy manifests.",
    image: <Diagram src="/diagrams/ci-pipeline.jpeg" alt="CI pipeline governing sandboxed builds and deployments" />,
  },
];

export default function Capabilities() {
  return (
    <section id="capabilities" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        <h2 className="font-hero text-[1.75rem] font-bold leading-[1.15] tracking-[-0.02em] text-paper md:text-[2rem]">
          What&apos;s supported.
        </h2>
        <p className="mt-3 max-w-[34rem] font-hero text-[1rem] leading-[1.65] text-muted">
          Explore the controls and workflows implemented in the CLI.
          Support and verification differ across backends; check the platform
          notes before relying on a feature.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Capabilities" items={GROUPS} />
        </div>
      </div>
    </section>
  );
}
