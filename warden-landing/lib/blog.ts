import fs from "fs";
import path from "path";
import matter from "gray-matter";

export interface Article {
  slug: string;
  title: string;
  description: string;
  /** ISO date string, exactly as written in frontmatter. */
  date: string;
  /** ISO date string, present only when the article was actually updated. */
  updated?: string;
  authorSlug: string;
  categorySlug: string;
  tags: string[];
  image?: string;
  draft: boolean;
  /** Placeholder article proving the publishing system; replace with real content. */
  sample: boolean;
  body: string;
}

export interface Category {
  slug: string;
  name: string;
  description: string;
}

export interface Author {
  slug: string;
  name: string;
  url: string;
  role: string;
  bio: string;
}

// Only real categories and authors. Add entries here as content actually grows.
export const CATEGORIES: Category[] = [
  {
    slug: "engineering",
    name: "Engineering",
    description:
      "How Warden works under the hood: policy enforcement, sandboxing backends, and platform internals.",
  },
  {
    slug: "security",
    name: "Security",
    description:
      "Sandboxing, trust boundaries, and practical security for MCP servers and AI tooling.",
  },
];

export const AUTHORS: Author[] = [
  {
    slug: "prof-bilal",
    name: "Prof-bilal",
    url: "https://github.com/Prof-bilal",
    role: "Creator and maintainer of Warden",
    bio: "Creator and maintainer of Warden, an open-source sandbox runtime for MCP servers working on OS-level sandboxing, policy enforcement, and developer tooling across Linux, macOS, and Windows.",
  },
];

const ARTICLES_DIR = path.join(process.cwd(), "content", "articles");

interface FrontMatter {
  title?: string;
  description?: string;
  date?: string;
  updated?: string;
  author?: string;
  category?: string;
  tags?: string[];
  image?: string;
  draft?: boolean;
  sample?: boolean;
}

function parseArticle(fileName: string): Article | null {
  const slug = fileName.replace(/\.md$/, "");
  let raw: string;
  try {
    raw = fs.readFileSync(path.join(ARTICLES_DIR, fileName), "utf-8");
  } catch {
    return null;
  }

  const { data, content: rawContent } = matter(raw);
  const fm = data as FrontMatter;

  // Raw HTML comments are not valid rendered content; strip them so they
  // never leak into the rendered article.
  const content = rawContent.replace(/<!--[\s\S]*?-->/g, "");

  if (!fm.title || !fm.description || !fm.date || !fm.author || !fm.category) {
    return null;
  }

  return {
    slug,
    title: fm.title,
    description: fm.description,
    date: fm.date,
    ...(fm.updated ? { updated: fm.updated } : {}),
    authorSlug: fm.author,
    categorySlug: fm.category,
    tags: Array.isArray(fm.tags) ? fm.tags : [],
    ...(fm.image ? { image: fm.image } : {}),
    draft: fm.draft === true,
    sample: fm.sample === true,
    body: content.trim(),
  };
}

export function getAllArticles(): Article[] {
  let files: string[] = [];
  try {
    files = fs.readdirSync(ARTICLES_DIR).filter((f) => f.endsWith(".md"));
  } catch {
    return [];
  }

  return files
    .map(parseArticle)
    .filter((a): a is Article => a !== null && !a.draft)
    .sort((a, b) => b.date.localeCompare(a.date));
}

export function getArticleBySlug(slug: string): Article | null {
  return getAllArticles().find((a) => a.slug === slug) ?? null;
}

export function getArticlesByCategory(categorySlug: string): Article[] {
  return getAllArticles().filter((a) => a.categorySlug === categorySlug);
}

export function getArticlesByAuthor(authorSlug: string): Article[] {
  return getAllArticles().filter((a) => a.authorSlug === authorSlug);
}

export function getCategoryBySlug(slug: string): Category | null {
  return CATEGORIES.find((c) => c.slug === slug) ?? null;
}

export function getAuthorBySlug(slug: string): Author | null {
  return AUTHORS.find((a) => a.slug === slug) ?? null;
}

export function getRelatedArticles(article: Article, limit = 2): Article[] {
  return getAllArticles()
    .filter((a) => a.slug !== article.slug)
    .sort((a, b) => {
      const aMatch = a.categorySlug === article.categorySlug ? 0 : 1;
      const bMatch = b.categorySlug === article.categorySlug ? 0 : 1;
      return aMatch - bMatch || b.date.localeCompare(a.date);
    })
    .slice(0, limit);
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat("en-US", {
    timeZone: "UTC",
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(new Date(iso));
}
