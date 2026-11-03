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
    <div className="sticky top-[64px] z-30 border-y border-hair bg-void/95 backdrop-blur">
      <div className="flex items-center gap-5 px-6 py-3">
        {/* Label */}
        <span className="hidden shrink-0 font-mono text-[9.5px] uppercase tracking-micro text-offwhite/35 sm:block">
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
                "shrink-0 rounded border px-3 py-1 font-mono text-[10.5px] uppercase tracking-micro transition-colors " +
                (active === f.id
                  ? "border-acid/40 bg-acid/10 text-acid"
                  : "border-hair text-offwhite/40 hover:border-offwhite/20 hover:text-offwhite")
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
                  (i <= activeIndex ? "bg-acid" : "bg-hair")
                }
              />
            ))}
          </div>
          <span className="font-mono text-[9.5px] tabular-nums text-offwhite/35">
            {String(activeIndex + 1).padStart(2, "0")}/
            {String(FEATURES.length).padStart(2, "0")}
          </span>
        </div>
      </div>
    </div>
  );
}
