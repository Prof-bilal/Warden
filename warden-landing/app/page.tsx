import type { Metadata } from "next";
import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import Problem from "@/components/Problem";
import HowItWorks from "@/components/HowItWorks";
import Backends from "@/components/Backends";
import Proof from "@/components/Proof";
import Demo from "@/components/Demo";
import Capabilities from "@/components/Capabilities";
import Testimonials from "@/components/Testimonials";
import Windows from "@/components/Windows";
import Compatibility from "@/components/Compatibility";
import Cta from "@/components/Cta";
import Footer from "@/components/Footer";

const SITE_URL = "https://warden-six-rouge.vercel.app";

export const metadata: Metadata = {
  title: "Warden — A Sandbox Runtime for MCP Servers",
  description:
    "Warden runs MCP servers in a restricted sandbox, so a server only ever gets the files, hosts, and environment variables you explicitly grant it. Open-source, fail-closed, audited.",
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    title: "Warden — A Sandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible. Open-source, fail-closed, audited.",
    url: SITE_URL,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden — A Sandbox Runtime for MCP Servers",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden — A Sandbox Runtime for MCP Servers",
    description:
      "Run MCP servers in a restricted sandbox. Only the files, hosts, and env vars you explicitly grant are accessible.",
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
      "A sandbox runtime for MCP servers. Runs third-party code in OS-native sandboxes with deny-by-default filesystem, network, and environment controls.",
    url: SITE_URL,
    downloadUrl: "https://github.com/Prof-bilal/Warden/releases",
    installUrl: "https://www.npmjs.com/package/warden-sandbox-cli",
    softwareVersion: "0.1.13",
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
      <Problem />
      <HowItWorks />
      <Backends />
      <Proof />
      <Demo />
      <Capabilities />
      <Testimonials />
      <Windows />
      <Compatibility />
      <Cta />
      <Footer />
    </main>
  );
}
