import { SITE_NAME, SITE_URL } from "./seo";
import type { Article, Author } from "./blog";

const GITHUB_URL = "https://github.com/Prof-bilal/Warden";

function organizationRef() {
  return {
    "@type": "Organization",
    name: SITE_NAME,
    url: SITE_URL,
    logo: {
      "@type": "ImageObject",
      url: `${SITE_URL}/icon.png`,
    },
  };
}

export function personRef(author: Author) {
  return {
    "@type": "Person",
    name: author.name,
    url: `${SITE_URL}/author/${author.slug}`,
  };
}

export function websiteSchema() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: SITE_NAME,
    url: SITE_URL,
    description:
      "A sandbox runtime for MCP servers. Run third-party code with first-party trust.",
    publisher: organizationRef(),
  };
}

export function organizationSchema() {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: SITE_NAME,
    url: SITE_URL,
    logo: {
      "@type": "ImageObject",
      url: `${SITE_URL}/icon.png`,
    },
    sameAs: [GITHUB_URL],
    description:
      "Open-source sandbox runtime for MCP servers. Run third-party AI tooling code safely.",
  };
}

export function articleSchema(article: Article, author: Author | null) {
  const url = `${SITE_URL}/blog/${article.slug}`;
  return {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    "@id": `${url}#article`,
    mainEntityOfPage: {
      "@type": "WebPage",
      "@id": url,
    },
    headline: article.title,
    description: article.description,
    image: [`${SITE_URL}${article.image ?? "/og-image.png"}`],
    datePublished: article.date,
    // Only present when the article was actually updated.
    ...(article.updated ? { dateModified: article.updated } : {}),
    author: author
      ? personRef(author)
      : { "@type": "Person", name: article.authorSlug },
    publisher: organizationRef(),
  };
}

export function breadcrumbSchema(items: { name: string; url: string }[]) {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((item, index) => ({
      "@type": "ListItem",
      position: index + 1,
      name: item.name,
      item: item.url,
    })),
  };
}

export function personSchema(author: Author) {
  return {
    "@context": "https://schema.org",
    "@type": "Person",
    name: author.name,
    url: `${SITE_URL}/author/${author.slug}`,
    description: author.bio,
    sameAs: [author.url],
  };
}

export function collectionPageSchema({
  name,
  description,
  url,
  itemUrls,
}: {
  name: string;
  description: string;
  url: string;
  itemUrls?: string[];
}) {
  return {
    "@context": "https://schema.org",
    "@type": "CollectionPage",
    name,
    description,
    url,
    isPartOf: { "@type": "WebSite", name: SITE_NAME, url: SITE_URL },
    ...(itemUrls && itemUrls.length > 0
      ? {
          mainEntity: {
            "@type": "ItemList",
            itemListElement: itemUrls.map((itemUrl, index) => ({
              "@type": "ListItem",
              position: index + 1,
              url: itemUrl,
            })),
          },
        }
      : {}),
  };
}
