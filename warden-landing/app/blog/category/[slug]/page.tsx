import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import ArticleCard from "@/components/ArticleCard";
import {
  CATEGORIES,
  getArticlesByCategory,
  getAuthorBySlug,
  getCategoryBySlug,
} from "@/lib/blog";
import { SITE_URL, createMetadata } from "@/lib/seo";
import { breadcrumbSchema, collectionPageSchema } from "@/lib/schema";

export function generateStaticParams() {
  return CATEGORIES.map((category) => ({ slug: category.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const category = getCategoryBySlug(slug);
  if (!category) return { title: "Not Found" };

  return createMetadata({
    title: category.name,
    description: `Explore Warden's articles about ${category.description.toLowerCase()}`,
    path: `/blog/category/${category.slug}`,
  });
}

export default async function CategoryPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const category = getCategoryBySlug(slug);
  if (!category) notFound();

  const articles = getArticlesByCategory(category.slug);

  const breadcrumbs = breadcrumbSchema([
    { name: "Home", url: `${SITE_URL}/` },
    { name: "Blog", url: `${SITE_URL}/blog` },
    {
      name: category.name,
      url: `${SITE_URL}/blog/category/${category.slug}`,
    },
  ]);

  const collection = collectionPageSchema({
    name: `${category.name} — Warden Blog`,
    description: category.description,
    url: `${SITE_URL}/blog/category/${category.slug}`,
    itemUrls: articles.map((a) => `${SITE_URL}/blog/${a.slug}`),
  });

  return (
    <main className="min-h-screen bg-ink-950">
      <JsonLd data={breadcrumbs} />
      <JsonLd data={collection} />
      <Nav />
      <div className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <nav
          aria-label="Breadcrumb"
          className="mb-8 text-[0.8125rem] text-muted"
        >
          <Link href="/" className="transition-colors hover:text-paper">
            Home
          </Link>
          <span className="mx-1.5 text-ink-600">/</span>
          <Link href="/blog" className="transition-colors hover:text-paper">
            Blog
          </Link>
          <span className="mx-1.5 text-ink-600">/</span>
          <span className="text-paper">{category.name}</span>
        </nav>

        <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          {category.name}
        </h1>
        <p className="mt-4 text-[1.0625rem] leading-relaxed text-muted">
          {category.description}
        </p>

        <div className="mt-12 border-t border-ink-800">
          {articles.length === 0 ? (
            <p className="py-12 text-muted">
              No articles in this category yet.
            </p>
          ) : (
            articles.map((article) => (
              <ArticleCard
                key={article.slug}
                article={article}
                category={category}
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
