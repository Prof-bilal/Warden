import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
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
    title: author.name,
    description: `Articles by ${author.name}, ${author.role.toLowerCase()}, published on the Warden blog.`,
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd data={personSchema(author)} />
      <JsonLd data={breadcrumbs} />
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
          <span className="text-paper">{author.name}</span>
        </nav>

        <h1 className="text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          {author.name}
        </h1>
        <p className="mt-2 text-[0.9375rem] text-blueprint">{author.role}</p>
        <p className="mt-4 text-[1.0625rem] leading-relaxed text-muted">
          {author.bio}
        </p>
        <p className="mt-4 text-[0.875rem]">
          <a
            href={author.url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-paper transition-colors hover:text-blueprint"
          >
            {author.url.replace("https://", "")}
          </a>
        </p>

        <h2 className="mt-14 border-t border-ink-800 pt-8 font-mono text-[0.6875rem] uppercase tracking-widest text-muted">
          Articles
        </h2>
        <div className="mt-4">
          {articles.length === 0 ? (
            <p className="py-12 text-muted">No articles published yet.</p>
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
      <Footer />
    </main>
  );
}
