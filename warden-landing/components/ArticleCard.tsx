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
    <article className="border-b border-hair py-8 first:pt-0 last:border-b-0">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35">
        {category && (
          <Link
            href={`/blog/category/${category.slug}`}
            className="text-acid transition-colors hover:text-acid-bright"
          >
            {category.name}
          </Link>
        )}
        <time dateTime={article.date}>{formatDate(article.date)}</time>
      </div>

      <h3 className="mt-3 font-mono text-[1.2rem] font-semibold leading-snug text-offwhite">
        <Link
          href={`/blog/${article.slug}`}
          className="transition-colors hover:text-acid"
        >
          {article.title}
        </Link>
      </h3>

      <p className="mt-2 font-sans text-[14px] leading-[1.7] text-offwhite/45">
        {article.description}
      </p>

      {author && (
        <p className="mt-3 font-mono text-[11px] text-offwhite/35">
          By{" "}
          <Link
            href={`/author/${author.slug}`}
            className="text-offwhite/70 transition-colors hover:text-acid"
          >
            {author.name}
          </Link>
        </p>
      )}
    </article>
  );
}
