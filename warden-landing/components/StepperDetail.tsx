"use client";

import { useState } from "react";

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
  const total = items.length;
  const item = items[active];

  return (
    <div className="grid gap-6 lg:grid-cols-[26rem_1fr] lg:gap-8">
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

        {/* Image panel */}
        <div className="mt-6 flex flex-1 items-center justify-center overflow-hidden rounded-[8px] border border-ink-700 bg-ink-950 p-8">
          {item.image}
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
  );
}
