"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import Link from "next/link";
import { ChevronDown, Download, Github, Menu, Star, X } from "lucide-react";
import { formatCount } from "@/lib/stats";

const productLinks = [
  { href: "/policies", label: "Policy packs" },
  { href: "/integrations", label: "Connections" },
  { href: "/testing", label: "Testing" },
  { href: "/blog", label: "Blog" },
  { href: "/guides", label: "Guides" },
];

const primaryLinks = [
  { href: "/setup", label: "Setup" },
  { href: "/compatibility", label: "Compatibility" },
  { href: "/docs", label: "Docs" },
];

type Stats = { stars: number | null; downloads: number | null };

export default function Nav() {
  const [open, setOpen] = useState(false);
  const [productOpen, setProductOpen] = useState(false);
  const [stats, setStats] = useState<Stats>({ stars: null, downloads: null });

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 15000);

    fetch("/api/stats", { signal: controller.signal })
      .then((response) => (response.ok ? response.json() : null))
      .then((data: Stats | null) => {
        if (cancelled || !data) return;
        const stars = typeof data.stars === "number" && data.stars >= 0 ? data.stars : null;
        const downloads = typeof data.downloads === "number" && data.downloads >= 0 ? data.downloads : null;
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

  const logo = (
    <Image src="/logo.png" alt="" width={28} height={28} className="rounded-full" priority />
  );
  const downloadsLabel = stats.downloads === null ? null : formatCount(stats.downloads);
  const starsLabel = stats.stars === null ? null : formatCount(stats.stars);

  function closeMenu() {
    setOpen(false);
    setProductOpen(false);
  }

  return (
    <header className="border-b border-ink-800 bg-ink-950/95">
      <div className="mx-auto flex h-[72px] max-w-display items-center justify-between gap-6 px-5 sm:px-6">
        <Link href="/" className="flex shrink-0 items-center gap-2.5" aria-label="Warden home">
          {logo}
          <span className="font-sans text-[1.05rem] font-semibold tracking-[-0.02em] text-paper">warden</span>
        </Link>

        <nav className="hidden items-center gap-7 lg:flex" aria-label="Primary navigation">
          {primaryLinks.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="font-sans text-[0.875rem] text-muted transition-colors hover:text-paper"
            >
              {link.label}
            </Link>
          ))}
          <div className="relative">
            <button
              type="button"
              onClick={() => setProductOpen((value) => !value)}
              aria-expanded={productOpen}
              aria-haspopup="menu"
              className="flex items-center gap-1 font-sans text-[0.875rem] text-muted transition-colors hover:text-paper"
            >
              Product
              <ChevronDown size={14} className={productOpen ? "rotate-180 transition-transform" : "transition-transform"} />
            </button>
            {productOpen && (
              <div className="absolute left-1/2 top-[calc(100%+14px)] z-50 w-48 -translate-x-1/2 rounded-lg border border-ink-700 bg-ink-950 p-2 shadow-[0_16px_40px_rgba(0,0,0,0.45)]" role="menu">
                {productLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    onClick={closeMenu}
                    className="block rounded-md px-3 py-2 font-sans text-[0.875rem] text-muted transition-colors hover:bg-ink-800 hover:text-paper"
                    role="menuitem"
                  >
                    {link.label}
                  </Link>
                ))}
              </div>
            )}
          </div>
        </nav>

        <div className="flex items-center gap-2">
          <a
            href="https://www.npmjs.com/package/warden-sandbox-cli"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-md border border-ink-700 bg-ink-900 px-2.5 py-1.5 font-sans text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
            aria-label="warden-sandbox-cli on npm"
          >
            <span className="font-medium">npm</span>
            {downloadsLabel !== null && (
              <span className="flex items-center gap-1 text-[0.625rem] text-muted">
                <Download size={10} />
                {downloadsLabel}
              </span>
            )}
          </a>
          <a
            href="https://github.com/Prof-bilal/Warden"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-md border border-ink-700 bg-ink-900 px-2.5 py-1.5 font-sans text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
            aria-label="Warden on GitHub"
          >
            <Github size={13} className="text-muted" />
            <span className="font-medium">GitHub</span>
            {starsLabel !== null && (
              <span className="flex items-center gap-1 text-[0.625rem] text-muted">
                <Star size={10} />
                {starsLabel}
              </span>
            )}
          </a>
          <Link
            href="/docs/install"
            className="hidden rounded-md bg-blueprint px-3.5 py-2 font-sans text-[0.75rem] font-semibold text-ink-950 transition-colors hover:bg-blueprint/90 md:block"
          >
            Install
          </Link>
          <button
            type="button"
            onClick={() => setOpen((value) => !value)}
            className="flex items-center justify-center rounded-md border border-ink-700 p-2 text-muted transition-colors hover:bg-ink-800 hover:text-paper lg:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
          >
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {open && (
        <nav className="border-t border-ink-800 px-5 py-4 lg:hidden" aria-label="Mobile navigation">
          <div className="mx-auto flex max-w-display flex-col gap-1">
            {primaryLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                onClick={closeMenu}
                className="rounded-md px-3 py-2.5 font-sans text-sm text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                {link.label}
              </Link>
            ))}
            <button
              type="button"
              onClick={() => setProductOpen((value) => !value)}
              aria-expanded={productOpen}
              className="flex items-center justify-between rounded-md px-3 py-2.5 text-left font-sans text-sm text-muted transition-colors hover:bg-ink-800 hover:text-paper"
            >
              Product
              <ChevronDown size={15} className={productOpen ? "rotate-180 transition-transform" : "transition-transform"} />
            </button>
            {productOpen && (
              <div className="ml-3 border-l border-ink-700 pl-3">
                {productLinks.map((link) => (
                  <Link
                    key={link.href}
                    href={link.href}
                    onClick={closeMenu}
                    className="block rounded-md px-3 py-2 font-sans text-sm text-muted transition-colors hover:bg-ink-800 hover:text-paper"
                  >
                    {link.label}
                  </Link>
                ))}
              </div>
            )}
            <Link
              href="/docs/install"
              onClick={closeMenu}
              className="mt-2 rounded-md bg-blueprint px-3 py-2.5 text-center font-sans text-sm font-semibold text-ink-950"
            >
              Install Warden
            </Link>
          </div>
        </nav>
      )}
    </header>
  );
}
