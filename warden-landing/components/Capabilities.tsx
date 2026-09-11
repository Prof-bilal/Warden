// Capabilities reflect the audited implementation (see future.md audit):
// only features with committed code + tests are listed here. The MCP proxy
// note is worded to match docs/client-proxy.md exactly — message-level
// filtering today; message framing is JSON-RPC lines, not HTTP.
const GROUPS = [
  {
    label: "Filesystem",
    body: "Deny-by-default read/write grants. Anything unlisted is invisible — not merely unreadable.",
  },
  {
    label: "Network",
    body: "Egress hostname allowlist with DNS blocked before resolution. No grant, no connection.",
  },
  {
    label: "Environment",
    body: "Only named variables pass through. Empty allowlist means an empty environment.",
  },
  {
    label: "Limits",
    body: "Wall-clock timeout and memory caps enforced by the kernel, with process-tree termination.",
  },
  {
    label: "Audit",
    body: "Every allow/deny decision lands in a JSONL log you can tail with `warden logs`.",
  },
  {
    label: "Policy tooling",
    body: "`warden trace` records real access, `warden init` drafts the policy, `warden doctor` verifies readiness.",
  },
  {
    label: "MCP client proxy",
    body: "`warden proxy` filters JSON-RPC in both directions — tool allowlists, secret deny patterns, payload caps — for local stdio and remote HTTPS upstreams.",
  },
  {
    label: "CI and deployment",
    body: "A GitHub Action wraps CI jobs in the same policy; `warden k8s render` emits hardened Deployment + NetworkPolicy manifests.",
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

        <dl className="mt-10 grid gap-x-10 gap-y-6 md:grid-cols-2">
          {GROUPS.map((g) => (
            <div
              key={g.label}
              className="flex flex-col gap-1 border-t border-ink-800 pt-4"
            >
              <dt className="font-mono text-[0.8125rem] uppercase tracking-[0.06em] text-blueprint">
                {g.label}
              </dt>
              <dd className="text-[0.9375rem] leading-[1.55] text-muted">
                {g.body}
              </dd>
            </div>
          ))}
        </dl>
      </div>
    </section>
  );
}
