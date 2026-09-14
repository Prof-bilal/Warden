import Link from "next/link";

export default function Footer() {
  return (
    <footer className="border-t border-ink-800">
      <div className="mx-auto flex max-w-content flex-col items-start justify-between gap-4 px-6 py-10 text-[0.8125rem] text-muted md:flex-row md:items-center">
        <span>wardenMIT licensed</span>
        <div className="flex flex-wrap gap-6">
          <Link href="/#how-it-works" className="transition-colors hover:text-paper">
            How it works
          </Link>
          <Link href="/#how-to-use" className="transition-colors hover:text-paper">
            How to use
          </Link>
          <Link href="/about" className="transition-colors hover:text-paper">
            About
          </Link>
          <Link href="/docs" className="transition-colors hover:text-paper">
            Docs
          </Link>
          <Link
            href="/docs/roadmap"
            className="transition-colors hover:text-paper"
          >
            Roadmap
          </Link>
          <Link href="/contact" className="transition-colors hover:text-paper">
            Contact
          </Link>
          <Link href="/privacy" className="transition-colors hover:text-paper">
            Privacy
          </Link>
          <Link href="/terms" className="transition-colors hover:text-paper">
            Terms
          </Link>
          <a
            href="https://github.com/Prof-bilal/Warden"
            className="transition-colors hover:text-paper"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  );
}
