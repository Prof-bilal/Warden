import type { Metadata } from "next";
import Link from "next/link";
import { SITE_URL } from "@/lib/seo";
import { getAllArticles } from "@/lib/blog";
import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import BoundaryDemo from "@/components/BoundaryDemo";
import HowItWorks from "@/components/HowItWorks";
import Backends from "@/components/Backends";
import Proof from "@/components/Proof";
import Demo from "@/components/Demo";
import HowToUse from "@/components/HowToUse";
import Capabilities from "@/components/Capabilities";
import Windows from "@/components/Windows";
import Compatibility from "@/components/Compatibility";
import Cta from "@/components/Cta";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";

export const metadata: Metadata = {
  title: { absolute: "Give MCP servers a sandbox, not your filesystem." },
  description:
    "Warden runs MCP servers in a restricted sandbox. Servers only get the files, hosts, and env vars you explicitly grant. Open-source, fail-closed, audited.",
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    title: "Give MCP servers a sandbox, not your filesystem.",
    description:
      "Warden runs MCP servers in a restricted sandbox. Servers only get the files, hosts, and env vars you explicitly grant. Open-source, fail-closed, audited.",
    url: SITE_URL,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Give MCP servers a sandbox, not your filesystem.",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Give MCP servers a sandbox, not your filesystem.",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

export default function Home() {
  // Latest articles surfaced on the homepage so every post is within
  // one internal link of the site's strongest page.
  const latestArticles = getAllArticles().slice(0, 3);

  const softwareSchema = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "Warden",
    applicationCategory: "DeveloperApplication",
    operatingSystem: ["Linux", "macOS", "Windows"],
    description:
      "A sandbox runtime for MCP servers. Runs third-party code in OS-native sandboxes with deny-by-default filesystem, network, and environment controls.",
    "@id": `${SITE_URL}/#software`,
    url: SITE_URL,
    isAccessibleForFree: true,
    downloadUrl: "https://github.com/Prof-bilal/Warden/releases",
    installUrl: "https://www.npmjs.com/package/warden-sandbox-cli",
    softwareVersion: "0.1.17",
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
          name: "Give MCP servers a sandbox, not your filesystem.",
          description:
            "Warden runs MCP servers in a restricted sandbox. Servers only get the files, hosts, and env vars you explicitly grant. Open-source, fail-closed, audited.",
          isPartOf: { "@type": "WebSite", name: "Warden", url: SITE_URL },
          mainEntity: { "@id": `${SITE_URL}/#software` },
        }}
      />
      <Nav />
      <Hero />
      <section className="border-y border-ink-700 bg-ink-900/60">
        <div className="mx-auto grid max-w-content gap-8 px-6 py-12 md:grid-cols-[1fr_auto] md:items-center">
          <div><p className="text-xs uppercase tracking-widest text-blueprint">MCP setup workspace</p><h2 className="mt-3 text-2xl font-semibold">Choose your client. Review your server’s access.</h2><p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted">Explore six candidate policy profiles and reversible setup for Claude, Cursor, Codex, VS Code, Gemini, and other local MCP hosts.</p></div>
          <div className="flex flex-wrap gap-3"><Link href="/setup" className="rounded-lg bg-blueprint px-5 py-3 text-sm font-semibold text-ink-950">Start setup →</Link><Link href="/policies" className="rounded-lg border border-ink-600 px-5 py-3 text-sm text-paper">Browse profiles</Link></div>
        </div>
      </section>
      <BoundaryDemo />
      <HowItWorks />
      <Backends />
      <Proof />
      <Demo />
      <HowToUse />
      <Capabilities />
      <Windows />
      <Compatibility />
      {latestArticles.length > 0 && (
        <section className="border-t border-ink-800">
          <div className="mx-auto max-w-content px-6 py-20">
            <div className="flex flex-wrap items-baseline justify-between gap-4">
              <h2 className="font-hero text-[1.75rem] font-bold leading-[1.2] tracking-[-0.02em] text-paper">
                Latest from the blog
              </h2>
              <Link
                href="/blog"
                className="font-hero text-[0.9375rem] text-muted transition-colors hover:text-paper"
              >
                View all articles →
              </Link>
            </div>
            <ul className="mt-10 grid gap-4 md:grid-cols-3">
              {latestArticles.map((article) => (
                <li key={article.slug}>
                  <Link
                    href={`/blog/${article.slug}`}
                    className="group flex h-full flex-col rounded-2xl border border-ink-700 bg-ink-900 p-6 transition-colors hover:border-blueprint/60"
                  >
                    <h3 className="font-hero text-[1.0625rem] font-bold leading-snug text-paper">
                      {article.title}
                    </h3>
                    <p className="mt-2 flex-1 font-hero text-[0.875rem] leading-[1.6] text-muted">
                      {article.description}
                    </p>
                    <span className="mt-4 font-hero text-[0.8125rem] text-blueprint transition-colors group-hover:text-paper">
                      Read article →
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        </section>
      )}
      <Cta />
      <Footer />
    </main>
  );
}
