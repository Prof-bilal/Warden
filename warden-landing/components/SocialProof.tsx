// Social proof badges for the hero: npm link, live GitHub stars, live npm
// downloads. Client-side fetch against /api/stats (hourly-cached server
// route) so static export/ISR never blocks on upstream APIs: every badge
// renders a static label first and only swaps in the live count on success.
// Any failure (network, 4xx/5xx, malformed body) keeps the static labels.
"use client";

import { useEffect, useState } from "react";
import { Download, Github, Star } from "lucide-react";
import { formatCount } from "@/lib/stats";

const NPM_URL = "https://www.npmjs.com/package/warden-sandbox-cli";
const GITHUB_URL = "https://github.com/Prof-bilal/Warden";

type Stats = { stars: number | null; downloads: number | null };

const pill =
  "inline-flex items-center gap-2 rounded-full border border-ink-700 bg-ink-900 px-4 py-2 text-[0.8125rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800";

export default function SocialProof() {
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
      .catch(() => {
        // Silent by design: badges keep their static labels.
      })
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
    <div className="mt-6 flex flex-wrap items-center gap-3">
      <a
        href="https://www.producthunt.com/products/warden-6?utm_source=other&utm_medium=social"
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-2.5 rounded-full border border-ink-700 bg-ink-900 px-4 py-2 text-[0.8125rem] text-paper transition-colors hover:border-blueprint/50 hover:bg-ink-800"
      >
        <span className="text-base">🏷️</span>
        <span className="leading-tight">
          <span className="block text-[0.625rem] uppercase tracking-[0.08em] text-muted">
            Featured on
          </span>
          <span className="font-medium">Product Hunt</span>
        </span>
      </a>
      <a
        href={NPM_URL}
        target="_blank"
        rel="noopener noreferrer"
        aria-label="warden-sandbox-cli on npm"
        className={pill}
      >
        {/* npm glyph: official logotype mark, currentColor so it tracks text */}
        <svg
          width={16}
          height={16}
          viewBox="0 0 24 24"
          fill="currentColor"
          aria-hidden="true"
          className="text-muted"
        >
          <path d="M0 0v16.8h6.8v3.5h3.4v-3.5h3.4v3.5h3.5v-3.5H24V0H0Zm20.5 13.4h-5.1V3.5h-2.8v9.9H7.4V3.5H3.4v9.9h17.1Z" />
        </svg>
        <span className="font-medium">npm</span>
        {downloadsLabel !== null && (
          <span className="flex items-center gap-1 rounded-full bg-ink-800 px-2 py-0.5 text-[0.6875rem] text-muted">
            <Download size={11} />
            {downloadsLabel}
          </span>
        )}
      </a>
      <a
        href={GITHUB_URL}
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Warden on GitHub"
        className={pill}
      >
        <Github size={14} className="text-muted" />
        <span className="font-medium">Warden</span>
        <span className="flex items-center gap-1 rounded-full bg-ink-800 px-2 py-0.5 text-[0.6875rem] text-muted">
          <Star size={11} />
          {starsLabel ?? "Star"}
        </span>
      </a>
    </div>
  );
}
