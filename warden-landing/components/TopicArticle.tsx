import Link from "next/link";
import PageShell from "@/components/PageShell";
import JsonLd from "@/components/JsonLd";
import ArticleMarkdown from "@/components/ArticleMarkdown";
import { breadcrumbSchema, faqPageSchema } from "@/lib/schema";
import { SITE_URL } from "@/lib/seo";
import { formatDate } from "@/lib/blog";
import type { Topic } from "@/lib/topics";

export default function TopicArticle({ topic }: { topic: Topic }) {
  const url = `${SITE_URL}${topic.path}`;

  const articleSchema = {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    headline: topic.title,
    description: topic.description,
    datePublished: topic.date,
    url,
    author: {
      "@type": "Person",
      name: "Abdullah Bilal",
      alternateName: "Prof-bilal",
      url: `${SITE_URL}/author/prof-bilal`,
    },
    publisher: {
      "@type": "Organization",
      name: "Warden",
      url: SITE_URL,
    },
    mainEntityOfPage: url,
  };

  return (
    <PageShell>
      <JsonLd data={articleSchema} />
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Guides", url: `${SITE_URL}/guides` },
          { name: topic.title, url },
        ])}
      />
      {topic.faqs.length > 0 && <JsonLd data={faqPageSchema(url, topic.faqs)} />}
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
            <Link href="/guides" className="transition-colors hover:text-offwhite">
              Guides
            </Link>
          </nav>

          <h1 className="font-mono text-[1.9rem] font-semibold leading-[1.12] tracking-[-0.02em] text-offwhite sm:text-[2.25rem]">
            {topic.title}
          </h1>
          <p className="mt-5 font-sans text-[15px] leading-[1.75] text-offwhite/50">
            {topic.answer}
          </p>
          <p className="mt-4 font-mono text-[11px] uppercase tracking-micro text-offwhite/35">
            <Link
              href="/author/prof-bilal"
              className="text-offwhite/70 transition-colors hover:text-acid"
            >
              Abdullah Bilal
            </Link>
            <span className="mx-2 text-offwhite/20">·</span>
            <time dateTime={topic.date}>Published {formatDate(topic.date)}</time>
          </p>

          {topic.tldr.length > 0 && (
            <section className="mt-8 border border-hair bg-cell px-5 py-4">
              <h2 className="font-mono text-[9.5px] uppercase tracking-micro text-acid">
                Answer in short
              </h2>
              <ul className="mt-3 list-disc space-y-1.5 pl-5 font-sans text-[14px] leading-[1.65] text-offwhite/50">
                {topic.tldr.map((point) => (
                  <li key={point}>{point}</li>
                ))}
              </ul>
            </section>
          )}

          <div className="mt-10">
            <ArticleMarkdown content={topic.body} />
          </div>

          {topic.faqs.length > 0 && (
            <section className="mt-12">
              <h2 className="font-mono text-[1.35rem] font-semibold text-offwhite">
                Questions people ask
              </h2>
              <dl className="mt-4 divide-y divide-hair border-y border-hair">
                {topic.faqs.map((item) => (
                  <div key={item.question} className="py-5">
                    <dt className="font-mono text-[15px] font-semibold text-offwhite">
                      {item.question}
                    </dt>
                    <dd className="mt-2 font-sans text-[14px] leading-[1.7] text-offwhite/50">
                      {item.answer}
                    </dd>
                  </div>
                ))}
              </dl>
            </section>
          )}

          {topic.related.length > 0 && (
            <section className="mt-12 border-t border-hair pt-8">
              <h2 className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
                Keep reading
              </h2>
              <ul className="mt-4 space-y-3">
                {topic.related.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={link.href}
                      className="font-mono text-[15px] font-medium text-offwhite transition-colors hover:text-acid"
                    >
                      {link.label}
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
