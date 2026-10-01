import Link from "next/link";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
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
    <main className="min-h-screen bg-ink-950">
      <JsonLd data={articleSchema} />
      <JsonLd
        data={breadcrumbSchema([
          { name: "Home", url: `${SITE_URL}/` },
          { name: "Guides", url: `${SITE_URL}/guides` },
          { name: topic.title, url },
        ])}
      />
      {topic.faqs.length > 0 && <JsonLd data={faqPageSchema(url, topic.faqs)} />}
      <Nav />
      <article className="mx-auto max-w-[48rem] px-6 pb-20 pt-10 md:pt-16">
        <nav aria-label="Breadcrumb" className="mb-8 font-hero text-[0.8125rem] text-muted">
          <Link href="/" className="transition-colors hover:text-paper">
            Home
          </Link>
          <span className="mx-1.5 text-ink-600">/</span>
          <Link href="/guides" className="transition-colors hover:text-paper">
            Guides
          </Link>
        </nav>

        <h1 className="font-hero text-[2.25rem] font-bold leading-[1.1] tracking-[-0.02em] text-paper sm:text-[2.5rem]">
          {topic.title}
        </h1>
        <p className="mt-5 font-hero text-[1.0625rem] leading-[1.7] text-muted">{topic.answer}</p>
        <p className="mt-4 font-hero text-[0.875rem] text-muted">
          <Link href="/author/prof-bilal" className="text-paper hover:text-blueprint">
            Abdullah Bilal
          </Link>
          <span className="mx-2 text-ink-600">·</span>
          <time dateTime={topic.date}>Published {formatDate(topic.date)}</time>
        </p>

        {topic.tldr.length > 0 && (
          <section className="mt-8 rounded-2xl border border-ink-700 bg-ink-900 px-5 py-4">
            <h2 className="font-hero text-[0.6875rem] font-bold uppercase tracking-[0.12em] text-grant">
              Answer in short
            </h2>
            <ul className="mt-3 list-disc space-y-1.5 pl-5 font-hero text-[0.9375rem] leading-[1.6] text-muted">
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
            <h2 className="font-hero text-[1.375rem] font-bold text-paper">Questions people ask</h2>
            <dl className="mt-4 divide-y divide-ink-800 border-y border-ink-800">
              {topic.faqs.map((item) => (
                <div key={item.question} className="py-5">
                  <dt className="font-hero text-[1rem] font-bold text-paper">{item.question}</dt>
                  <dd className="mt-2 font-hero text-[0.9375rem] leading-[1.65] text-muted">
                    {item.answer}
                  </dd>
                </div>
              ))}
            </dl>
          </section>
        )}

        {topic.related.length > 0 && (
          <section className="mt-12 border-t border-ink-800 pt-8">
            <h2 className="font-hero text-[0.6875rem] font-bold uppercase tracking-[0.12em] text-muted">
              Keep reading
            </h2>
            <ul className="mt-4 space-y-3">
              {topic.related.map((link) => (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    className="font-hero text-[1.0625rem] font-medium text-paper hover:text-blueprint"
                  >
                    {link.label}
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
