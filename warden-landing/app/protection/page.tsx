import Link from "next/link";
import EcosystemShell from "@/components/EcosystemShell";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Warden protection boundary and evidence",
  description:
    "Understand what local execution, gateway wrapping, filtering proxy and tracing protect, and what still needs verification.",
  path: "/protection",
});
const modes = [
  [
    "warden connect / serve (local)",
    "Managed process and routed MCP connection",
    "Digest-pinned sandbox plus tool/argument/resource/prompt rules",
    "2025-11-25 bounded subset; unsupported capabilities are refused. Native platform and host coverage still need review.",
  ],
  [
    "warden serve (remote)",
    "Routed MCP traffic",
    "Separate authentication, credentials and client sessions; filtered discovery/calls",
    "Provider process is not sandboxed. No automatic upstream OAuth, MRTR, tasks or subscriptions.",
  ],
  [
    "warden run",
    "Local managed process",
    "OS sandbox; explicit file, egress and environment grants",
    "Backend required. Runtime mounts and scratch directories are additional to data grants. Allowed API actions still require provider-side privileges.",
  ],
  [
    "Client / gateway wrap",
    "Configuration change",
    "Launches local stdio servers through warden run",
    "Wrapping is not an MCP workflow test. Remote entries cannot gain process isolation through a config edit.",
  ],
  [
    "warden proxy",
    "Messages routed through custom TCP listener",
    "Tool-name / content filtering and upstream environment allowlisting",
    "Local proxy upstreams are not OS-sandboxed by proxy itself. The downstream listener is not a standard Streamable HTTP endpoint.",
  ],
  [
    "warden trace",
    "Observation of one workload",
    "Captures accesses for a candidate policy",
    "Runs unsandboxed. Use trusted code, disposable test data and fake credentials. Observation does not prove safety.",
  ],
];
export default function Page() {
  return (
    <EcosystemShell
      active="/protection"
      eyebrow="Boundary & evidence"
      title="Know what your connection protects."
      description="Protection follows the managed process and the routed connection. Warden does not cover unrelated shell tools, direct connections, or a remote provider’s machine."
    >
      <div className="overflow-x-auto rounded-2xl border border-ink-700">
        <table className="w-full min-w-[780px] text-left text-sm">
          <caption className="sr-only">
            Warden mode capabilities and limitations
          </caption>
          <thead className="bg-ink-800 text-xs text-muted">
            <tr>
              {["Mode", "Boundary", "Controls", "Limitations"].map((h) => (
                <th key={h} scope="col" className="px-5 py-4 font-medium">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {modes.map(([mode, boundary, controls, limits]) => (
              <tr key={mode} className="border-t border-ink-700 align-top">
                <th
                  scope="row"
                  className="whitespace-nowrap px-5 py-5 font-mono text-xs text-blueprint"
                >
                  {mode}
                </th>
                <td className="px-5 py-5 text-muted">{boundary}</td>
                <td className="px-5 py-5 leading-relaxed">{controls}</td>
                <td className="max-w-sm px-5 py-5 leading-relaxed text-muted">
                  {limits}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="mt-8 grid gap-6 md:grid-cols-2">
        <section className="rounded-xl border border-ink-700 p-7">
          <h2 className="text-xl font-semibold">Evidence has levels</h2>
          <ol className="mt-5 space-y-4 text-sm leading-relaxed text-muted">
            <li>
              <strong className="text-paper">Metadata reviewed:</strong> an
              upstream and version are identified.
            </li>
            <li>
              <strong className="text-paper">Fixture checked:</strong> policy
              parsing and config transformations pass deterministic checks.
            </li>
            <li>
              <strong className="text-paper">Backend exercised:</strong> an
              actual sandbox allows and blocks representative operations.
            </li>
            <li>
              <strong className="text-paper">Workflow verified:</strong> a
              pinned server and client complete the real task and denial checks
              on the advertised platform.
            </li>
          </ol>
        </section>
        <section className="rounded-xl border border-ink-700 p-7">
          <h2 className="text-xl font-semibold">Local setup checks</h2>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            Applying a wrapper probes the selected sandbox using an inert Warden
            process before writing. It never starts the target server for that
            readiness check. Failure leaves the configuration unchanged.
          </p>
          <p className="mt-4 text-sm leading-relaxed text-muted">
            After restart, verify one allowed task and one denied task. A
            configured wrapper remains marked “workflow unverified” in the CLI
            inventory until evidence exists.
          </p>
          <Link
            href="/docs/ecosystem-setup"
            className="mt-5 block text-sm text-blueprint"
          >
            Read the verification guide →
          </Link>
        </section>
      </div>
      <p className="mt-8 rounded-xl border border-progress/30 bg-progress-subtle p-5 text-sm leading-relaxed text-muted">
        Native Linux/macOS egress filtering covers HTTP/HTTPS through the
        enforced proxy. Raw database TCP needs separate transport work. The
        development gateway supports a bounded 2025-11-25 subset; newer
        protocol-era support and complete remote assurance remain release work.
        Auditing coverage depends on the backend; a policy fixture is not fresh
        end-to-end evidence.
      </p>
    </EcosystemShell>
  );
}
