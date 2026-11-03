import type { Metadata } from "next";
import { SITE_URL } from "@/lib/seo";
import fs from "fs";
import path from "path";
import { notFound } from "next/navigation";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import MarkdownContent from "@/components/MarkdownContent";
import DocsSidebar from "@/components/DocsSidebar";
import JsonLd from "@/components/JsonLd";
import { breadcrumbSchema, faqPageSchema } from "@/lib/schema";
import {
  docSocialTitle,
  docTitle,
  extractDocDescription,
  extractDocHeadline,
  extractFaqItems,
} from "@/lib/doc-seo";

const DOCS_DIR = path.resolve(
  process.cwd(),
  "content/docs"
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

function MobileDocsNav() {
  return (
    <details className="border border-hair bg-cell p-4 lg:hidden">
      <summary className="cursor-pointer list-none font-mono text-[12px] uppercase tracking-micro text-offwhite/70">
        Browse documentation
      </summary>
      <div className="mt-5">
        <DocsSidebar />
      </div>
    </details>
  );
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const content = readDoc(slug);
  if (!content) return { title: "Not Found" };

  const title = extractDocHeadline(content, slug.replace(/-/g, " "));
  const description = extractDocDescription(content, title);
  const socialTitle = docSocialTitle(title);

  return {
    title: docTitle(title),
    description,
    alternates: {
      canonical: `${SITE_URL}/docs/${slug}`,
    },
    openGraph: {
      title: socialTitle,
      description,
      url: `${SITE_URL}/docs/${slug}`,
      type: "article",
      images: [
        {
          url: `${SITE_URL}/og-image.png`,
          width: 1200,
          height: 630,
          alt: socialTitle,
        },
      ],
    },
    twitter: {
      card: "summary_large_image",
      title: socialTitle,
      description,
      images: [`${SITE_URL}/og-image.png`],
    },
  };
}

export default async function DocPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const content = readDoc(slug);
  if (!content) notFound();

  const title = extractDocHeadline(content, slug.replace(/-/g, " "));
  const description = extractDocDescription(content, title);
  const faqItems = slug === "faq" ? extractFaqItems(content) : [];

  const allSlugs = getAllDocSlugs();
  const currentIndex = allSlugs.indexOf(slug);
  const prevSlug = currentIndex > 0 ? allSlugs[currentIndex - 1] : null;
  const nextSlug =
    currentIndex < allSlugs.length - 1 ? allSlugs[currentIndex + 1] : null;

  const slugToTitle = (s: string) => {
    const c = readDoc(s);
    if (!c) return s.replace(/-/g, " ");
    return extractDocHeadline(c, s.replace(/-/g, " "));
  };

  const articleSchema = {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    headline: title,
    description,
    url: `${SITE_URL}/docs/${slug}`,
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
      "@id": `${SITE_URL}/docs/${slug}`,
    },
  };

  return (
    <PageShell>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleSchema) }}
      />
      {faqItems.length > 0 && (
        <JsonLd
          data={faqPageSchema(`${SITE_URL}/docs/${slug}`, faqItems)}
        />
      )}
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Docs", url: `${SITE_URL}/docs` },
          { name: title, url: `${SITE_URL}/docs/${slug}` },
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
          <Link href="/docs" className="transition-colors hover:text-offwhite">
            Docs
          </Link>
          <span className="text-offwhite/20">/</span>
          <span className="text-offwhite/70">{slugToTitle(slug)}</span>
        </nav>

        <div className="mb-8 lg:hidden">
          <MobileDocsNav />
        </div>

        <div className="flex gap-12">
          <aside className="hidden w-56 shrink-0 lg:block">
            <div className="sticky top-24">
              <DocsSidebar />
            </div>
          </aside>

          <article className="min-w-0 flex-1">
            <h1 className="font-mono text-[1.9rem] font-semibold leading-[1.08] tracking-[-0.03em] text-offwhite sm:text-[2.5rem]">
              {title}
            </h1>

            <div className="mt-8">
              {/* The page already renders an H1; strip the markdown's own H1 to keep exactly one per page. */}
              <MarkdownContent content={content.replace(/^#[^\n]*\n+/, "")} />
            </div>

            <div className="mt-16 flex items-center justify-between border-t border-hair pt-8">
              {prevSlug ? (
                <Link
                  href={`/docs/${prevSlug}`}
                  className="font-mono text-[11.5px] uppercase tracking-micro text-offwhite/40 transition-colors hover:text-acid"
                >
                  ← {slugToTitle(prevSlug)}
                </Link>
              ) : (
                <span />
              )}
              {nextSlug ? (
                <Link
                  href={`/docs/${nextSlug}`}
                  className="font-mono text-[11.5px] uppercase tracking-micro text-offwhite/40 transition-colors hover:text-acid"
                >
                  {slugToTitle(nextSlug)} →
                </Link>
              ) : (
                <span />
              )}
            </div>
          </article>
        </div>
      </div>
    </PageShell>
  );
}
