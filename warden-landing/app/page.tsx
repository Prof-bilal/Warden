import type { Metadata } from "next";
import Link from "next/link";
import { SITE_URL } from "@/lib/seo";
import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import { PanelRowA, PanelRowB } from "@/components/BlueprintGrid";
import BoundaryExample from "@/components/BoundaryExample";
import AgentWorkspace from "@/components/AgentWorkspace";
import EcosystemFeatures from "@/components/EcosystemFeatures";
import RunAnywhere from "@/components/RunAnywhere";
import Cta from "@/components/Cta";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import SectionMarker from "@/components/bp/SectionMarker";
import Reveal from "@/components/bp/Reveal";
import StatRow from "@/components/bp/StatRow";

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

const problemCards = [
  {
    tag: "Files",
    body: "Keep project access separate from credentials, home directories, and unrelated folders.",
  },
  {
    tag: "Secrets",
    body: "Pass only the environment names a tool actually needs — never the whole shell.",
  },
  {
    tag: "Network",
    body: "Allow specific APIs and block undeclared destinations before a connection opens.",
  },
];

const proofStats: [string, string][] = [
  ["18", "MCP fixtures"],
  ["8/8", "Linux proof checks"],
  ["7/7", "attack simulations"],
];

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
    <main className="relative min-h-screen bg-void text-offwhite">
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

      <div className="bp-grid-bg pointer-events-none absolute inset-0" aria-hidden />
      <div
        className="pointer-events-none absolute inset-x-0 top-0 h-[560px] bg-[radial-gradient(ellipse_60%_100%_at_50%_0%,rgba(94,240,138,0.07),transparent_70%)]"
        aria-hidden
      />
      <Nav />

      <div className="bp-shell relative z-10">
        <Hero />

        <PanelRowA />
        <PanelRowB />

        <section className="border-b border-hair">
          <div className="grid lg:grid-cols-[0.85fr_1.15fr]">
            <div className="border-b border-hair p-6 sm:p-9 lg:border-b-0 lg:border-r">
              <SectionMarker index="02" />
              <h2 className="mt-4 max-w-xl font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
                AI tools run with your permissions.
              </h2>
              <p className="mt-4 max-w-xl font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
                A local agent or MCP server can inherit access to files,
                environment values, and network destinations available to the user
                who started it. Warden turns that implicit access into explicit
                policy.
              </p>
            </div>

            <Reveal className="grid gap-px bg-hair sm:grid-cols-3">
              {problemCards.map((card) => (
                <div key={card.tag} className="bg-cell p-5">
                  <p className="font-mono text-[9px] uppercase tracking-micro text-acid/80">
                    {card.tag}
                  </p>
                  <p className="mt-3 font-sans text-[12px] leading-[1.7] text-offwhite/40">
                    {card.body}
                  </p>
                </div>
              ))}
            </Reveal>
          </div>
        </section>

        <BoundaryExample />
        <AgentWorkspace />

        <section className="border-b border-hair">
          <div className="px-6 py-16 md:py-20">
            <div className="max-w-2xl">
              <SectionMarker index="06" />
              <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
                See how Warden is verified.
              </h2>
              <p className="mt-4 font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
                The project publishes compatibility fixtures, sandbox proof
                harnesses, and platform-specific limitations. Verification maturity
                differs by operating system.
              </p>
            </div>

            <Reveal className="mt-10">
              <StatRow stats={proofStats} className="grid-cols-3 border border-hair sm:grid-cols-3" />
            </Reveal>

            <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <p className="max-w-[560px] font-mono text-[10px] leading-[1.7] text-offwhite/25">
                MCP compatibility fixtures are tracked against the published test
                matrix — the boundary is only as good as what is measured.
              </p>
              <Link
                href="/docs/proof"
                className="shrink-0 font-sans text-[12.5px] font-semibold text-acid transition-colors hover:text-acid-bright"
              >
                Read the verification notes →
              </Link>
            </div>
          </div>
        </section>

        <EcosystemFeatures />
        <RunAnywhere />
        <Cta />
        <Footer />
      </div>
    </main>
  );
}
