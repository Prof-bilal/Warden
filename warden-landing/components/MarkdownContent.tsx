"use client";

import { useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";
import { Check, Copy } from "lucide-react";

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

/** A markdown code block with a hover-revealed copy-to-clipboard button. */
function CodeBlock({ children }: { children: React.ReactNode }) {
  const preRef = useRef<HTMLPreElement>(null);
  const [copied, setCopied] = useState(false);

  function handleCopy() {
    const text = preRef.current?.textContent ?? "";
    if (!text) return;
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  }

  return (
    <div className="group relative">
      <pre ref={preRef}>{children}</pre>
      <button
        onClick={handleCopy}
        aria-label="Copy code"
        className="absolute right-2 top-2 flex items-center gap-1.5 rounded-md border border-ink-700 bg-ink-900 px-2 py-1 font-mono text-[0.6875rem] text-muted opacity-0 transition-all hover:border-ink-500 hover:text-paper focus-visible:opacity-100 group-hover:opacity-100"
      >
        {copied ? (
          <>
            <Check size={11} className="text-grant" />
            <span className="text-grant">Copied</span>
          </>
        ) : (
          <>
            <Copy size={11} />
            <span>Copy</span>
          </>
        )}
      </button>
    </div>
  );
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
  pre({ children, ...props }) {
    return (
      <CodeBlock>{children}</CodeBlock>
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
