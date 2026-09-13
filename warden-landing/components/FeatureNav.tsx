"use client";

import { useEffect, useState } from "react";

const FEATURES = [
  { id: "run", label: "run" },
  { id: "init", label: "init" },
  { id: "trace", label: "trace" },
  { id: "logs", label: "logs" },
  { id: "doctor", label: "doctor" },
  { id: "gateway", label: "gateway" },
  { id: "proxy", label: "proxy" },
  { id: "k8s", label: "k8s" },
  { id: "update", label: "update" },
];

export default function FeatureNav() {
  const [active, setActive] = useState("run");

  useEffect(() => {
    function onScroll() {
      // Find the last section whose top has crossed the scroll marker.
      let current = FEATURES[0].id;
      for (const f of FEATURES) {
        const el = document.getElementById(f.id);
        if (el && el.getBoundingClientRect().top <= 140) {
          current = f.id;
        }
      }
      setActive(current);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const activeIndex = FEATURES.findIndex((f) => f.id === active);

  return (
    <div className="sticky top-[57px] z-30 border-y border-ink-800 bg-ink-950/95 backdrop-blur">
      <div className="mx-auto flex max-w-content items-center gap-5 px-6 py-3">
        {/* Label */}
        <span className="hidden shrink-0 font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted sm:block">
          Features
        </span>

        {/* Pills */}
        <nav
          className="flex flex-1 gap-1.5 overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          aria-label="Feature sections"
        >
          {FEATURES.map((f) => (
            <a
              key={f.id}
              href={`#${f.id}`}
              className={
                "shrink-0 rounded-full border px-3 py-1 font-mono text-[0.75rem] transition-colors " +
                (active === f.id
                  ? "border-blueprint/40 bg-blueprint/10 text-blueprint"
                  : "border-ink-700 text-muted hover:border-ink-600 hover:text-paper")
              }
            >
              {f.label}
            </a>
          ))}
        </nav>

        {/* Right: progress counter + bar */}
        <div className="hidden shrink-0 items-center gap-3 md:flex">
          <div className="flex gap-1">
            {FEATURES.map((f, i) => (
              <span
                key={f.id}
                className={
                  "h-1 w-3 rounded-full transition-colors " +
                  (i <= activeIndex ? "bg-blueprint" : "bg-ink-700")
                }
              />
            ))}
          </div>
          <span className="font-mono text-[0.6875rem] tabular-nums text-muted">
            {String(activeIndex + 1).padStart(2, "0")}/
            {String(FEATURES.length).padStart(2, "0")}
          </span>
        </div>
      </div>
    </div>
  );
}
