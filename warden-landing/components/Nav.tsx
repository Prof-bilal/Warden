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
    const timer = setTimeout(() => controller.abort(), 15000);
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
    stats.downloads !== null ? formatCount(stats.downloads) : "1.9k";

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
            {/* Discord */}
            <a
              href="https://discord.gg/Mx5BhwNP4"
              target="_blank"
              rel="noopener noreferrer"
              className="hidden items-center gap-1.5 whitespace-nowrap rounded-full border border-ink-700 bg-ink-900 px-3 py-1.5 text-[0.75rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800 sm:inline-flex"
              aria-label="Warden on Discord"
            >
              <svg
                width={13}
                height={13}
                viewBox="0 0 24 24"
                fill="currentColor"
                aria-hidden="true"
                className="text-muted"
              >
                <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561 19.9312 19.9312 0 005.9932 3.0397.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286 19.8975 19.8975 0 006.0023-3.0397.0771.0771 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189z" />
              </svg>
              <span className="font-medium">Discord</span>
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

            {/* Right: Discord, GitHub, npm, hamburger */}
            <div className="flex items-center gap-1.5">
              <a
                href="https://discord.gg/Mx5BhwNP4"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="Warden on Discord"
                className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-ink-800 hover:text-paper"
              >
                <svg
                  width={16}
                  height={16}
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561 19.9312 19.9312 0 005.9932 3.0397.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286 19.8975 19.8975 0 006.0023-3.0397.0771.0771 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189z" />
                </svg>
              </a>
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
                  <a
                    href="https://discord.gg/Mx5BhwNP4"
                    target="_blank"
                    rel="noopener noreferrer"
                    onClick={() => setPillOpen(false)}
                    className="flex items-center gap-2 rounded-lg px-3 py-2 text-[0.875rem] text-muted transition-colors hover:bg-ink-800 hover:text-paper"
                  >
                    <svg
                      width={14}
                      height={14}
                      viewBox="0 0 24 24"
                      fill="currentColor"
                    >
                      <path d="M20.317 4.3698a19.7913 19.7913 0 00-4.8851-1.5152.0741.0741 0 00-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 00-.0785-.037 19.7363 19.7363 0 00-4.8852 1.515.0699.0699 0 00-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 00.0312.0561 19.9312 19.9312 0 005.9932 3.0397.0777.0777 0 00.0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 00-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 01-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 01.0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 01.0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 01-.0066.1276 12.2986 12.2986 0 01-1.873.8914.0766.0766 0 00-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 00.0842.0286 19.8975 19.8975 0 006.0023-3.0397.0771.0771 0 00.0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 00-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189z" />
                    </svg>
                    Discord
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
