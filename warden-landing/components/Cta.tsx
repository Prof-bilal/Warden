import { Github } from "lucide-react";

export default function Cta() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20 text-center">
        <h2 className="mx-auto max-w-[26rem] text-[1.75rem] font-medium leading-[1.2] tracking-[-0.01em] text-paper">
          Try it against a server you didn&apos;t write.
        </h2>
        <p className="mx-auto mt-3 max-w-[26rem] text-[1rem] leading-[1.6] text-muted">
          That&apos;s the real test — not the one that already trusts you.
        </p>
        <a
          href="https://github.com/Prof-bilal/Warden"
          className="mt-8 inline-flex items-center gap-2 rounded-sm border border-ink-600 px-5 py-2.5 text-[0.9375rem] text-paper transition-colors hover:border-blueprint hover:text-blueprint"
        >
          <Github size={16} />
          View on GitHub
        </a>
      </div>
    </section>
  );
}
