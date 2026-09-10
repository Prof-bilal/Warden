import type { Metadata } from "next";
import fs from "fs";
import path from "path";
import { notFound } from "next/navigation";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import MarkdownContent from "@/components/MarkdownContent";
import DocsSidebar from "@/components/DocsSidebar";

const SITE_URL = "https://warden-six-rouge.vercel.app";

const DOCS_DIR = path.resolve(
  process.cwd(),
  "../warden-starter/warden/docs"
);

function getDocsDir() {
  return DOCS_DIR;
}

function readDoc(slug: string): string | null {
  const filePath = path.join(getDocsDir(), `${slug}.md`);
  try {
    return fs.readFileSync(filePath, "utf-8");
  } catch {
    return null;
  }
}

function getAllDocSlugs(): string[] {
  try {
    return fs
      .readdirSync(getDocsDir())
      .filter((f) => f.endsWith(".md"))
      .map((f) => f.replace(/\.md$/, ""));
  } catch {
    return [];
  }
}

export function generateStaticParams() {
  return getAllDocSlugs().map((slug) => ({ slug }));
}

export function generateMetadata({ params }: { params: { slug: string } }): Metadata {
  const content = readDoc(params.slug);
  if (!content) return { title: "Not Found" };

  const firstLine = content.split("\n").find((l) => l.startsWith("# "));
  const title = firstLine
    ? firstLine.replace(/^#\s*/, "")
    : params.slug.replace(/-/g, " ");

  const description = `Warden documentation: ${title}. Learn how to use Warden to sandbox MCP servers.`;

  return {
    title,
    description,
    alternates: {
      canonical: `${SITE_URL}/docs/${params.slug}`,
    },
    openGraph: {
      title: `${title} — Warden`,
      description,
      url: `${SITE_URL}/docs/${params.slug}`,
      type: "article",
      images: [
        {
          url: `${SITE_URL}/og-image.png`,
          width: 1200,
          height: 630,
          alt: `${title} — Warden Documentation`,
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: `${title} — Warden`,
      description,
      images: [`${SITE_URL}/og-image.png`],
    },
  };
}

const PUBLIC_SLUGS = [
  "install",
  "quickstart",
  "schema",
  "cli",
  "compatibility",
  "faq",
  "architecture",
  "examples",
  "security",
  "roadmap",
  "contributing",
  "testing",
  "testing-platforms",
];

const slugToLabel: Record<string, string> = {
  install: "Installation",
  quickstart: "Quickstart",
  schema: "Policy schema",
  cli: "CLI reference",
  compatibility: "Compatibility",
  faq: "FAQ",
  architecture: "Architecture",
  examples: "Example policies",
  security: "Security review",
  roadmap: "Roadmap",
  contributing: "Contributing",
  testing: "Testing guide",
  "testing-platforms": "Cross-platform testing",
};

export default function DocPage({ params }: { params: { slug: string } }) {
  const content = readDoc(params.slug);
  if (!content) notFound();

  const firstLine = content.split("\n").find((l) => l.startsWith("# "));
  const title = firstLine
    ? firstLine.replace(/^#\s*/, "")
    : params.slug.replace(/-/g, " ");

  const slugs = PUBLIC_SLUGS;
  const allSlugs = getAllDocSlugs();
  const currentIndex = allSlugs.indexOf(params.slug);
  const prevSlug = currentIndex > 0 ? allSlugs[currentIndex - 1] : null;
  const nextSlug =
    currentIndex < allSlugs.length - 1 ? allSlugs[currentIndex + 1] : null;

  const slugToTitle = (s: string) => {
    if (slugToLabel[s]) return slugToLabel[s];
    const c = readDoc(s);
    if (!c) return s.replace(/-/g, " ");
    const h1 = c.split("\n").find((l) => l.startsWith("# "));
    return h1 ? h1.replace(/^#\s*/, "") : s.replace(/-/g, " ");
  };

  const articleSchema = {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    headline: title,
    description: `Warden documentation: ${title}`,
    url: `${SITE_URL}/docs/${params.slug}`,
    author: {
      "@type": "Person",
      name: "Prof-bilal",
      url: "https://github.com/Prof-bilal",
    },
    publisher: {
      "@type": "Organization",
      name: "Warden",
      logo: {
        "@type": "ImageObject",
        url: `${SITE_URL}/icon.png`,
      },
    },
    mainEntityOfPage: {
      "@type": "WebPage",
      "@id": `${SITE_URL}/docs/${params.slug}`,
    },
  };

  return (
    <main className="min-h-screen bg-ink-950">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleSchema) }}
      />
      <Nav />
      <div className="mx-auto max-w-content px-6 pb-20 pt-8 md:pt-12">
        {/* Breadcrumb — icons.devigner.cc style */}
        <nav className="mb-6 text-[0.875rem] text-muted" aria-label="Breadcrumb">
          <span className="hover:text-paper transition-colors">
            <a href="/">Warden</a>
          </span>
          <span className="mx-1.5 text-ink-600">/</span>
          <span className="hover:text-paper transition-colors">
            <a href="/docs">Docs</a>
          </span>
          <span className="mx-1.5 text-ink-600">/</span>
          <span className="text-paper">{slugToTitle(params.slug)}</span>
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
            <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
              {title}
            </h1>

            <div className="mt-8">
              <MarkdownContent content={content} />
            </div>

            {/* Prev / Next navigation */}
            <div className="mt-16 flex items-center justify-between border-t border-ink-700 pt-8">
              {prevSlug ? (
                <a
                  href={`/docs/${prevSlug}`}
                  className="text-[0.875rem] text-muted transition-colors hover:text-paper"
                >
                  ← {slugToTitle(prevSlug)}
                </a>
              ) : (
                <span />
              )}
              {nextSlug ? (
                <a
                  href={`/docs/${nextSlug}`}
                  className="text-[0.875rem] text-muted transition-colors hover:text-paper"
                >
                  {slugToTitle(nextSlug)} →
                </a>
              ) : (
                <span />
              )}
            </div>


          </article>
        </div>
      </div>
      <Footer />
    </main>
  );
}
