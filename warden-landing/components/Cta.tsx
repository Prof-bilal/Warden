import { Github } from "lucide-react";

export default function Cta() {
  return (
    <section className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20 text-center">
        <h2 className="mx-auto max-w-[26rem] font-hero text-[1.75rem] font-bold leading-[1.2] tracking-[-0.02em] text-paper md:text-[2rem]">
          Start with one local tool.
        </h2>
        <p className="mx-auto mt-3 max-w-[26rem] font-hero text-[1rem] leading-[1.6] text-muted">
          Install Warden, define what the tool needs, and keep unlisted access outside the boundary.
        </p>
        <a
          href="https://github.com/Prof-bilal/Warden"
          className="mt-8 inline-flex items-center gap-2 rounded-[10px] border border-ink-600 bg-ink-900 px-5 py-2.5 font-hero text-[0.9375rem] text-paper transition-colors hover:border-blueprint hover:text-blueprint"
        >
          <Github size={16} />
          View on GitHub
        </a>
      </div>
    </section>
  );
}
