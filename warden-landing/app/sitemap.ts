import type { MetadataRoute } from "next";
import fs from "fs";
import path from "path";
import { SITE_URL } from "@/lib/seo";
import { AUTHORS, CATEGORIES, getAllArticles } from "@/lib/blog";

const DOCS_DIR = path.resolve(
  process.cwd(),
  "content/docs"
);

function getAllDocSlugs(): string[] {
  try {
    return fs
      .readdirSync(DOCS_DIR)
      .filter((f) => f.endsWith(".md"))
      .map((f) => f.replace(/\.md$/, ""))
      // /docs/index duplicates the docs hub at /docs — keep it out of the sitemap.
      .filter((slug) => slug !== "index");
  } catch {
    return [];
  }
}

export default function sitemap(): MetadataRoute.Sitemap {
  // lastmod is only set where we have a real content date (articles).
  // Faking build-time lastmod on static pages erodes crawler trust.
  const staticPages: MetadataRoute.Sitemap = [
    {
      url: SITE_URL,
      changeFrequency: "weekly",
      priority: 1,
    },
    {
      url: `${SITE_URL}/blog`,
      changeFrequency: "weekly",
      priority: 0.9,
    },
    {
      url: `${SITE_URL}/docs`,
      changeFrequency: "weekly",
      priority: 0.9,
    },
    {
      url: `${SITE_URL}/about`,
      changeFrequency: "monthly",
      priority: 0.8,
    },
    {
      url: `${SITE_URL}/features`,
      changeFrequency: "monthly",
      priority: 0.8,
    },
    {
      url: `${SITE_URL}/testing`,
      changeFrequency: "monthly",
      priority: 0.7,
    },
    {
      url: `${SITE_URL}/contact`,
      changeFrequency: "yearly",
      priority: 0.3,
    },
    {
      url: `${SITE_URL}/privacy`,
      changeFrequency: "yearly",
      priority: 0.3,
    },
    {
      url: `${SITE_URL}/terms`,
      changeFrequency: "yearly",
      priority: 0.3,
    },
  ];

  // Article lastModified comes from frontmatter (updated ?? date) — real dates only.
  const articlePages: MetadataRoute.Sitemap = getAllArticles().map(
    (article) => ({
      url: `${SITE_URL}/blog/${article.slug}`,
      lastModified: new Date(article.updated ?? article.date),
      changeFrequency: "monthly" as const,
      priority: 0.8,
    })
  );

  const categoryPages: MetadataRoute.Sitemap = CATEGORIES.map((category) => ({
    url: `${SITE_URL}/blog/category/${category.slug}`,
    changeFrequency: "weekly" as const,
    priority: 0.6,
  }));

  const authorPages: MetadataRoute.Sitemap = AUTHORS.map((author) => ({
    url: `${SITE_URL}/author/${author.slug}`,
    changeFrequency: "weekly" as const,
    priority: 0.5,
  }));

  const docPages: MetadataRoute.Sitemap = getAllDocSlugs().map((slug) => ({
    url: `${SITE_URL}/docs/${slug}`,
    changeFrequency: "monthly" as const,
    priority: 0.7,
  }));

  return [
    ...staticPages,
    ...articlePages,
    ...categoryPages,
    ...authorPages,
    ...docPages,
  ];
}
