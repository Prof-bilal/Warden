import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
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
    <details className="rounded-lg border border-ink-800 bg-ink-900/40 p-4 lg:hidden">
      <summary className="cursor-pointer list-none font-sans text-sm font-semibold text-paper">
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Docs", url: `${SITE_URL}/docs` },
        ])}
      />
      <Nav />
      <div className="mx-auto max-w-content px-6 pb-20 pt-8 md:pt-12">
        <nav className="mb-6 font-sans text-[0.875rem] text-muted" aria-label="Breadcrumb">
          <a href="/" className="transition-colors hover:text-paper">Warden</a>
          <span className="mx-1.5 text-ink-600">/</span>
          <span className="text-paper">Docs</span>
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
            <h1 className="font-sans text-[2.75rem] font-semibold leading-[1.08] tracking-[-0.03em] text-paper">
              Warden documentation
            </h1>
            <p className="mt-5 max-w-[42rem] font-sans text-[1.0625rem] leading-[1.7] text-muted">
              Run local tools with only the files, network destinations, environment
              variables, and resources they need. Start with the quickstart or protect
              an AI coding agent with a sanitized workspace.
            </p>

            <div className="mt-12 grid gap-4 md:grid-cols-2">
              {CARDS.map((card) => (
                <a
                  key={card.title}
                  href={card.href}
                  className="group rounded-lg border border-ink-800 bg-ink-900/40 p-6 transition-colors hover:border-blueprint/50 hover:bg-ink-900"
                >
                  <h2 className="font-sans text-[1.125rem] font-semibold text-paper">
                    {card.title}
                    <span className="ml-2 inline-block text-blueprint transition-transform group-hover:translate-x-0.5" aria-hidden="true">
                      →
                    </span>
                  </h2>
                  <p className="mt-2 font-sans text-[0.9375rem] leading-[1.65] text-muted">{card.body}</p>
                </a>
              ))}
            </div>
          </article>
        </div>
      </div>
      <Footer />
    </main>
  );
}
