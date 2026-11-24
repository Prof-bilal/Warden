"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { ArrowRight, Download, Github, Menu, Moon, Star, Sun, X } from "lucide-react";
import { formatCount } from "@/lib/stats";
import { themeStorageKey } from "@/lib/theme";

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
  const [dark, setDark] = useState(false);

  useEffect(() => {
    setDark(document.documentElement.dataset.theme === "dark");
  }, []);

  function toggleTheme() {
    const nextTheme = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = nextTheme;
    setDark(nextTheme === "dark");
    try {
      localStorage.setItem(themeStorageKey, nextTheme);
    } catch {
      // The current page can still switch themes when storage is unavailable.
    }
  }

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
    <header className="site-header sticky top-0 z-50 border-b border-hair bg-void">
      <div className="site-nav-row">
        <Link href="/" className="site-brand" aria-label="Warden home">
          <Image
            src="/warden-mark.png"
            alt=""
            width={26}
            height={26}
            className="h-[26px] w-[26px] shrink-0"
            priority
          />
          <span className="font-semibold">
            warden
          </span>
        </Link>

        <nav className="hidden items-center gap-7 lg:flex" aria-label="Primary navigation">
          {navLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="site-nav-link"
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
            className="hidden items-center gap-1.5 rounded-full border border-hair px-3 py-1.5 font-mono text-[11px] text-muted transition-colors hover:border-offwhite/25 hover:text-offwhite sm:inline-flex"
            aria-label="warden-sandbox-cli on npm"
          >
            <span className="lowercase">npm</span>
            {downloadsLabel !== null && (
              <span className="flex items-center gap-1 border-l border-hair pl-1.5 text-muted">
                <Download size={9} aria-hidden />
                {downloadsLabel}
              </span>
            )}
          </a>
          <a
            href="https://github.com/Prof-bilal/Warden"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-hair px-3 py-1.5 font-mono text-[11px] text-muted transition-colors hover:border-offwhite/25 hover:text-offwhite sm:inline-flex"
            aria-label="Warden on GitHub"
          >
            <Github size={13} aria-hidden />
            GitHub
            {starsLabel !== null && (
              <span className="flex items-center gap-1 border-l border-hair pl-1.5 text-muted">
                <Star size={9} aria-hidden />
                {starsLabel}
              </span>
            )}
          </a>
          <Link
            href="/docs/install"
            className="hidden items-center gap-1.5 rounded-md px-3.5 py-2 font-semibold md:inline-flex"
          >
            Get started
            <ArrowRight size={13} aria-hidden />
          </Link>
          <button
            type="button"
            onClick={toggleTheme}
            aria-label="Dark mode"
            aria-pressed={dark}
            className="theme-toggle inline-flex min-h-9 shrink-0 items-center rounded-full border border-hair px-2.5 text-xs hover:bg-offwhite/5"
          >
            <span className="theme-light" aria-hidden="true"><Sun size={14} /> Light</span>
            <span className="theme-dark" aria-hidden="true"><Moon size={14} /> Dark</span>
          </button>
          <button
            type="button"
            onClick={() => setOpen((value) => !value)}
            className="flex items-center justify-center rounded-md border border-hair p-2 text-muted transition-colors hover:bg-offwhite/5 hover:text-offwhite lg:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            aria-controls={open ? "mobile-navigation" : undefined}
          >
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {open && (
        <nav
          className="border-t border-hair px-5 py-4 lg:hidden"
          aria-label="Mobile navigation"
          id="mobile-navigation"
        >
          <div className="mx-auto flex max-w-[960px] flex-col gap-1">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                onClick={() => setOpen(false)}
                className="rounded-md px-3 py-2.5 site-nav-link"
              >
                {link.label}
              </Link>
            ))}
            <div className="mt-2 flex items-center gap-2 border-t border-hair pt-3">
              <a
                href="https://www.npmjs.com/package/warden-sandbox-cli"
                target="_blank"
                rel="noopener noreferrer"
                className="flex flex-1 items-center justify-center gap-1.5 rounded-md border border-hair px-3 py-2 font-mono text-[11px] text-muted"
              >
                <span className="lowercase">npm</span>
                {downloadsLabel !== null && (
                  <span className="flex items-center gap-1 text-muted">
                    <Download size={9} aria-hidden />
                    {downloadsLabel}
                  </span>
                )}
              </a>
              <a
                href="https://github.com/Prof-bilal/Warden"
                target="_blank"
                rel="noopener noreferrer"
                className="flex flex-1 items-center justify-center gap-1.5 rounded-md border border-hair px-3 py-2 font-mono text-[11px] text-muted"
              >
                <Github size={12} aria-hidden />
                <span>GitHub</span>
                {starsLabel !== null && (
                  <span className="flex items-center gap-1 text-muted">
                    <Star size={9} aria-hidden />
                    {starsLabel}
                  </span>
                )}
              </a>
            </div>
            <Link
              href="/docs/install"
              onClick={() => setOpen(false)}
              className="mt-2 flex items-center justify-center gap-2 rounded-md px-3 py-2.5 text-center font-semibold"
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
