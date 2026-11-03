import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import DocsSidebar from "@/components/DocsSidebar";

export const metadata: Metadata = {
  title: { absolute: "Warden documentation" },
  description:
    "Learn how to run local tools and MCP servers with explicit file, network, environment, and resource permissions.",
  alternates: { canonical: `${SITE_URL}/docs` },
  openGraph: {
    title: "Warden documentation",
    description:
      "Learn how to run local tools and MCP servers with explicit file, network, environment, and resource permissions.",
    url: `${SITE_URL}/docs`,
    images: [{ url: `${SITE_URL}/og-image.png`, width: 1200, height: 630, alt: "Warden documentation" }],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden documentation",
    description:
      "Learn how to run local tools and MCP servers with explicit file, network, environment, and resource permissions.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

const CARDS = [
  {
    title: "Start here",
    body: "Install Warden, write a small policy, and run your first protected local command.",
    href: "/docs/quickstart",
  },
  {
    title: "Protect an AI agent",
    body: "Let a coding agent work on a sanitized project copy while protected values stay hidden and host changes require review.",
    href: "/docs/agent-workspace",
  },
  {
    title: "Write a policy",
    body: "Choose the files, hosts, environment names, and runtime limits a tool actually needs.",
    href: "/docs/write-policy",
  },
  {
    title: "Connect an MCP server",
    body: "Run local MCP servers through Warden and keep their filesystem, network, and environment access explicit.",
    href: "/docs/client-proxy",
  },
  {
    title: "Install",
    body: "Check platform requirements for Linux, macOS, Windows, and the Docker fallback before you run.",
    href: "/docs/install",
  },
  {
    title: "Policy schema",
    body: "Look up every policy field, validation rule, and platform-specific enforcement note.",
    href: "/docs/schema",
  },
  {
    title: "CLI reference",
    body: "Find the commands and flags for running, tracing, logging, connecting, and reviewing.",
    href: "/docs/cli",
  },
  {
    title: "Compatibility matrix",
    body: "See which tested MCP servers work, need setup, or have known limitations.",
    href: "/docs/compatibility",
  },
  {
    title: "Security review",
    body: "Understand what Warden enforces, what it cannot guarantee, and how to verify your own setup.",
    href: "/docs/security",
  },
];

function Sidebar({ mobile = false }: { mobile?: boolean }) {
  return mobile ? (
    <details className="border border-hair bg-cell p-4 lg:hidden">
      <summary className="cursor-pointer list-none font-mono text-[12px] uppercase tracking-micro text-offwhite/70">
        Browse documentation
      </summary>
      <div className="mt-5">
        <DocsSidebar />
      </div>
    </details>
  ) : (
    <DocsSidebar />
  );
}

export default function Docs() {
  return (
    <PageShell>
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Docs", url: `${SITE_URL}/docs` },
        ])}
      />
      <div className="px-6 pb-20 pt-8 md:pt-12">
        <nav
          className="mb-6 flex items-center gap-2 font-mono text-[10.5px] uppercase tracking-micro text-offwhite/35"
          aria-label="Breadcrumb"
        >
          <Link href="/" className="transition-colors hover:text-offwhite">
            Warden
          </Link>
          <span className="text-offwhite/20">/</span>
          <span className="text-offwhite/70">Docs</span>
        </nav>

        <div className="mb-8 lg:hidden">
          <Sidebar mobile />
        </div>

        <div className="flex gap-12">
          <aside className="hidden w-56 shrink-0 lg:block">
            <div className="sticky top-24">
              <Sidebar />
            </div>
          </aside>

          <article className="min-w-0 flex-1">
            <h1 className="font-mono text-[1.9rem] font-semibold leading-[1.1] tracking-[-0.02em] text-offwhite sm:text-[2.4rem]">
              Warden documentation
            </h1>
            <p className="mt-5 max-w-[42rem] font-sans text-[15px] leading-[1.75] text-offwhite/45">
              Run local tools with only the files, network destinations, environment
              variables, and resources they need. Start with the quickstart or protect
              an AI coding agent with a sanitized workspace.
            </p>

            <div className="mt-12 grid gap-px border border-hair bg-hair md:grid-cols-2">
              {CARDS.map((card) => (
                <Link
                  key={card.title}
                  href={card.href}
                  className="group bg-cell p-6 transition-colors hover:bg-offwhite/[0.03]"
                >
                  <h2 className="font-mono text-[15px] font-semibold text-offwhite">
                    {card.title}
                    <span
                      className="ml-2 inline-block text-acid transition-transform group-hover:translate-x-0.5"
                      aria-hidden="true"
                    >
                      →
                    </span>
                  </h2>
                  <p className="mt-2 font-sans text-[13px] leading-[1.7] text-offwhite/45">
                    {card.body}
                  </p>
                </Link>
              ))}
            </div>
          </article>
        </div>
      </div>
    </PageShell>
  );
}
