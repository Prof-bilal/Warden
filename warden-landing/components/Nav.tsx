"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowRight, Download, Github, Menu, Star, X } from "lucide-react";
import { formatCount } from "@/lib/stats";
import WardenMark from "@/components/bp/WardenMark";

const navLinks = [
  { href: "/docs", label: "Docs" },
  { href: "/protection", label: "Security" },
  { href: "/guides", label: "Examples" },
  { href: "/blog", label: "Blog" },
];

type Stats = { stars: number | null; downloads: number | null };

export default function Nav() {
  const [open, setOpen] = useState(false);
  const [stats, setStats] = useState<Stats>({ stars: null, downloads: null });

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 15000);

    fetch("/api/stats", { signal: controller.signal })
      .then((response) => (response.ok ? response.json() : null))
      .then((data: Stats | null) => {
        if (cancelled || !data) return;
        const stars =
          typeof data.stars === "number" && data.stars >= 0 ? data.stars : null;
        const downloads =
          typeof data.downloads === "number" && data.downloads >= 0
            ? data.downloads
            : null;
        if (stars !== null || downloads !== null) setStats({ stars, downloads });
      })
      .catch(() => {})
      .finally(() => clearTimeout(timer));

    return () => {
      cancelled = true;
      controller.abort();
      clearTimeout(timer);
    };
  }, []);

  const starsLabel = stats.stars === null ? null : formatCount(stats.stars);
  const downloadsLabel =
    stats.downloads === null ? null : formatCount(stats.downloads);

  return (
    <header className="sticky top-0 z-50 border-b border-hair bg-void/95 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-[1280px] items-center justify-between gap-6 px-5 sm:px-8">
        <Link href="/" className="flex shrink-0 items-center gap-2.5" aria-label="Warden home">
          <WardenMark size={22} className="text-acid" />
          <span className="font-mono text-[15px] font-semibold lowercase tracking-[-0.01em] text-offwhite">
            warden
          </span>
        </Link>

        <nav className="hidden items-center gap-7 lg:flex" aria-label="Primary navigation">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="font-mono text-[11px] uppercase tracking-[0.09em] text-offwhite/60 transition-colors hover:text-offwhite"
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-2.5">
          <span className="hidden h-4 w-px bg-hair sm:block" aria-hidden />
          <a
            href="https://www.npmjs.com/package/warden-sandbox-cli"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-hair px-3 py-1.5 font-mono text-[11px] text-offwhite/65 transition-colors hover:border-offwhite/25 hover:text-offwhite sm:inline-flex"
            aria-label="warden-sandbox-cli on npm"
          >
            <span className="lowercase">npm</span>
            {downloadsLabel !== null && (
              <span className="flex items-center gap-1 border-l border-hair pl-1.5 text-offwhite/45">
                <Download size={9} aria-hidden />
                {downloadsLabel}
              </span>
            )}
          </a>
          <a
            href="https://github.com/Prof-bilal/Warden"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-hair px-3 py-1.5 font-mono text-[11px] text-offwhite/65 transition-colors hover:border-offwhite/25 hover:text-offwhite sm:inline-flex"
            aria-label="Warden on GitHub"
          >
            <Github size={13} aria-hidden />
            GitHub
            {starsLabel !== null && (
              <span className="flex items-center gap-1 border-l border-hair pl-1.5 text-offwhite/45">
                <Star size={9} aria-hidden />
                {starsLabel}
              </span>
            )}
          </a>
          <Link
            href="/docs/install"
            className="hidden items-center gap-1.5 rounded-md bg-acid px-3.5 py-2 font-sans text-[12.5px] font-semibold text-void transition-colors hover:bg-acid-bright md:inline-flex"
          >
            Get started
            <ArrowRight size={13} aria-hidden />
          </Link>
          <button
            type="button"
            onClick={() => setOpen((value) => !value)}
            className="flex items-center justify-center rounded-md border border-hair p-2 text-offwhite/65 transition-colors hover:bg-offwhite/5 hover:text-offwhite lg:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
          >
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {open && (
        <nav
          className="border-t border-hair px-5 py-4 lg:hidden"
          aria-label="Mobile navigation"
        >
          <div className="mx-auto flex max-w-[1280px] flex-col gap-1">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                onClick={() => setOpen(false)}
                className="rounded-md px-3 py-2.5 font-mono text-[12px] uppercase tracking-[0.08em] text-offwhite/60 transition-colors hover:bg-offwhite/5 hover:text-offwhite"
              >
                {link.label}
              </Link>
            ))}
            <div className="mt-2 flex items-center gap-2 border-t border-hair pt-3">
              <a
                href="https://www.npmjs.com/package/warden-sandbox-cli"
                target="_blank"
                rel="noopener noreferrer"
                className="flex flex-1 items-center justify-center gap-1.5 rounded-md border border-hair px-3 py-2 font-mono text-[11px] text-offwhite/60"
              >
                <span className="lowercase">npm</span>
                {downloadsLabel !== null && (
                  <span className="flex items-center gap-1 text-offwhite/45">
                    <Download size={9} aria-hidden />
                    {downloadsLabel}
                  </span>
                )}
              </a>
              <a
                href="https://github.com/Prof-bilal/Warden"
                target="_blank"
                rel="noopener noreferrer"
                className="flex flex-1 items-center justify-center gap-1.5 rounded-md border border-hair px-3 py-2 font-mono text-[11px] text-offwhite/60"
              >
                <Github size={12} aria-hidden />
                <span>GitHub</span>
                {starsLabel !== null && (
                  <span className="flex items-center gap-1 text-offwhite/45">
                    <Star size={9} aria-hidden />
                    {starsLabel}
                  </span>
                )}
              </a>
            </div>
            <Link
              href="/docs/install"
              onClick={() => setOpen(false)}
              className="mt-2 flex items-center justify-center gap-2 rounded-md bg-acid px-3 py-2.5 text-center font-sans text-sm font-semibold text-void"
            >
              Get started
              <ArrowRight size={14} aria-hidden />
            </Link>
          </div>
        </nav>
      )}
    </header>
  );
}
