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

export const metadata: Metadata = {
  title: { absolute: "WardenSandbox Runtime for MCP Servers" },
  description:
    "Warden runs MCP servers in a restricted sandbox. Servers only get the files, hosts, and env vars you explicitly grant. Open-source, fail-closed, audited.",
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    title: "WardenSandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible. Open-source, fail-closed, audited.",
    url: SITE_URL,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "WardenA Sandbox Runtime for MCP Servers",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "WardenSandbox Runtime for MCP Servers",
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
    url: SITE_URL,
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
      name: "Prof-bilal",
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
      <Nav />
      <Hero />
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
