import Image from "next/image";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import type { Components } from "react-markdown";

// Server component: article content is rendered to static HTML so it is
// fully available to search engines without client-side JavaScript.
const components: Components = {
  img({ node, src, alt }) {
    if (typeof src !== "string") return null;
    return (
      <Image
        src={src}
        alt={alt ?? ""}
        width={1376}
        height={768}
        loading="lazy"
        sizes="(min-width: 48rem) 720px, calc(100vw - 3rem)"
      />
    );
  },
};

interface Props {
  content: string;
}

export default function ArticleMarkdown({ content }: Props) {
  return (
    <div className="prose-warden">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
}
