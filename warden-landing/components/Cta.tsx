import Link from "next/link";
import { ArrowRight, Github } from "lucide-react";
import CopyChip from "@/components/bp/CopyChip";

export default function Cta() {
  return (
    <section>
      <div className="px-6 py-20 text-center md:py-24">
        <p className="font-mono text-[9.5px] uppercase tracking-micro text-offwhite/30">
          Start here
        </p>
        <h2 className="mx-auto mt-4 max-w-[26rem] font-mono text-[1.55rem] font-semibold leading-[1.2] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
          Start with one local tool.
        </h2>
        <p className="mx-auto mt-3 max-w-[30rem] font-sans text-[13.5px] leading-[1.75] text-offwhite/45">
          Install Warden, define what your tool needs, and keep everything else
          outside the boundary.
        </p>

        <CopyChip
          value="npm i -g warden-sandbox-cli"
          className="mx-auto mt-7 w-fit"
        />

        <div className="mt-7 flex flex-wrap justify-center gap-3">
          <Link
            href="/setup"
            className="inline-flex items-center gap-2 rounded-md bg-acid px-5 py-2.5 font-sans text-[13px] font-semibold text-void transition-colors hover:bg-acid-bright"
          >
            Start setup
            <ArrowRight size={14} aria-hidden />
          </Link>
          <a
            href="https://github.com/Prof-bilal/Warden"
            className="inline-flex items-center gap-2 rounded-md border border-hair bg-cell px-5 py-2.5 font-sans text-[13px] font-semibold text-offwhite/85 transition-colors hover:border-offwhite/20"
          >
            <Github size={14} aria-hidden />
            View on GitHub
          </a>
        </div>

        <p className="mx-auto mt-6 max-w-[30rem] font-mono text-[10px] leading-[1.8] text-offwhite/25">
          Pre-1.0 · MIT licensed · Runs locally. Verification maturity differs by
          operating system.
        </p>
      </div>
    </section>
  );
}
