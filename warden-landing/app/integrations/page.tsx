import Link from "next/link";
import EcosystemShell from "@/components/EcosystemShell";
import IntegrationBuilder from "@/components/IntegrationBuilder";
import { createMetadata } from "@/lib/seo";
export const metadata = createMetadata({
  title: "Connect any compatible MCP agent",
  description:
    "Build a reviewed local or HTTP MCP gateway connection for custom agents and SDKs.",
  path: "/integrations",
});
export default function Page() {
  return (
    <EcosystemShell
      active="/integrations"
      eyebrow="MCP connections"
      title="Bring your agent. Choose its access."
      description="A standard command or authenticated HTTP endpoint lets compatible MCP hosts share the same reviewed rules."
    >
      <div className="mb-8 rounded-xl border border-progress/30 bg-progress-subtle p-5 text-sm leading-relaxed text-muted">
        Development checkout · Protocol 2025-11-25 request/response subset.
        MRTR, tasks, subscriptions, sampling and elicitation are refused. Newer
        clients must negotiate the supported version.{" "}
        <Link href="/docs/mcp-gateway" className="text-blueprint underline">
          Check the supported boundary
        </Link>
        .
      </div>
      <IntegrationBuilder />
      <div className="mt-10 grid gap-6 md:grid-cols-2">
        <section className="rounded-xl border border-ink-700 p-6">
          <h2 className="text-xl font-semibold">SDKs and custom agents</h2>
          <p className="mt-3 text-sm text-muted">
            Use structured stdio arguments or connect to /mcp with a bearer
            credential. Examples cover OpenAI Agents, LangChain and Pydantic AI.
          </p>
          <Link href="/docs/mcp-gateway" className="mt-5 block text-blueprint">
            SDK recipes →
          </Link>
        </section>
        <section className="rounded-xl border border-ink-700 p-6">
          <h2 className="text-xl font-semibold">Remote providers</h2>
          <p className="mt-3 text-sm leading-relaxed text-muted">
            Warden can filter a remote MCP endpoint. It cannot sandbox the
            provider&apos;s machine. Upstream credentials stay separate from
            client authentication.
          </p>
          <Link href="/docs/mcp-gateway" className="mt-5 block text-blueprint">
            Hosting and authentication →
          </Link>
        </section>
      </div>
    </EcosystemShell>
  );
}
