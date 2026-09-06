"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";

function rewriteHref(href: string): string {
  if (!href) return href;

  // External links — keep as-is
  if (href.startsWith("http://") || href.startsWith("https://") || href.startsWith("mailto:")) {
    return href;
  }

  // Anchor-only links — keep as-is
  if (href.startsWith("#")) {
    return href;
  }

  // Split off anchor fragment
  const hashIndex = href.indexOf("#");
  const hash = hashIndex !== -1 ? href.slice(hashIndex) : "";
  const base = hashIndex !== -1 ? href.slice(0, hashIndex) : href;

  // Rewrite .md links: "schema.md" → "/docs/schema", "./schema.md" → "/docs/schema"
  const mdMatch = base.match(/^(?:\.\/)?([^/].*)\.md$/);
  if (mdMatch) {
    return `/docs/${mdMatch[1]}${hash}`;
  }

  // Links already starting with /docs/ or / — keep as-is
  if (base.startsWith("/")) {
    return base + hash;
  }

  // Other relative links — prefix with /docs/
  if (base.length > 0) {
    return `/docs/${base}${hash}`;
  }

  return href + hash;
}

const components: Components = {
  a({ href, children, ...props }) {
    const rewritten = rewriteHref(href ?? "");
    return (
      <a href={rewritten} {...props}>
        {children}
      </a>
    );
  },
};

interface Props {
  content: string;
}

export default function MarkdownContent({ content }: Props) {
  return (
    <div className="prose-warden">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}
