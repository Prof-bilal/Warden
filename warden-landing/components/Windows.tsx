const WINDOWS_LAYERS = [
  {
    name: "AppContainer Token",
    description:
      "Every sandboxed process runs under a LowBox token that denies all filesystem, network, and environment access by default. The token is the hard boundary — no syscall can cross it.",
  },
  {
    name: "WFP Egress Filters",
    description:
      "Windows Filtering Platform rules permit only the loopback proxy bridge and block every other outbound connection. DNS is denied by the token; TCP is denied by the filters.",
  },
  {
    name: "Job Object Limits",
    description:
      "Wall-clock timeout, memory cap, and kill-on-close are enforced by the kernel. A runaway process is terminated with its entire tree — no orphaned children.",
  },
  {
    name: "ETW Audit Trail",
    description:
      "A private real-time trace session captures kernel file I/O events scoped to the sandbox tree. Every file access is logged, not guessed — the audit is the product.",
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

        <div className="mt-10 grid gap-6 sm:grid-cols-2">
          {WINDOWS_LAYERS.map((layer) => (
            <div
              key={layer.name}
              className="rounded-sm border border-ink-700 bg-ink-900 p-5"
            >
              <dt className="font-mono text-[0.875rem] text-blueprint">
                {layer.name}
              </dt>
              <dd className="mt-2 text-[0.9375rem] leading-[1.55] text-muted">
                {layer.description}
              </dd>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
