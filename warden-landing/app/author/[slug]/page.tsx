import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import PageShell from "@/components/PageShell";
import JsonLd from "@/components/JsonLd";
import ArticleCard from "@/components/ArticleCard";
import {
  AUTHORS,
  getArticlesByAuthor,
  getAuthorBySlug,
  getCategoryBySlug,
} from "@/lib/blog";
import { SITE_URL, createMetadata } from "@/lib/seo";
import { breadcrumbSchema, personSchema } from "@/lib/schema";

export function generateStaticParams() {
  return AUTHORS.map((author) => ({ slug: author.slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const author = getAuthorBySlug(slug);
  if (!author) return { title: "Not Found" };

  return createMetadata({
    title: `Articles by ${author.name}`,
    description: author.bio,
    path: `/author/${author.slug}`,
  });
}

export default async function AuthorPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const author = getAuthorBySlug(slug);
  if (!author) notFound();

  const articles = getArticlesByAuthor(author.slug);

  const breadcrumbs = breadcrumbSchema([
    { name: "Home", url: `${SITE_URL}/` },
    { name: "Blog", url: `${SITE_URL}/blog` },
    { name: author.name, url: `${SITE_URL}/author/${author.slug}` },
  ]);

  return (
    <PageShell>
      <JsonLd data={personSchema(author)} />
      <JsonLd data={breadcrumbs} />

      <div className="px-6 pb-20 pt-10 md:pt-14">
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
            <span className="text-offwhite/20">/</span>
            <span className="text-offwhite/70">{author.name}</span>
          </nav>

          <h1 className="font-mono text-[1.9rem] font-semibold leading-[1.12] tracking-[-0.02em] text-offwhite sm:text-[2.25rem]">
            Articles by {author.name}
          </h1>
          <p className="mt-2 font-mono text-[11px] uppercase tracking-micro text-acid">
            {author.role}
          </p>
          <p className="mt-4 font-sans text-[15px] leading-[1.75] text-offwhite/50">
            {author.bio}
          </p>
          <p className="mt-4 font-mono text-[12px]">
            <a
              href={author.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-offwhite/70 transition-colors hover:text-acid"
            >
              {author.url.replace("https://", "")}
            </a>
          </p>

          <h2 className="mt-14 border-t border-hair pt-8 font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
            Articles
          </h2>
          <div className="mt-4">
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
                  author={author}
                />
              ))
            )}
          </div>
        </div>
      </div>
    </PageShell>
  );
}
