import Link from "next/link";
import { notFound } from "next/navigation";
import EcosystemShell from "@/components/EcosystemShell";
import { policyPacks } from "@/lib/ecosystem";
import { createMetadata } from "@/lib/seo";
export function generateStaticParams() {
  return policyPacks.map((p) => ({ id: p.id }));
}
export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const p = policyPacks.find((p) => p.id === id);
  return createMetadata({
    title: `${p?.name ?? "MCP"} policy profile`,
    description: p?.profile ?? "MCP candidate policy",
    path: `/policies/${id}`,
  });
}
export default async function Page({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const p = policyPacks.find((p) => p.id === id);
  if (!p) notFound();
  return (
    <EcosystemShell
      active="/policies"
      eyebrow={`${p.category} / candidate profile`}
      title={p.name}
      description={p.profile}
    >
      <div className="grid gap-6 lg:grid-cols-[1.4fr_1fr]">
        <section className="rounded-2xl border border-ink-700 bg-ink-900 p-7">
          <h2 className="text-xl font-semibold">What this profile grants</h2>
          <dl className="mt-6 space-y-5 text-sm">
            <div>
              <dt className="text-muted">Files</dt>
              <dd className="mt-1">
                {p.pathMode === "none"
                  ? "Read the prepared runtime directory. No data directory."
                  : p.pathMode === "read"
                    ? "Read the prepared runtime and one selected data directory. Writes denied."
                    : "Read the prepared runtime. Read and write one selected storage directory."}
              </dd>
            </div>
            <div>
              <dt className="text-muted">API hosts</dt>
              <dd className="mt-1 font-mono">
                {p.hosts.join(", ") || "None · network denied"}
              </dd>
            </div>
            <div>
              <dt className="text-muted">Environment variable names</dt>
              <dd className="mt-1 font-mono">{p.env.join(", ") || "None"}</dd>
            </div>
            <div>
              <dt className="text-muted">Resource settings</dt>
              <dd className="mt-1">
                512 MB memory · 1 hour session cap. Enforcement depends on the
                selected backend.
              </dd>
            </div>
          </dl>
          <p className="mt-7 border-t border-ink-700 pt-5 text-sm leading-relaxed text-muted">
            {p.note}
          </p>
        </section>
        <aside className="rounded-2xl border border-ink-700 p-7">
          <span className="text-xs font-medium uppercase tracking-wide text-progress">
            Candidate · workflow unverified
          </span>
          <h2 className="mt-5 text-xl font-semibold">Version & evidence</h2>
          <p className="mt-4 break-all font-mono text-xs text-muted">
            {p.artifact}@{p.version}
          </p>
          <p className="mt-3 text-sm leading-relaxed text-muted">
            Metadata reviewed {p.reviewed}. This identifies an upstream release;
            it does not attest your installed runtime or certify the server.
          </p>
          <a
            href={p.upstream}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-5 block text-sm text-blueprint"
          >
            View upstream source ↗
          </a>
          <Link
            href={`/setup?pack=${p.id}`}
            className="mt-8 block rounded-xl bg-blueprint px-5 py-3 text-center text-sm font-semibold text-ink-950"
          >
            Set up this profile →
          </Link>
          <Link
            href="/protection"
            className="mt-4 block text-center text-sm text-muted"
          >
            Read protection limits
          </Link>
        </aside>
      </div>
    </EcosystemShell>
  );
}
