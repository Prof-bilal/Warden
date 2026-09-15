import type { Metadata } from "next";
import Link from "next/link";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import ArticleCard from "@/components/ArticleCard";
import { getAllArticles, getAuthorBySlug, getCategoryBySlug, CATEGORIES } from "@/lib/blog";
import { SITE_URL, createMetadata } from "@/lib/seo";
import { breadcrumbSchema, collectionPageSchema } from "@/lib/schema";

export const metadata: Metadata = createMetadata({
  title: "Blog",
  description:
    "Articles from the Warden team on sandboxing MCP servers, policy enforcement, OS-level security, and running AI tooling safely.",
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd data={breadcrumbs} />
      <JsonLd data={collection} />
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <p className="font-hero text-[0.6875rem] font-bold uppercase tracking-[0.12em] text-grant">
          Blog
        </p>
        <h1 className="mt-3 font-hero text-[2.5rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper">
          Warden articles
        </h1>
        <p className="mt-4 font-hero text-[1.0625rem] leading-relaxed text-muted">
          Sandboxing, policy enforcement, and platform internals for MCP
          servers and AI tooling.
        </p>

        <nav
          aria-label="Blog categories"
          className="mt-8 flex flex-wrap gap-2"
        >
          {CATEGORIES.map((category) => (
            <Link
              key={category.slug}
              href={`/blog/category/${category.slug}`}
              className="rounded-full border border-ink-700 px-3 py-1 font-hero text-[0.8125rem] text-muted transition-colors hover:border-blueprint/50 hover:text-paper"
            >
              {category.name}
            </Link>
          ))}
        </nav>

        <div className="mt-12 border-t border-ink-800">
          {articles.length === 0 ? (
            <p className="py-12 font-hero text-muted">
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
      <Footer />
    </main>
  );
}
