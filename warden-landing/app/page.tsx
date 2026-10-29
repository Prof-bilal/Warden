import type { Metadata } from "next";
import Link from "next/link";
import { SITE_URL } from "@/lib/seo";
import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import BoundaryDemo from "@/components/BoundaryDemo";
import AgentWorkspace from "@/components/AgentWorkspace";
import Cta from "@/components/Cta";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";

export const metadata: Metadata = {
  title: { absolute: "Warden — Permissions for Local AI Tools" },
  description:
    "Run local AI tools and MCP servers with explicit file, network, environment, and resource permissions. Open source, local, and fail-closed.",
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    title: "Warden — Permissions for Local AI Tools",
    description:
      "Run local AI tools and MCP servers with explicit file, network, environment, and resource permissions. Open source, local, and fail-closed.",
    url: SITE_URL,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden — Permissions for Local AI Tools",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden — Permissions for Local AI Tools",
    description:
      "Give local AI tools and MCP servers only the access they need, with explicit permissions.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

export default function Home() {
  const softwareSchema = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "Warden",
    applicationCategory: "DeveloperApplication",
    operatingSystem: ["Linux", "macOS", "Windows"],
    description:
      "An open-source permission layer for local AI tools, MCP servers, and other local processes, using OS-native sandboxing with explicit permissions.",
    "@id": `${SITE_URL}/#software`,
    url: SITE_URL,
    isAccessibleForFree: true,
    downloadUrl: "https://github.com/Prof-bilal/Warden/releases",
    installUrl: "https://www.npmjs.com/package/warden-sandbox-cli",
    license: "https://opensource.org/licenses/MIT",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "USD",
    },
    author: {
      "@type": "Person",
      name: "Abdullah Bilal",
      alternateName: "Prof-bilal",
      url: "https://github.com/Prof-bilal",
    },
    keywords: "MCP, sandbox, security, AI tooling, Model Context Protocol",
    screenshot: `${SITE_URL}/og-image.png`,
  };

  return (
    <main className="min-h-screen bg-ink-950">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareSchema) }}
      />
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "WebPage",
          "@id": `${SITE_URL}/#webpage`,
          url: SITE_URL,
          name: "Warden — Permissions for Local AI Tools",
          description:
            "Run local AI tools and MCP servers with explicit file, network, environment, and resource permissions. Open source, local, and fail-closed.",
          isPartOf: { "@type": "WebSite", name: "Warden", url: SITE_URL },
          mainEntity: { "@id": `${SITE_URL}/#software` },
        }}
      />
      <Nav />
      <Hero />
      <section className="border-y border-ink-800 bg-ink-900/30">
        <div className="mx-auto max-w-content px-6 py-20 md:py-24">
          <div className="max-w-2xl">
            <p className="font-sans text-xs font-semibold uppercase tracking-[0.14em] text-blueprint">The problem</p>
            <h2 className="mt-4 font-sans text-[2rem] font-semibold leading-[1.1] tracking-[-0.03em] text-paper md:text-[2.5rem]">
              AI tools run with your permissions.
            </h2>
            <p className="mt-5 font-sans text-base leading-[1.75] text-muted">
              A local agent or MCP server can inherit access to files, environment values, and network destinations available to the user who started it. Warden turns that implicit access into explicit policy.
            </p>
          </div>
          <div className="mt-10 grid gap-4 md:grid-cols-3">
            {[
              ["Files", "Keep project access separate from credentials and unrelated folders."],
              ["Secrets", "Pass only the environment names a tool actually needs."],
              ["Network", "Allow specific APIs and block undeclared destinations."],
            ].map(([title, body]) => (
              <div key={title} className="rounded-lg border border-ink-700 bg-ink-950/60 p-5">
                <h3 className="font-sans text-base font-semibold text-paper">{title}</h3>
                <p className="mt-2 font-sans text-sm leading-[1.65] text-muted">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>
      <BoundaryDemo />
      <AgentWorkspace />
      <section className="border-t border-ink-800">
        <div className="mx-auto grid max-w-content gap-10 px-6 py-20 md:py-24 lg:grid-cols-[0.8fr_1.2fr] lg:items-start lg:gap-16">
          <div>
            <p className="font-sans text-xs font-semibold uppercase tracking-[0.14em] text-blueprint">One policy</p>
            <h2 className="mt-4 font-sans text-[2rem] font-semibold leading-[1.1] tracking-[-0.03em] text-paper">Define what the tool needs.</h2>
            <p className="mt-4 font-sans text-base leading-[1.75] text-muted">Everything not listed stays outside the sandbox. Keep the first policy small, run it, then expand only when the audit shows a real need.</p>
            <Link href="/docs/schema" className="mt-6 inline-flex font-sans text-sm font-semibold text-blueprint hover:text-paper">Explore the policy reference →</Link>
          </div>
          <pre className="overflow-x-auto rounded-xl border border-ink-700 bg-ink-950 p-5 font-mono text-sm leading-[1.8] text-paper/85"><code>{`filesystem:\n  read:\n    - ./project\n\nnetwork:\n  allow:\n    - api.github.com\n\nenv:\n  allow:\n    - GITHUB_TOKEN`}</code></pre>
        </div>
      </section>
      <section className="border-t border-ink-800 bg-ink-900/30">
        <div className="mx-auto max-w-content px-6 py-20 md:py-24">
          <div className="flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-2xl">
              <p className="font-sans text-xs font-semibold uppercase tracking-[0.14em] text-blueprint">Proof, not promises</p>
              <h2 className="mt-4 font-sans text-[2rem] font-semibold leading-[1.1] tracking-[-0.03em] text-paper md:text-[2.5rem]">See how Warden is verified.</h2>
              <p className="mt-4 font-sans text-base leading-[1.75] text-muted">The project publishes compatibility fixtures, sandbox proof harnesses, and platform-specific limitations. Verification maturity differs by operating system.</p>
            </div>
            <div className="grid gap-3 sm:grid-cols-3 lg:w-[34rem]">
              {[["18", "MCP fixtures"], ["14", "pass"], ["4", "backend paths"]].map(([value, label]) => (
                <div key={label} className="rounded-lg border border-ink-700 bg-ink-950 p-4 text-center">
                  <p className="font-sans text-2xl font-semibold text-blueprint">{value}</p>
                  <p className="mt-1 font-sans text-xs uppercase tracking-[0.1em] text-muted">{label}</p>
                </div>
              ))}
            </div>
          </div>
          <Link href="/docs/proof" className="mt-7 inline-flex font-sans text-sm font-semibold text-blueprint hover:text-paper">Read the verification notes →</Link>
        </div>
      </section>
      <Cta />
      <Footer />
    </main>
  );
}
