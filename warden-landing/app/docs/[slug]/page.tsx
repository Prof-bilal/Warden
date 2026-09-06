import fs from "fs";
import path from "path";
import { notFound } from "next/navigation";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import MarkdownContent from "@/components/MarkdownContent";

const DOCS_DIR = path.resolve(
  process.cwd(),
  "../warden-starter/warden/docs"
);

function getDocsDir() {
  return DOCS_DIR;
}

function readDoc(slug: string): string | null {
  const filePath = path.join(getDocsDir(), `${slug}.md`);
  try {
    return fs.readFileSync(filePath, "utf-8");
  } catch {
    return null;
  }
}

function getAllDocSlugs(): string[] {
  try {
    return fs
      .readdirSync(getDocsDir())
      .filter((f) => f.endsWith(".md"))
      .map((f) => f.replace(/\.md$/, ""));
  } catch {
    return [];
  }
}

export function generateStaticParams() {
  return getAllDocSlugs().map((slug) => ({ slug }));
}

export function generateMetadata({ params }: { params: { slug: string } }) {
  const content = readDoc(params.slug);
  if (!content) return { title: "Not Found — Warden" };

  const firstLine = content.split("\n").find((l) => l.startsWith("# "));
  const title = firstLine
    ? firstLine.replace(/^#\s*/, "")
    : params.slug.replace(/-/g, " ");

  return {
    title: `${title} — Warden`,
    description: `Warden documentation: ${title}`,
  };
}

export default function DocPage({ params }: { params: { slug: string } }) {
  const content = readDoc(params.slug);
  if (!content) notFound();

  const firstLine = content.split("\n").find((l) => l.startsWith("# "));
  const title = firstLine
    ? firstLine.replace(/^#\s*/, "")
    : params.slug.replace(/-/g, " ");

  const slugs = getAllDocSlugs();
  const currentIndex = slugs.indexOf(params.slug);
  const prevSlug = currentIndex > 0 ? slugs[currentIndex - 1] : null;
  const nextSlug =
    currentIndex < slugs.length - 1 ? slugs[currentIndex + 1] : null;

  const slugToTitle = (s: string) => {
    const c = readDoc(s);
    if (!c) return s.replace(/-/g, " ");
    const h1 = c.split("\n").find((l) => l.startsWith("# "));
    return h1 ? h1.replace(/^#\s*/, "") : s.replace(/-/g, " ");
  };

  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <article className="mx-auto max-w-content px-6 pb-20 pt-16 md:pt-24">
        <div className="mx-auto max-w-[52rem]">
          <div className="mb-10">
            <a
              href="/docs"
              className="text-[0.875rem] text-muted transition-colors hover:text-blueprint"
            >
              ← Back to Docs
            </a>
          </div>

          <MarkdownContent content={content} />

          <div className="mt-16 flex items-center justify-between border-t border-ink-700 pt-8">
            {prevSlug ? (
              <a
                href={`/docs/${prevSlug}`}
                className="text-[0.875rem] text-muted transition-colors hover:text-paper"
              >
                ← {slugToTitle(prevSlug)}
              </a>
            ) : (
              <span />
            )}
            {nextSlug ? (
              <a
                href={`/docs/${nextSlug}`}
                className="text-[0.875rem] text-muted transition-colors hover:text-paper"
              >
                {slugToTitle(nextSlug)} →
              </a>
            ) : (
              <span />
            )}
          </div>
        </div>
      </article>
      <Footer />
    </main>
  );
}
