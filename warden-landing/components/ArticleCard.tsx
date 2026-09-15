import Link from "next/link";
import type { Article, Author, Category } from "@/lib/blog";
import { formatDate } from "@/lib/blog";

interface Props {
  article: Article;
  category: Category | null;
  author: Author | null;
}

export default function ArticleCard({ article, category, author }: Props) {
  return (
    <article className="border-b border-ink-800 py-8 first:pt-0 last:border-b-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 font-hero text-[0.6875rem] uppercase tracking-[0.12em] text-muted">
        {category && (
          <Link
            href={`/blog/category/${category.slug}`}
            className="text-blueprint transition-colors hover:text-paper"
          >
            {category.name}
          </Link>
        )}
        <time dateTime={article.date}>{formatDate(article.date)}</time>
      </div>

      <h3 className="mt-3 font-hero text-[1.25rem] font-bold leading-snug text-paper">
        <Link
          href={`/blog/${article.slug}`}
          className="transition-colors hover:text-blueprint"
        >
          {article.title}
        </Link>
      </h3>

      <p className="mt-2 font-hero text-[0.9375rem] leading-relaxed text-muted">
        {article.description}
      </p>

      {author && (
        <p className="mt-3 font-hero text-[0.8125rem] text-muted">
          By{" "}
          <Link
            href={`/author/${author.slug}`}
            className="text-paper transition-colors hover:text-blueprint"
          >
            {author.name}
          </Link>
        </p>
      )}
    </article>
  );
}
