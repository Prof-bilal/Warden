import StepperDetail from "@/components/StepperDetail";
import type { StepperItem } from "@/components/StepperDetail";
import Diagram from "@/components/Diagram";

const GROUPS: StepperItem[] = [
  {
    title: "Filesystem",
    summary: "Deny-by-default read/write.",
    body: "Deny-by-default read/write grants. Anything unlisted is invisible — not merely unreadable.",
    image: <Diagram src="/diagrams/filesystem-capabilities.jpeg" alt="Filesystem capabilities architecture with deny-by-default grants" />,
  },
  {
    title: "Network",
    summary: "Hostname allowlist.",
    body: "Egress hostname allowlist with DNS blocked before resolution. No grant, no connection.",
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
    body: "Wall-clock timeout and memory caps enforced by the kernel, with process-tree termination.",
    image: <Diagram src="/diagrams/process-termination-limits.jpeg" alt="Process termination on resource limit breach" />,
  },
  {
    title: "Audit",
    summary: "JSONL decision log.",
    body: "Every allow/deny decision lands in a JSONL log you can tail with `warden logs`.",
    image: <Diagram src="/diagrams/audit-trail.jpeg" alt="Security audit trail architecture logging every allow and deny decision" />,
  },
  {
    title: "Policy tooling",
    summary: "init → trace → doctor.",
    body: "`warden trace` records real access, `warden init` drafts the policy, `warden doctor` verifies readiness.",
    image: <Diagram src="/diagrams/policy-flow.jpeg" alt="Security policy architecture flow from creation to enforcement" />,
  },
  {
    title: "MCP client proxy",
    summary: "JSON-RPC filtering.",
    body: "`warden proxy` filters JSON-RPC in both directions — tool allowlists, secret deny patterns, payload caps — for local stdio and remote HTTPS upstreams.",
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
        <h2 className="text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper">
          What&apos;s supported.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Every item below ships in the CLI today and is covered by the test
          suite. Nothing on this page is a roadmap promise — that list lives
          separately, labeled as roadmap.
        </p>

        <div className="mt-10">
          <StepperDetail sectionLabel="Capabilities" items={GROUPS} />
        </div>
      </div>
    </section>
  );
}
