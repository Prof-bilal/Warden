import type { Metadata } from "next";
import Link from "next/link";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
import JsonLd from "@/components/JsonLd";
import ArticleCard from "@/components/ArticleCard";
import { getAllArticles, getAuthorBySlug, getCategoryBySlug, CATEGORIES } from "@/lib/blog";
import { SITE_URL, createMetadata } from "@/lib/seo";
import { breadcrumbSchema, collectionPageSchema } from "@/lib/schema";

export const metadata: Metadata = createMetadata({
  title: "Articles on sandboxing MCP servers",
  description:
    "Articles on sandboxing MCP servers, policy enforcement, OS-level security, and running AI tooling safely.",
  path: "/blog",
});

export default function BlogIndex() {
  const articles = getAllArticles();

  const breadcrumbs = breadcrumbSchema([
    { name: "Home", url: `${SITE_URL}/` },
    { name: "Blog", url: `${SITE_URL}/blog` },
  ]);

  const collection = collectionPageSchema({
    name: "Warden Blog",
    description:
      "Articles on sandboxing MCP servers, policy enforcement, and OS-level security for AI tooling.",
    url: `${SITE_URL}/blog`,
    itemUrls: articles.map((a) => `${SITE_URL}/blog/${a.slug}`),
  });

  return (
    <PageShell>
      <JsonLd data={breadcrumbs} />
      <JsonLd data={collection} />

      <PageHeader
        eyebrow="Blog"
        title="Articles on sandboxing MCP servers"
        description="Sandboxing, policy enforcement, and platform internals for MCP servers and AI tooling. Warden is the open-source sandbox runtime these articles describe."
      />

      <div className="px-6 py-12 md:py-16">
        <div className="mx-auto max-w-[48rem]">
          <nav aria-label="Topic guides">
            <h2 className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
              Start with a question
            </h2>
            <ul className="mt-4 divide-y divide-hair border-y border-hair">
              {[
                ["/blog/why-mcp-servers-need-sandboxes", "Why an MCP server needs a sandbox"],
                ["/blog/how-warden-enforces-policies", "How a Warden policy becomes an OS sandbox"],
                [
                  "/blog/sandboxing-backends-bubblewrap-seatbelt-appcontainer",
                  "How Linux, macOS, and Windows backends differ",
                ],
                ["/blog/i-tested-18-mcp-servers", "Which of 18 MCP servers can run under Warden"],
                [
                  "/blog/deny-by-default-invisible-not-denied",
                  "Why hiding a path is stronger than permission denied",
                ],
                ["/blog/ai-tried-to-escape-my-sandbox", "What the published escape tests actually showed"],
                ["/guides/mcp-server-security", "How to securely run an MCP server"],
                ["/docs/install", "How to install Warden and run a server"],
              ].map(([href, label]) => (
                <li key={href}>
                  <Link
                    href={href}
                    className="block py-2.5 font-mono text-[14px] text-offwhite/80 transition-colors hover:text-acid"
                  >
                    {label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>

          <nav aria-label="Blog categories" className="mt-8 flex flex-wrap gap-2">
            {CATEGORIES.map((category) => (
              <Link
                key={category.slug}
                href={`/blog/category/${category.slug}`}
                className="border border-hair px-3 py-1 font-mono text-[11px] uppercase tracking-micro text-offwhite/45 transition-colors hover:border-acid/40 hover:text-acid"
              >
                {category.name}
              </Link>
            ))}
          </nav>

          <div className="mt-12 border-t border-hair">
            {articles.length === 0 ? (
              <p className="py-12 font-sans text-offwhite/45">
                No articles published yet.
              </p>
            ) : (
              articles.map((article) => (
                <ArticleCard
                  key={article.slug}
                  article={article}
                  category={getCategoryBySlug(article.categorySlug)}
                  author={getAuthorBySlug(article.authorSlug)}
                />
              ))
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}
