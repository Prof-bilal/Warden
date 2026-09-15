import Link from "next/link";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col bg-ink-950">
      <Nav />
      <div className="flex flex-1 flex-col items-center justify-center px-6 py-24 text-center">
        <p className="font-mono text-[0.6875rem] uppercase tracking-widest text-blueprint">
          404
        </p>
        <h1 className="mt-4 text-[2.5rem] font-semibold leading-[1.1] tracking-[-0.02em] text-paper">
          Page not found
        </h1>
        <p className="mt-4 max-w-md text-[1.0625rem] leading-relaxed text-muted">
          The page you are looking for does not exist or may have moved.
        </p>
        <div className="mt-10 flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/"
            className="rounded-lg border border-ink-600 px-4 py-2 text-[0.875rem] font-medium text-paper transition-colors hover:border-blueprint/50"
          >
            Go to homepage
          </Link>
          <Link
            href="/blog"
            className="rounded-lg border border-ink-600 px-4 py-2 text-[0.875rem] font-medium text-paper transition-colors hover:border-blueprint/50"
          >
            Read the blog
          </Link>
          <Link
            href="/docs"
            className="rounded-lg border border-ink-600 px-4 py-2 text-[0.875rem] font-medium text-paper transition-colors hover:border-blueprint/50"
          >
            Browse the docs
          </Link>
        </div>
      </div>
      <Footer />
    </main>
  );
}
