import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import PageShell from "@/components/PageShell";
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
    <PageShell>
      <JsonLd data={articleSchema(article, author)} />
      <JsonLd data={breadcrumbs} />
      <article className="px-6 pb-20 pt-10 md:pt-14">
        <div className="mx-auto max-w-[48rem]">
          <nav
            aria-label="Breadcrumb"
            className="mb-8 flex items-center gap-2 font-mono text-[10.5px] uppercase tracking-micro text-offwhite/35"
          >
            <Link href="/" className="transition-colors hover:text-offwhite">
              Home
            </Link>
            <span className="text-offwhite/20">/</span>
            <Link href="/blog" className="transition-colors hover:text-offwhite">
              Blog
            </Link>
            {category && (
              <>
                <span className="text-offwhite/20">/</span>
                <Link
                  href={`/blog/category/${category.slug}`}
                  className="transition-colors hover:text-offwhite"
                >
                  {category.name}
                </Link>
              </>
            )}
          </nav>

          <h1 className="font-mono text-[1.9rem] font-semibold leading-[1.12] tracking-[-0.02em] text-offwhite sm:text-[2.25rem]">
            {article.title}
          </h1>

          <div className="mt-5 flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[11px] uppercase tracking-micro text-offwhite/35">
            {author && (
              <Link
                href={`/author/${author.slug}`}
                className="text-offwhite/70 transition-colors hover:text-acid"
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

          {article.tldr && article.tldr.length > 0 && (
            <section className="mt-8 border border-hair bg-cell px-5 py-4">
              <h2 className="font-mono text-[9.5px] uppercase tracking-micro text-acid">
                Answer in short
              </h2>
              <ul className="mt-3 list-disc space-y-1.5 pl-5 font-sans text-[14px] leading-[1.65] text-offwhite/50">
                {article.tldr.map((point) => (
                  <li key={point}>{point}</li>
                ))}
              </ul>
            </section>
          )}

          {article.image && (
            <Image
              src={article.image}
              alt={article.title}
              width={1376}
              height={768}
              className="mt-10 border border-hair"
            />
          )}

          <div className="mt-10">
            <ArticleMarkdown content={article.body} />
          </div>

          {author && (
            <div className="mt-14 border-t border-hair pt-6">
              <p className="font-sans text-[14px] leading-[1.7] text-offwhite/45">
                Written by{" "}
                <Link
                  href={`/author/${author.slug}`}
                  className="text-offwhite/80 transition-colors hover:text-acid"
                >
                  {author.name}
                </Link>
                . {author.bio}
              </p>
            </div>
          )}

          {related.length > 0 && (
            <section className="mt-14 border-t border-hair pt-8">
              <h2 className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
                Related articles
              </h2>
              <ul className="mt-4 space-y-3">
                {related.map((relatedArticle) => (
                  <li key={relatedArticle.slug}>
                    <Link
                      href={`/blog/${relatedArticle.slug}`}
                      className="font-mono text-[15px] font-medium text-offwhite transition-colors hover:text-acid"
                    >
                      {relatedArticle.title}
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </div>
      </article>
    </PageShell>
  );
}
