import type { Metadata } from "next";
import Link from "next/link";
import PageShell from "@/components/PageShell";

export const metadata: Metadata = {
  title: "Page not found",
  description: "The page you are looking for does not exist or may have moved.",
};

export default function NotFound() {
  return (
    <PageShell>
      <div className="flex min-h-[60vh] flex-col items-center justify-center px-6 py-24 text-center">
        <p className="font-mono text-[10px] uppercase tracking-micro text-acid">
          404
        </p>
        <h1 className="mt-4 font-mono text-[2rem] font-semibold leading-[1.1] tracking-[-0.02em] text-offwhite sm:text-[2.5rem]">
          Page not found
        </h1>
        <p className="mt-4 max-w-md font-sans text-[15px] leading-[1.75] text-offwhite/45">
          The page you are looking for does not exist or may have moved.
        </p>
        <div className="mt-10 flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/"
            className="rounded-md bg-acid px-4 py-2 font-sans text-[13px] font-semibold text-void transition-colors hover:bg-acid-bright"
          >
            Go to homepage
          </Link>
          <Link
            href="/blog"
            className="rounded-md border border-hair bg-cell px-4 py-2 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
          >
            Read the blog
          </Link>
          <Link
            href="/docs"
            className="rounded-md border border-hair bg-cell px-4 py-2 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
          >
            Browse the docs
          </Link>
        </div>
      </div>
    </PageShell>
  );
}
