"use client";

import { useState } from "react";
import Link from "next/link";
import { Download, Github, Menu, Star, X } from "lucide-react";
import { formatCount } from "@/lib/stats";
import { useEffect } from "react";

const links = [
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#how-to-use", label: "How to use" },
  { href: "/#backends", label: "Backends" },
  { href: "/#proof", label: "Proof" },
  { href: "/#compatibility", label: "Compatibility" },
  { href: "/testing", label: "Testing" },
  { href: "/about", label: "About" },
  { href: "/docs", label: "Docs" },
];

type Stats = { stars: number | null; downloads: number | null };

export default function Nav() {
  const [open, setOpen] = useState(false);
  const [stats, setStats] = useState<Stats>({ stars: null, downloads: null });

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 8000);
    fetch("/api/stats", { signal: controller.signal })
      .then((res) => {
        if (!res.ok) return;
        return res.json() as Promise<Stats>;
      })
      .then((data) => {
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

  const starsLabel = stats.stars !== null ? formatCount(stats.stars) : null;
  const downloadsLabel =
    stats.downloads !== null ? `${formatCount(stats.downloads)}/mo` : null;

  return (
    <header className="border-b border-ink-700">
      <div className="mx-auto flex max-w-content items-center justify-between px-6 py-5">
        <Link href="#" className="flex items-center gap-2.5">
          <svg width="22" height="22" viewBox="0 0 22 22" fill="none" aria-hidden="true">
            <rect x="1" y="1" width="6" height="20" rx="1" stroke="#6E93E8" strokeWidth="1.4" />
            <rect x="15" y="1" width="6" height="20" rx="1" stroke="#6E93E8" strokeWidth="1.4" />
            <path d="M7 11H15" stroke="#3FB27E" strokeWidth="1.4" strokeDasharray="1.5 2.2" />
          </svg>
          <span className="text-[1.0625rem] font-medium text-paper">warden</span>
        </Link>

        <nav className="hidden items-center gap-8 lg:flex">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="text-[0.9375rem] text-muted transition-colors hover:text-paper"
            >
              {l.label}
            </a>
          ))}
        </nav>

        <div className="flex items-center gap-2.5">
          {/* Social proof badges */}
          <a
            href="https://www.producthunt.com/products/warden-6?utm_source=other&utm_medium=social"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 xl:inline-flex"
          >
            <span className="text-sm">🏷️</span>
            <span className="leading-tight">
              <span className="block text-[0.5625rem] uppercase tracking-[0.08em] text-muted">
                Featured on
              </span>
              <span className="font-medium">Product Hunt</span>
            </span>
          </a>
          <a
            href="https://www.npmjs.com/package/warden-sandbox-cli"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
            aria-label="warden-sandbox-cli on npm"
          >
            <svg
              width={13}
              height={13}
              viewBox="0 0 24 24"
              fill="currentColor"
              aria-hidden="true"
              className="text-muted"
            >
              <path d="M0 0v16.8h6.8v3.5h3.4v-3.5h3.4v3.5h3.5v-3.5H24V0H0Zm20.5 13.4h-5.1V3.5h-2.8v9.9H7.4V3.5H3.4v9.9h17.1Z" />
            </svg>
            <span className="font-medium">npm</span>
            {downloadsLabel !== null && (
              <span className="flex items-center gap-1 rounded-full bg-ink-800 px-1.5 py-0.5 text-[0.625rem] text-muted">
                <Download size={10} />
                {downloadsLabel}
              </span>
            )}
          </a>
          <a
            href="https://github.com/Prof-bilal/Warden"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden items-center gap-1.5 rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
            aria-label="Warden on GitHub"
          >
            <Github size={13} className="text-muted" />
            <span className="font-medium">Warden</span>
            <span className="flex items-center gap-1 rounded-full bg-ink-800 px-1.5 py-0.5 text-[0.625rem] text-muted">
              <Star size={10} />
              {starsLabel ?? "Star"}
            </span>
          </a>
          <a
            href="https://github.com/Prof-bilal/Warden"
            className="flex items-center gap-2 rounded-sm border border-ink-600 px-3.5 py-1.5 text-[0.875rem] text-paper transition-colors hover:border-blueprint hover:text-blueprint"
          >
            <Github size={15} />
            <span className="hidden sm:inline">GitHub</span>
          </a>
          <button
            onClick={() => setOpen(!open)}
            className="flex items-center justify-center rounded-sm border border-ink-600 p-1.5 text-muted transition-colors hover:text-paper md:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
          >
            {open ? <X size={18} /> : <Menu size={18} />}
          </button>
        </div>
      </div>

      {open && (
        <nav className="border-t border-ink-700 px-6 py-4 md:hidden">
          <div className="flex flex-col gap-3">
            {links.map((l) => (
              <a
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className="text-[0.9375rem] text-muted transition-colors hover:text-paper"
              >
                {l.label}
              </a>
            ))}
          </div>
        </nav>
      )}
    </header>
  );
}
