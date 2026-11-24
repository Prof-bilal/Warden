import type { Metadata } from "next";
import Link from "next/link";
import { SITE_URL } from "@/lib/seo";
import Hero from "@/components/Hero";
import BoundaryExample from "@/components/BoundaryExample";
import AgentWorkspace from "@/components/AgentWorkspace";
import RunAnywhere from "@/components/RunAnywhere";
import JsonLd from "@/components/JsonLd";

import "./home.css";

const description =
  "Use Warden to review npm and PyPI releases, sandbox local AI tools and MCP servers, and block changes to pinned MCP tool definitions.";

export const metadata: Metadata = {
  title: { absolute: "Warden | Permissions for Local AI Tools" },
  description,
  alternates: {
    canonical: SITE_URL,
  },
  openGraph: {
    title: "Warden | Permissions for Local AI Tools",
    description,
    url: SITE_URL,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden | Permissions for Local AI Tools",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Warden | Permissions for Local AI Tools",
    description,
    images: [`${SITE_URL}/og-image.png`],
  },
};

const workflows = [
  {
    title: "Check package releases",
    body: "Install Guard checks npm and PyPI releases before installation. Its default 48-hour cooldown gives new releases time to be reviewed.",
    href: "#install-guard",
    link: "See the npm workflow",
  },
  {
    title: "Run a reviewed MCP server",
    body: "The bundled Filesystem pack selects an exact release and gives the server read-only access to a chosen data directory. Start without writing YAML.",
    href: "/docs/zero-config",
    link: "Run the Filesystem quickstart",
  },
  {
    title: "Pin MCP tool definitions",
    body: "Enable pinning on the bridge to record each allowed tool’s first observed definition. A change hides the tool, denies calls, and leaves a journal entry for review.",
    href: "/docs/mcp-tool-pins",
    link: "Set up tool pins",
  },
];

const guardCommand = "warden guard npm install is-number@7.0.0 --output ./reviewed-npm --strict";

const permissions = [
  ["Filesystem", "Share specific paths for reading or writing and keep the rest of your files private."],
  ["Network", "Allow the destinations a tool needs and block connections to other hosts."],
  ["Environment", "Pass selected environment variables without exposing the whole shell."],
  ["Runtime limits", "Set resource limits and timeouts for sandboxed processes."],
];

const proofStats = [
  ["18", "MCP fixtures"],
  ["8/8", "Linux proof checks"],
  ["7/7", "Attack simulations contained"],
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
    <main id="main-content" className="home home-shell">
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
          name: "Warden | Permissions for Local AI Tools",
          description,
          isPartOf: { "@type": "WebSite", name: "Warden", url: SITE_URL },
          mainEntity: { "@id": `${SITE_URL}/#software` },
        }}
      />

      <Hero />

      <section id="warden-2" className="home-section" aria-labelledby="warden-2-title">
        <h2 id="warden-2-title" className="accent-heading">What’s new in Warden 2.0</h2>
        <p className="home-intro">
          Check the package you install, control what the server can access, and review
          changes to the tools it exposes.
        </p>
        <div className="home-grid home-workflows">
          {workflows.map(({ title, body, href, link }) => (
            <article key={href}>
              <h3>{title}</h3>
              <p className="mt-2">{body}</p>
              <Link className="home-section-link" href={href}>
                {link} <span aria-hidden="true">→</span>
              </Link>
            </article>
          ))}
        </div>
        <div className="home-actions">
          <Link href="/v2">Explore the Warden 2.0 preview <span aria-hidden="true">→</span></Link>
          <Link href="#autopilot">See the Autopilot benchmark <span aria-hidden="true">→</span></Link>
          <Link href="/docs/migration">Preview your migration <span aria-hidden="true">→</span></Link>
        </div>
      </section>

      <section id="install-guard" className="home-section home-workspace" aria-labelledby="install-guard-title">
        <div>
          <h2 id="install-guard-title" className="accent-heading">Review npm packages before installing</h2>
          <p className="home-intro">
            Install Guard checks the release age and looks for install hooks and native build
            metadata. With <code>--strict</code>, missing age metadata blocks the install too.
          </p>
          <p className="home-intro">
            Dependency scripts stay disabled. An exact <code>name@version</code> allowlist entry
            can authorize a reviewed package’s hooks in a separate sandbox with network access denied.
          </p>
          <Link className="home-section-link" href="/docs/install-guard">
            Read the Install Guard guide <span aria-hidden="true">→</span>
          </Link>
        </div>
        <div className="home-terminal">
          <div className="home-terminal-label">
            <span>Install Guard</span>
            <span>Strict npm example</span>
          </div>
          <pre><code>{guardCommand}</code></pre>
          <div className="home-terminal-command">
            <p className="mb-2 text-muted">Use a new output directory and a build that includes Install Guard.</p>
            <Link href="/videos/warden-install-guard.mp4">
              Watch a two-hour-old release get blocked <span aria-hidden="true">→</span>
            </Link>
          </div>
        </div>
      </section>

      <section id="autopilot" className="home-section" aria-labelledby="autopilot-title">
        <h2 id="autopilot-title" className="accent-heading">Warden Autopilot preview</h2>
        <p className="home-intro">
          The Linux development preview detects project markers and installed tools.
          Approve an installed program for a project and Warden generates a read-only,
          offline policy without a YAML file. You choose when to run it and which arguments
          to pass. Edit mode previews changes and asks again before applying them to the
          original project.
        </p>
        <dl className="home-grid home-metrics">
          <div><dt>Onboarding and native checks, each</dt><dd>5/5</dd></div>
          <div><dt>Project detection</dt><dd>6.7 ms</dd></div>
          <div><dt>Protected file read</dt><dd>79.5 ms</dd></div>
        </dl>
        <p className="mt-4 text-sm text-muted">
          Tested with a development build on Linux amd64 with a Ryzen 5 7430U on
          November 24, 2026. Timings are medians of 10 fixture runs. The same read took
          0.9 ms directly. Discovery ran no project scripts.
        </p>
        <p className="home-intro">
          Claude Code and the direct Codex binary passed offline version checks. The
          Codex and OpenCode shell wrappers failed to launch, and the direct OpenCode
          binary could not resolve a home directory. Cloud-backed coding sessions were
          outside the benchmark.
        </p>
      </section>

      <section id="how-it-works" className="home-section">
        <h2 className="accent-heading">Define what tools can access</h2>
        <p className="home-intro">
          Local agents and MCP servers inherit the permissions of the user who starts them.
          With Warden, you declare access in a policy and review decisions in the audit log.
        </p>
        <dl className="home-grid home-permissions">
          {permissions.map(([name, detail]) => (
            <div key={name}>
              <dt>{name}</dt>
              <dd>{detail}</dd>
            </div>
          ))}
        </dl>
        <Link className="home-section-link" href="/docs/quickstart">
          Walk through your first policy <span aria-hidden="true">→</span>
        </Link>
      </section>

      <BoundaryExample />
      <AgentWorkspace />

      <section className="home-section">
        <h2 className="accent-heading">See how Warden is verified</h2>
        <p className="home-intro">
          Warden publishes compatibility fixtures, sandbox proof harnesses, and platform
          limitations. Verification maturity varies by operating system.
        </p>
        <dl className="home-grid home-metrics">
          {proofStats.map(([value, label]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
        <Link className="home-section-link" href="/docs/proof">
          Read the verification notes <span aria-hidden="true">→</span>
        </Link>
      </section>

      <RunAnywhere />

      <section id="how-to-use" className="home-section">
        <h2 className="accent-heading">Start with one local tool</h2>
        <p className="home-intro">
          Choose your client, install Warden, and give your tool only the access it needs.
        </p>
        <Link className="home-section-link" href="/setup">
          Find your setup guide <span aria-hidden="true">→</span>
        </Link>
      </section>
    </main>
  );
}
