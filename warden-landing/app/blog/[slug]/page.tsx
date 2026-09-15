import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import JsonLd from "@/components/JsonLd";
import ArticleMarkdown from "@/components/ArticleMarkdown";
import {
  formatDate,
  getAllArticles,
  getArticleBySlug,
  getAuthorBySlug,
  getCategoryBySlug,
  getRelatedArticles,
} from "@/lib/blog";
import { SITE_URL, createMetadata } from "@/lib/seo";
import { articleSchema, breadcrumbSchema } from "@/lib/schema";

export function generateStaticParams() {
  return getAllArticles().map((article) => ({ slug: article.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const article = getArticleBySlug(slug);
  if (!article) return { title: "Not Found" };

  const author = getAuthorBySlug(article.authorSlug);

  return createMetadata({
    title: article.title,
    description: article.description,
    path: `/blog/${article.slug}`,
    // og/twitter images come from the opengraph-image file convention
    image: null,
    type: "article",
    publishedTime: article.date,
    modifiedTime: article.updated,
    authors: author ? [author.name] : undefined,
  });
}

export default async function ArticlePage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const article = getArticleBySlug(slug);
  if (!article) notFound();

  const category = getCategoryBySlug(article.categorySlug);
  const author = getAuthorBySlug(article.authorSlug);
  const related = getRelatedArticles(article, 2);

  const breadcrumbs = breadcrumbSchema([
    { name: "Home", url: `${SITE_URL}/` },
    { name: "Blog", url: `${SITE_URL}/blog` },
    ...(category
      ? [
          {
            name: category.name,
            url: `${SITE_URL}/blog/category/${category.slug}`,
          },
        ]
      : []),
    { name: article.title, url: `${SITE_URL}/blog/${article.slug}` },
  ]);

  return (
    <main className="min-h-screen bg-ink-950">
      <JsonLd data={articleSchema(article, author)} />
      <JsonLd data={breadcrumbs} />
      <Nav />
      <article className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <nav
          aria-label="Breadcrumb"
          className="mb-8 font-hero text-[0.8125rem] text-muted"
        >
          <Link href="/" className="transition-colors hover:text-paper">
            Home
          </Link>
          <span className="mx-1.5 text-ink-600">/</span>
          <Link href="/blog" className="transition-colors hover:text-paper">
            Blog
          </Link>
          {category && (
            <>
              <span className="mx-1.5 text-ink-600">/</span>
              <Link
                href={`/blog/category/${category.slug}`}
                className="transition-colors hover:text-paper"
              >
                {category.name}
              </Link>
            </>
          )}
        </nav>

        <h1 className="font-hero text-[2.5rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper">
          {article.title}
        </h1>

        <div className="mt-5 flex flex-wrap items-center gap-x-3 gap-y-1 font-hero text-[0.875rem] text-muted">
          {author && (
            <Link
              href={`/author/${author.slug}`}
              className="text-paper transition-colors hover:text-blueprint"
            >
              {author.name}
            </Link>
          )}
          <time dateTime={article.date}>
            Published {formatDate(article.date)}
          </time>
          {article.updated && (
            <time dateTime={article.updated}>
              Updated {formatDate(article.updated)}
            </time>
          )}
        </div>

        {article.image && (
          <Image
            src={article.image}
            alt={article.title}
            width={1376}
            height={768}
            className="mt-10 rounded-xl border border-ink-700"
          />
        )}

        {article.sample && (
          <p className="mt-6 rounded-lg border border-progress/30 bg-progress-subtle px-4 py-3 font-hero text-[0.875rem] text-paper">
            Sample contentthis article demonstrates the publishing system.
            Replace it with a real article before promoting it.
          </p>
        )}

        <div className="mt-10">
          <ArticleMarkdown content={article.body} />
        </div>

        {author && (
          <div className="mt-14 border-t border-ink-800 pt-6">
            <p className="font-hero text-[0.9375rem] text-muted">
              Written by{" "}
              <Link
                href={`/author/${author.slug}`}
                className="text-paper transition-colors hover:text-blueprint"
              >
                {author.name}
              </Link>{" "}
             {author.role}.
            </p>
          </div>
        )}

        {related.length > 0 && (
          <section className="mt-14 border-t border-ink-800 pt-8">
            <h2 className="font-hero text-[0.6875rem] font-bold uppercase tracking-[0.12em] text-muted">
              Related articles
            </h2>
            <ul className="mt-4 space-y-3">
              {related.map((relatedArticle) => (
                <li key={relatedArticle.slug}>
                  <Link
                    href={`/blog/${relatedArticle.slug}`}
                    className="font-hero text-[1.0625rem] font-medium text-paper transition-colors hover:text-blueprint"
                  >
                    {relatedArticle.title}
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        )}
      </article>
      <Footer />
    </main>
  );
}
