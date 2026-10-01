import Link from "next/link";
import EcosystemShell from "@/components/EcosystemShell";
import { clients } from "@/lib/ecosystem";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Agent and MCP compatibility",
  description:
    "Configuration adapters, stdio and authenticated HTTP connections, with scoped host test evidence.",
  path: "/compatibility",
});
export default function Page() {
  return (
    <EcosystemShell
      active="/compatibility"
      eyebrow="Cross-agent support"
      title="One boundary. Many MCP hosts."
      description="Local command wrapping works independently of the model vendor. Configuration adapters reduce setup friction; each host still needs a real workflow check."
    >
      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        {[
          ["8", "named client adapters"],
          ["stdio", "portable local execution"],
          ["Codex Linux", "app-server workflow tested"],
        ].map(([value, label]) => (
          <div key={label} className="rounded-xl border border-ink-700 p-6">
            <p className="text-2xl font-semibold text-blueprint">{value}</p>
            <p className="mt-2 text-xs text-muted">{label}</p>
          </div>
        ))}
      </div>
      <div className="overflow-x-auto rounded-2xl border border-ink-700">
        <table className="w-full min-w-[700px] text-left text-sm">
          <caption className="sr-only">
            Local configuration adapters and scoped host workflow evidence
          </caption>
          <thead className="bg-ink-800 text-xs text-muted">
            <tr>
              {["Client", "Config", "Scope", "Evidence", "Setup"].map((h) => (
                <th key={h} scope="col" className="px-5 py-4 font-medium">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {clients.map((c) => (
              <tr key={c.id} className="border-t border-ink-700">
                <th scope="row" className="px-5 py-5 font-medium">
                  {c.name}
                </th>
                <td className="px-5 text-muted">{c.format}</td>
                <td className="px-5 text-muted">{c.scopes.join(" / ")}</td>
                <td className="px-5">
                  <span className="text-xs text-progress">
                    {c.id === "codex"
                      ? "Linux app-server 0.159.2: npm + stdio tested"
                      : "Adapter fixtures; live host unverified"}
                  </span>
                  <a
                    href={c.source}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-2 block text-xs text-blueprint"
                  >
                    Official config reference ↗
                  </a>
                </td>
                <td className="px-5">
                  <Link href={`/setup/${c.id}`} className="text-blueprint">
                    Set up →
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-4 text-xs leading-relaxed text-muted">
        The Codex check used the official filesystem server in an isolated
        configuration: discovery, allowed reads, denied writes and denied reads
        outside the policy. It made no model turn. Desktop UI and other native
        platforms still need their own workflow checks.
      </p>
      <div className="mt-8 grid gap-6 md:grid-cols-2">
        <section className="rounded-xl border border-ink-700 p-6">
          <h2 className="text-xl font-semibold">Custom agents & SDKs</h2>
          <p className="mt-3 text-sm leading-relaxed text-muted">
            Use a stdio connection that launches Warden with a policy, followed
            by the original command and argument array. In-process servers need
            a separate process to use this boundary.
          </p>
          <Link
            href="/setup/generic"
            className="mt-5 block text-sm text-blueprint"
          >
            Generic setup →
          </Link>
        </section>
        <section className="rounded-xl border border-progress/30 bg-progress-subtle p-6">
          <h2 className="text-xl font-semibold">Remote & hosted agents</h2>
          <p className="mt-3 text-sm leading-relaxed text-muted">
            The development gateway exposes authenticated Streamable HTTP for
            its supported 2025-11-25 subset. Public hosting requires a reachable
            HTTPS endpoint. Remote providers&apos; machines are not sandboxed.
          </p>
          <Link
            href="/integrations"
            className="mt-5 block text-sm text-blueprint"
          >
            Build a gateway connection →
          </Link>
        </section>
      </div>
    </EcosystemShell>
  );
}
