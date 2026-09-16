import type { Metadata } from "next";
import fs from "fs";
import path from "path";
import { SITE_URL } from "@/lib/seo";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema } from "@/lib/schema";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import DocsSidebar from "@/components/DocsSidebar";

export const metadata: Metadata = {
  title: "Documentation",
  description:
    "Warden documentation: install, quickstart, policy schema, CLI reference, compatibility matrix, security review, and more.",
  alternates: {
    canonical: `${SITE_URL}/docs`,
  },
  openGraph: {
    title: "Documentation",
    description:
      "Install, quickstart, policy schema, CLI reference, compatibility matrix, and security review for Warden.",
    url: `${SITE_URL}/docs`,
    images: [
      {
        url: `${SITE_URL}/og-image.png`,
        width: 1200,
        height: 630,
        alt: "Warden Documentation",
      },
    ],
  },
  twitter: {
    card: "summary_large_image",
    title: "Documentation",
    description:
      "Install, quickstart, policy schema, CLI reference, compatibility matrix, and security review for Warden.",
    images: [`${SITE_URL}/og-image.png`],
  },
};

const CARDS = [
  {
    title: "Install",
    body: "Linux, macOS, Windows, Docker fallbackor build from source with Go 1.22+. What each platform needs before warden will run.",
    href: "/docs/install",
  },
  {
    title: "Quickstart",
    body: "Sandbox your first MCP server in five minutes: copy a fixture policy, run it, then trace-and-generate for your own servers.",
    href: "/docs/quickstart",
  },
  {
    title: "Policy schema",
    body: "Every policy.yaml field, validation rule, and enforcement notethe one reference to keep open while writing policies.",
    href: "/docs/schema",
  },
  {
    title: "CLI reference",
    body: "run, trace, init, logs, gateway, --approveevery subcommand and flag, verified against the source.",
    href: "/docs/cli",
  },
  {
    title: "Compatibility matrix",
    body: "18 tested servers with exact policies: 14 pass, 2 conditional, 2 fail with classified reasons.",
    href: "/docs/compatibility",
  },
  {
    title: "FAQ",
    body: "Missing backends, bare npx commands, blocked access, the HOME footgun, and what Warden can't do.",
    href: "/docs/faq",
  },
  {
    title: "Architecture",
    body: "How the CLI, policy engine, sandbox backends, egress proxy, and audit logger fit togetherwith component diagrams.",
    href: "/docs/architecture",
  },
  {
    title: "Example policies",
    body: "Copy-paste policies for filesystem, GitHub, Slack, PostgreSQL, Brave Search, and a comprehensive reference template.",
    href: "/docs/examples",
  },
  {
    title: "Security review",
    body: "Threat model, known limitations, credential exposure risks, and best practices before relying on any policy in production.",
    href: "/docs/security",
  },
  {
    title: "Roadmap",
    body: "M0 through M8 milestoneswhat's built, what's shipped, and what's next for Warden.",
    href: "/docs/roadmap",
  },
  {
    title: "Contributing",
    body: "Development setup, pull request guidance, and the three highest-impact contribution areas right now.",
    href: "/docs/contributing",
  },
  {
    title: "Testing guide",
    body: "Unit tests, integration tests, escape tests, fixtures, and CI expectations for the project.",
    href: "/docs/testing",
  },
  {
    title: "Cross-platform testing",
    body: "Test Warden on Linux, macOS, and Windowsbuild binaries, run MCP servers, verify security boundaries.",
    href: "/docs/testing-platforms",
  },
];

const DOCS_DIR = path.join(process.cwd(), "content", "docs");

// Docs that are not featured in CARDS above still get linked in a
// "More documentation" section so every docs page is reachable from
// the hub (no orphan pages).
const CURATED_HREFS = new Set(CARDS.map((c) => c.href));

const MORE_ORDER = [
  "about",
  "how-to-use",
  "approve",
  "gateway",
  "client-proxy",
  "container-k8s",
  "proof",
  "deploy",
  "prd",
  "mvp",
  "beta",
  "design",
  "reviewing",
  "codestyle",
  "license",
];

function getDocTitle(slug: string): string {
  try {
    const raw = fs.readFileSync(path.join(DOCS_DIR, `${slug}.md`), "utf-8");
    const h1 = raw.split("\n").find((l) => l.startsWith("# "));
    if (h1) return h1.replace(/^#\s*/, "").trim();
  } catch {
    // fall through to humanized slug
  }
  return slug.replace(/-/g, " ");
}

function getMoreDocs(): { slug: string; title: string }[] {
  let slugs: string[] = [];
  try {
    slugs = fs
      .readdirSync(DOCS_DIR)
      .filter((f) => f.endsWith(".md"))
      .map((f) => f.replace(/\.md$/, ""))
      .filter((s) => s !== "index" && !CURATED_HREFS.has(`/docs/${s}`));
  } catch {
    return [];
  }
  return slugs
    .sort((a, b) => {
      const ai = MORE_ORDER.indexOf(a);
      const bi = MORE_ORDER.indexOf(b);
      return (
        (ai === -1 ? MORE_ORDER.length : ai) -
          (bi === -1 ? MORE_ORDER.length : bi) || a.localeCompare(b)
      );
    })
    .map((slug) => ({ slug, title: getDocTitle(slug) }));
}

const MORE_DOCS = getMoreDocs();

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
        {/* Breadcrumbicons.devigner.cc style */}
        <nav className="mb-6 text-[0.875rem] text-muted" aria-label="Breadcrumb">
          <span className="hover:text-paper transition-colors">
            <a href="/">Warden</a>
          </span>
          <span className="mx-1.5 text-ink-600">/</span>
          <span className="text-paper">Docs</span>
        </nav>

        {/* Sidebar + Content layout */}
        <div className="flex gap-12">
          {/* Sidebar */}
          <aside className="hidden w-56 shrink-0 lg:block">
            <div className="sticky top-24">
              <DocsSidebar />
            </div>
          </aside>

          {/* Main content */}
          <article className="min-w-0 flex-1">
            <h1 className="font-hero text-[2.5rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper">
              Docs.
            </h1>
            <p className="mt-5 max-w-[34rem] font-hero text-[1.0625rem] leading-[1.65] text-muted">
              Install it, run your first sandboxed server, then go deep on the
              policy schema. Start with Install and Quickstartin that order.
            </p>

            <div className="mt-14 grid gap-px overflow-hidden rounded-sm border border-ink-800 bg-ink-800 md:grid-cols-2">
              {CARDS.map((c) => (
                <a
                  key={c.title}
                  href={c.href}
                  className="group bg-ink-950 p-7 transition-colors hover:bg-ink-900"
                >
                  <h2 className="font-hero text-[1.125rem] font-bold text-paper">
                    {c.title}
                    <span className="ml-2 inline-block text-blueprint transition-transform group-hover:translate-x-0.5">
                      →
                    </span>
                  </h2>
                  <p className="mt-2 font-hero text-[0.9375rem] leading-[1.6] text-muted">{c.body}</p>
                </a>
              ))}
            </div>
          </article>
        </div>

        {/* More documentation */}
        {MORE_DOCS.length > 0 && (
          <section className="pb-4">
            <h2 className="font-hero text-[0.6875rem] font-bold uppercase tracking-[0.12em] text-muted">
              More documentation
            </h2>
            <ul className="mt-4 grid gap-x-8 gap-y-2 sm:grid-cols-2">
              {MORE_DOCS.map((doc) => (
                <li key={doc.slug}>
                  <a
                    href={`/docs/${doc.slug}`}
                    className="font-hero text-[0.9375rem] text-muted transition-colors hover:text-paper"
                  >
                    {doc.title}
                  </a>
                </li>
              ))}
            </ul>
          </section>
        )}
      </div>
      <Footer />
    </main>
  );
}
