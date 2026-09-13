"use client";

import { useState, useEffect } from "react";
import Image from "next/image";
import Link from "next/link";
import { Download, Github, Menu, Star, X } from "lucide-react";
import { formatCount } from "@/lib/stats";

const links = [
  { href: "/#how-it-works", label: "How it works" },
  { href: "/features", label: "Features" },
  { href: "/#how-to-use", label: "How to use" },
  { href: "/#backends", label: "Backends" },
  { href: "/#proof", label: "Proof" },
  { href: "/#compatibility", label: "Compatibility" },
  { href: "/testing", label: "Testing" },
  { href: "/about", label: "About" },
  { href: "/docs", label: "Docs" },
];

const PH_URL =
  "https://www.producthunt.com/products/warden-6?utm_source=other&utm_medium=social";

type Stats = { stars: number | null; downloads: number | null };

export default function Nav() {
  const [open, setOpen] = useState(false);
  const [pillOpen, setPillOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const [phOpen, setPhOpen] = useState(true);
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

  // Show the floating pill navbar once the page header has scrolled away.
  useEffect(() => {
    function onScroll() {
      setScrolled(window.scrollY > 80);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const starsLabel = stats.stars !== null ? formatCount(stats.stars) : "5";
  const downloadsLabel =
    stats.downloads !== null ? `${formatCount(stats.downloads)}/mo` : "1.7k/mo";

  const logo = (
    <Image
      src="/logo.png"
      alt=""
      width={28}
      height={28}
      className="rounded-full"
      priority
    />
  );

  return (
    <>
      {/* ── Product Hunt announcement banner ── */}
      {phOpen && (
        <div className="relative bg-[#D8F3E6] text-[#0A2E1F]">
          <div className="mx-auto flex max-w-display items-center justify-center gap-3 px-10 py-2 text-center">
            <p className="text-[0.8125rem] font-semibold leading-snug sm:text-[0.875rem]">
              Warden is featured on{" "}
              <a
                href={PH_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="underline decoration-[#0A2E1F]/40 underline-offset-2 hover:decoration-[#0A2E1F]"
              >
                Product Hunt
              </a>
            </p>
            <a
              href={PH_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="shrink-0 rounded-md bg-[#4B40EE] px-3 py-1 text-[0.75rem] font-semibold text-white shadow-sm transition-colors hover:bg-[#3D33D6] sm:text-[0.8125rem]"
            >
              Vote for us
            </a>
          </div>
          <button
            onClick={() => setPhOpen(false)}
            aria-label="Dismiss announcement"
            className="absolute right-3 top-1/2 flex h-5 w-5 -translate-y-1/2 items-center justify-center rounded-full bg-white/70 text-[#4B5563] transition-colors hover:bg-white hover:text-[#111827]"
          >
            <X size={12} strokeWidth={2.5} />
          </button>
        </div>
      )}

      {/* ── Static page header ── */}
      <header className="border-b border-ink-700">
        <div className="mx-auto flex max-w-display items-center justify-between gap-6 px-6 py-5">
          <Link href="/" className="flex shrink-0 items-center gap-2.5" aria-label="Warden home">
            {logo}
            <span className="text-[1.0625rem] font-medium text-paper">warden</span>
          </Link>

          <nav className="hidden items-center gap-6 lg:flex">
            {links.map((l) => (
              <a
                key={l.href}
                href={l.href}
                className="whitespace-nowrap text-[0.9375rem] text-muted transition-colors hover:text-paper"
              >
                {l.label}
              </a>
            ))}
          </nav>

          <div className="flex items-center gap-2.5">
            {/* npm quick link */}
            <a
              href="https://www.npmjs.com/package/warden-sandbox-cli"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden items-center gap-1.5 whitespace-nowrap rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
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
              <span className="flex items-center gap-1 rounded-full bg-ink-800 px-1.5 py-0.5 text-[0.625rem] text-muted">
                <Download size={10} />
                {downloadsLabel}
              </span>
            </a>
            {/* GitHub quick link */}
            <a
              href="https://github.com/Prof-bilal/Warden"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden items-center gap-1.5 whitespace-nowrap rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
              aria-label="Warden on GitHub"
            >
              <Github size={13} className="text-muted" />
              <span className="font-medium">Warden</span>
              <span className="flex items-center gap-1 rounded-full bg-ink-800 px-1.5 py-0.5 text-[0.625rem] text-muted">
                <Star size={10} />
                {starsLabel}
              </span>
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
                  className="whitespace-nowrap text-[0.9375rem] text-muted transition-colors hover:text-paper"
                >
                  {l.label}
                </a>
              ))}
            </div>
          </nav>
        )}
      </header>

      {/* ── Floating pill navbar (appears on scroll) ── */}
      <div
        className={
          "fixed inset-x-0 top-2 z-50 transition-all duration-300 " +
          (scrolled
            ? "translate-y-0 opacity-100"
            : "pointer-events-none -translate-y-4 opacity-0")
        }
      >
        <div className="mx-auto max-w-display px-4">
          <div className="relative flex items-center justify-between gap-3 rounded-2xl border border-ink-700 bg-ink-950/90 py-2 pl-4 pr-2 shadow-[0_8px_30px_rgba(0,0,0,0.45)] backdrop-blur-md">
            {/* Left: logo + wordmark */}
            <Link
              href="/"
              className="flex shrink-0 items-center gap-2"
              aria-label="Warden home"
              onClick={() => setPillOpen(false)}
            >
              {logo}
              <span className="text-[0.9375rem] font-medium text-paper">warden</span>
            </Link>

            {/* Right: GitHub, npm, hamburger */}
            <div className="flex items-center gap-1.5">
              <a
                href="https://github.com/Prof-bilal/Warden"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Warden on GitHub"
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                <Github size={16} />
              </a>
              <a
                href="https://www.npmjs.com/package/warden-sandbox-cli"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="warden-sandbox-cli on npm"
                className="hidden h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper sm:flex"
              >
                <Download size={15} />
              </a>
              <a
                href="/docs/install"
                className="mr-1 hidden rounded-lg border border-ink-600 px-3 py-1.5 text-[0.75rem] font-medium text-paper transition-colors hover:border-blueprint/50 md:block"
              >
                Install
              </a>
              <button
                onClick={() => setPillOpen(!pillOpen)}
                aria-label={pillOpen ? "Close menu" : "Open menu"}
                aria-expanded={pillOpen}
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                {pillOpen ? <X size={17} /> : <Menu size={17} />}
              </button>
            </div>

            {/* Dropdown menu */}
            {pillOpen && (
              <nav className="absolute right-0 top-[calc(100%+8px)] w-60 rounded-2xl border border-ink-700 bg-ink-950/95 p-2 shadow-[0_16px_40px_rgba(0,0,0,0.55)] backdrop-blur-md">
                {links.map((l) => (
                  <a
                    key={l.href}
                    href={l.href}
                    onClick={() => setPillOpen(false)}
                    className="block rounded-lg px-3 py-2 text-[0.875rem] text-muted transition-colors hover:bg-ink-800 hover:text-paper"
                  >
                    {l.label}
                  </a>
                ))}
                <div className="mt-2 border-t border-ink-800 pt-2 md:hidden">
                  <a
                    href="/docs/install"
                    onClick={() => setPillOpen(false)}
                    className="block rounded-lg px-3 py-2 text-[0.875rem] font-medium text-blueprint transition-colors hover:bg-ink-800"
                  >
                    Install Warden
                  </a>
                </div>
              </nav>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
