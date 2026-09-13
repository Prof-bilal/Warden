"use client";

import { useRef, useState } from "react";

export type StepperItem = {
  title: string;
  summary: string;
  body: string;
  image: React.ReactNode;
};

type Props = {
  sectionLabel: string;
  items: StepperItem[];
};

export default function StepperDetail({ sectionLabel, items }: Props) {
  const [active, setActive] = useState(0);
  const [mobileIdx, setMobileIdx] = useState(0);
  const trackRef = useRef<HTMLDivElement>(null);
  const total = items.length;

  // ── Mobile carousel: derive the active slide from scroll position ──
  function onTrackScroll() {
    const el = trackRef.current;
    if (!el || el.clientWidth === 0) return;
    const idx = Math.round(el.scrollLeft / el.clientWidth);
    setMobileIdx(Math.min(total - 1, Math.max(0, idx)));
  }

  function goTo(i: number) {
    const el = trackRef.current;
    setMobileIdx(i);
    el?.scrollTo({ left: i * el.clientWidth, behavior: "smooth" });
  }

  const item = items[active];

  return (
    <div>
      {/* ══════════ Mobile: swipeable image carousel (lg:hidden) ══════════ */}
      <div className="lg:hidden">
        <span className="inline-block rounded-full border border-ink-700 bg-ink-900 px-3 py-1 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
          {sectionLabel}
        </span>

        {/* Snap carousel: one card per step, swipes horizontally */}
        <div
          ref={trackRef}
          onScroll={onTrackScroll}
          className="mt-3 flex snap-x snap-mandatory gap-3 overflow-x-auto rounded-[12px] [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          role="region"
          aria-label={`${sectionLabel} carousel — swipe to browse`}
        >
          {items.map((s, i) => (
            <div
              key={i}
              className="w-full shrink-0 snap-center rounded-[12px] border border-ink-700 bg-ink-900 p-4"
              aria-hidden={i !== mobileIdx}
            >
              <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
                Step {i + 1} of {total}
              </span>
              {/* Each slide renders its own image — all slides stay mounted,
                  so swiping never waits on an image load. */}
              <div className="mt-3 overflow-hidden rounded-[8px] border border-ink-700 bg-ink-950 p-1.5">
                {s.image}
              </div>
              <h3 className="mt-4 text-[1.25rem] font-medium leading-[1.2] tracking-[-0.01em] text-paper">
                {s.title}
              </h3>
              <p className="mt-2 text-[0.875rem] leading-[1.6] text-muted">
                {s.body}
              </p>
            </div>
          ))}
        </div>

        {/* Dots + counter */}
        <div className="mt-4 flex items-center justify-between">
          <div className="flex gap-1.5">
            {items.map((_, i) => (
              <button
                key={i}
                onClick={() => goTo(i)}
                aria-label={`Go to step ${i + 1}: ${items[i].title}`}
                className={
                  "h-1.5 rounded-full transition-all " +
                  (i === mobileIdx ? "w-6 bg-blueprint" : "w-1.5 bg-ink-600")
                }
              />
            ))}
          </div>
          <span className="font-mono text-[0.6875rem] tabular-nums text-muted">
            {String(mobileIdx + 1).padStart(2, "0")}/
            {String(total).padStart(2, "0")}
          </span>
        </div>

        {/* Quick-jump chips (compact titles) */}
        <div className="mt-3 flex gap-1.5 overflow-x-auto pb-1 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          {items.map((s, i) => (
            <button
              key={i}
              onClick={() => goTo(i)}
              className={
                "shrink-0 rounded-full border px-2.5 py-1 font-mono text-[0.6875rem] transition-colors " +
                (i === mobileIdx
                  ? "border-blueprint/40 bg-blueprint/10 text-blueprint"
                  : "border-ink-700 text-muted")
              }
            >
              {String(i + 1).padStart(2, "0")}
            </button>
          ))}
        </div>
      </div>

      {/* ══════════ Desktop: stepper list + detail panel (lg+) ══════════ */}
      <div className="hidden gap-8 lg:grid lg:grid-cols-[26rem_1fr]">
        {/* Left: stepper list */}
        <div className="flex flex-col gap-2">
          <span className="mb-2 inline-block w-fit rounded-full border border-ink-700 bg-ink-900 px-3 py-1 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            {sectionLabel}
          </span>
          {items.map((s, i) => (
            <button
              key={i}
              onClick={() => setActive(i)}
              className={
                "flex items-center gap-4 rounded-[12px] border px-5 py-4 text-left transition-all " +
                (i === active
                  ? "border-blueprint/40 bg-ink-900 shadow-[0_0_0_1px_rgba(110,147,232,0.15)]"
                  : "border-ink-700 bg-ink-950 opacity-50 hover:opacity-75")
              }
            >
              <span
                className={
                  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border font-mono text-[0.75rem] " +
                  (i === active
                    ? "border-blueprint/40 bg-blueprint/10 text-blueprint"
                    : "border-ink-600 text-muted")
                }
              >
                {String(i + 1).padStart(2, "0")}
              </span>
              <div className="min-w-0">
                <span
                  className={
                    "block truncate text-[1rem] font-medium " +
                    (i === active ? "text-paper" : "text-muted")
                  }
                >
                  {s.title}
                </span>
                <span className="mt-0.5 block truncate text-[0.8125rem] leading-[1.4] text-muted">
                  {s.summary}
                </span>
              </div>
            </button>
          ))}
        </div>

        {/* Right: detail panel */}
        <div className="flex flex-col rounded-[12px] border border-ink-700 bg-ink-900 p-8">
          <span className="font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
            Step {active + 1} of {total}
          </span>
          <h3 className="mt-2 text-[1.5rem] font-medium leading-[1.2] tracking-[-0.01em] text-paper">
            {item.title}
          </h3>
          <p className="mt-2 text-[0.9375rem] leading-[1.6] text-muted">
            {item.body}
          </p>

          {/* Image panel — all images stay mounted and stacked in one grid
              cell so every tab switch is instant (no per-click image load). */}
          <div className="mt-6 grid min-h-[200px] flex-1 overflow-hidden rounded-[8px] border border-ink-700 bg-ink-950 p-2.5 sm:min-h-[240px] lg:min-h-[280px]">
            {items.map((s, i) => (
              <div
                key={i}
                aria-hidden={i !== active}
                className={
                  "[grid-area:1/1] flex items-center justify-center transition-opacity duration-200 " +
                  (i === active ? "opacity-100" : "pointer-events-none opacity-0")
                }
              >
                {s.image}
              </div>
            ))}
          </div>

          {/* Progress dots */}
          <div className="mt-6 flex gap-2">
            {items.map((_, i) => (
              <div
                key={i}
                className={
                  "h-1 flex-1 rounded-full transition-colors " +
                  (i <= active ? "bg-blueprint" : "bg-ink-700")
                }
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
