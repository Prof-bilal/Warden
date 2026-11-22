import Link from "next/link";
import { ArrowRight } from "lucide-react";
import PageShell from "@/components/PageShell";
import { createMetadata } from "@/lib/seo";

export const metadata = createMetadata({
  title: "Warden 2.0",
  description: "Review package installs, run local MCP servers under a policy, and catch changes to bridged tool definitions.",
  path: "/v2",
});

const workflows = [
  {
    title: "Run a reviewed launcher",
    description: "The Filesystem pack selects a bundled release and grants read-only access to a chosen directory. Use an explicit policy for other launchers.",
    href: "/docs/zero-config",
    link: "Run the Filesystem quickstart",
  },
  {
    title: "Review before installing",
    description: "Install Guard checks npm and PyPI releases against a default 48-hour cooldown. An exact npm allowlist entry can authorize the selected package’s hooks in an offline sandbox.",
    href: "/docs/install-guard",
    link: "Read the Install Guard guide",
  },
  {
    title: "Catch changed MCP tools",
    description: "The bridge can pin the first definition it sees. A changed definition hides the tool, blocks calls, and leaves a journal entry for review.",
    href: "/docs/mcp-tool-pins",
    link: "Set up MCP tool pins",
  },
];

export default function WardenTwoPage() {
  return (
    <PageShell>
      <section aria-labelledby="release-title" className="grid min-w-0 gap-9 border-b border-hair px-6 py-12 md:px-10 lg:grid-cols-[1fr_1.25fr] lg:items-center lg:py-20">
        <div className="min-w-0">
          <p className="font-mono text-xs uppercase tracking-micro text-acid">Warden 2.0 preview</p>
          <h1 id="release-title" className="mt-4 max-w-lg text-4xl font-semibold leading-[1.05] tracking-tight md:text-5xl">
            Install, run, and review MCP servers.
          </h1>
          <p className="mt-5 max-w-lg leading-7 text-offwhite/70">
            Check a release before installation. Give the local process only the access it needs.
            Review tool-definition changes at the bridge.
          </p>
          <div className="mt-7 flex flex-wrap gap-3">
            <Link href="/docs/zero-config" className="inline-flex items-center gap-2 rounded-md bg-acid px-4 py-3 text-sm font-semibold text-void focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-acid">
              Start with Filesystem <ArrowRight size={15} aria-hidden="true" />
            </Link>
            <Link href="/docs/migration" className="rounded-md border border-hair px-4 py-3 text-sm font-semibold text-offwhite focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-acid">
              Review migration
            </Link>
          </div>
        </div>

        <figure className="min-w-0">
          <video
            aria-label="Warden 2.0 introduction"
            className="aspect-video w-full rounded-lg border border-hair bg-void"
            controls
            playsInline
            preload="metadata"
            poster="/videos/warden-v2-intro-poster.jpg"
          >
            <source src="/videos/warden-v2-intro.mp4" type="video/mp4" />
          </video>
          <figcaption className="mt-3 text-xs text-offwhite/70">
            <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
              <span>70-second walkthrough · Linux fixtures</span>
              <a href="/videos/warden-v2-intro.mp4" className="text-acid underline underline-offset-4">Open the MP4</a>
            </div>
            <details className="mt-3">
              <summary className="w-fit cursor-pointer text-acid">Read the walkthrough</summary>
              <ol className="mt-2 list-decimal space-y-2 pl-5 leading-5">
                <li>The bundled Filesystem server reads the shared file. An attempted write, outside read, and access to a protected environment file are blocked.</li>
                <li>Install Guard refuses the two-hour-old npm fixture under the default 48-hour cooldown.</li>
                <li>The bridge lists the read tool on first discovery. After its description changes, the tool disappears, direct calls are denied, and a journal entry records the reason.</li>
                <li>Run <code>warden migrate</code> to preview changes to existing policies and launchers.</li>
              </ol>
            </details>
          </figcaption>
        </figure>
      </section>

      <section aria-labelledby="workflows-title" className="border-b border-hair px-6 py-10 md:px-10 md:py-14">
        <h2 id="workflows-title" className="text-2xl font-semibold tracking-tight">What’s in 2.0</h2>
        <div className="mt-6 grid gap-3 md:grid-cols-3">
          {workflows.map(({ title, description, href, link }) => (
            <article key={href} className="flex min-w-0 flex-col rounded-lg border border-hair bg-cell p-5">
              <h3 className="text-lg font-semibold">{title}</h3>
              <p className="mb-5 mt-3 flex-1 text-sm leading-6 text-offwhite/70">{description}</p>
              <Link href={href} className="text-sm font-medium text-acid underline underline-offset-4">{link}</Link>
            </article>
          ))}
        </div>
        <p className="mt-6 max-w-3xl text-sm leading-6 text-offwhite/70">
          The <Link href="/docs/claude-code-hook" className="text-acid underline underline-offset-4">Claude Code hook</Link> screens supported requests as an advisory check.{" "}
          Use the native sandbox when you need OS-level enforcement.
        </p>
      </section>
    </PageShell>
  );
}
