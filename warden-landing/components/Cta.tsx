import Link from "next/link";
import { Github } from "lucide-react";

export default function Cta() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20 text-center">
        <h2 className="mx-auto max-w-[26rem] font-hero text-[1.75rem] font-bold leading-[1.2] tracking-[-0.02em] text-paper md:text-[2rem]">
          Start with one local tool.
        </h2>
        <p className="mx-auto mt-3 max-w-[30rem] font-sans text-[1rem] leading-[1.7] text-muted">
          Install Warden, define what your tool needs, and keep everything else outside the boundary.
        </p>
        <code className="mx-auto mt-7 block w-fit rounded-md border border-ink-700 bg-ink-900 px-4 py-2.5 font-mono text-sm text-paper">
          npm install -g warden-sandbox-cli
        </code>
        <div className="mt-7 flex flex-wrap justify-center gap-3">
          <Link
            href="/setup"
            className="inline-flex items-center rounded-md bg-blueprint px-5 py-2.5 font-sans text-sm font-semibold text-ink-950 transition-colors hover:bg-blueprint/90"
          >
            Start setup
          </Link>
          <a
            href="https://github.com/Prof-bilal/Warden"
            className="inline-flex items-center gap-2 rounded-md border border-ink-600 bg-ink-900 px-5 py-2.5 font-sans text-sm text-paper transition-colors hover:border-blueprint hover:text-blueprint"
          >
            <Github size={16} />
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  );
}
