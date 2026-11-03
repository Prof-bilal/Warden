import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import PageShell from "@/components/PageShell";
import PageHeader from "@/components/bp/PageHeader";
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
    title: `${category.name} articles`,
    description: category.description,
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
    name: `${category.name} | Warden Blog`,
    description: category.description,
    url: `${SITE_URL}/blog/category/${category.slug}`,
    itemUrls: articles.map((a) => `${SITE_URL}/blog/${a.slug}`),
  });

  return (
    <PageShell>
      <JsonLd data={breadcrumbs} />
      <JsonLd data={collection} />

      <div className="px-6 pt-10 md:pt-14">
        <div className="mx-auto max-w-[48rem]">
          <nav
            aria-label="Breadcrumb"
            className="flex items-center gap-2 font-mono text-[10.5px] uppercase tracking-micro text-offwhite/35"
          >
            <Link href="/" className="transition-colors hover:text-offwhite">
              Home
            </Link>
            <span className="text-offwhite/20">/</span>
            <Link href="/blog" className="transition-colors hover:text-offwhite">
              Blog
            </Link>
            <span className="text-offwhite/20">/</span>
            <span className="text-offwhite/70">{category.name}</span>
          </nav>
        </div>
      </div>

      <PageHeader
        eyebrow="Category"
        title={`${category.name} articles`}
        description={category.description}
      />

      <div className="px-6 py-12 md:py-16">
        <div className="mx-auto max-w-[48rem]">
          <div className="border-t border-hair">
            {articles.length === 0 ? (
              <p className="py-12 font-sans text-offwhite/45">
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
      </div>
    </PageShell>
  );
}
