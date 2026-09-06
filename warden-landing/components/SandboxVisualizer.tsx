"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";

type GrantKey = "data" | "output" | "github";

type Attempt = {
  id: number;
  action: string;
  target: string;
  requires: GrantKey | null;
};

const CYCLE: Omit<Attempt, "id">[] = [
  { action: "read", target: "./data/report.json", requires: "data" },
  { action: "read", target: "~/.ssh/id_rsa", requires: null },
  { action: "connect", target: "api.github.com", requires: "github" },
  { action: "write", target: "./output/summary.md", requires: "output" },
  { action: "connect", target: "evil-telemetry.example", requires: null },
];

const GRANT_LABELS: Record<GrantKey, string> = {
  data: "./data — read",
  output: "./output — write",
  github: "api.github.com — network",
};

export default function SandboxVisualizer() {
  const [grants, setGrants] = useState<Record<GrantKey, boolean>>({
    data: true,
    output: false,
    github: true,
  });
  const [log, setLog] = useState<(Attempt & { granted: boolean })[]>([]);
  const grantsRef = useRef(grants);
  const cursorRef = useRef(0);
  const idRef = useRef(0);
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    grantsRef.current = grants;
  }, [grants]);

  useEffect(() => {
    const interval = setInterval(() => {
      const next = CYCLE[cursorRef.current % CYCLE.length];
      cursorRef.current += 1;
      idRef.current += 1;
      const granted = next.requires ? grantsRef.current[next.requires] : false;
      setLog((prev) => [{ ...next, id: idRef.current, granted }, ...prev].slice(0, 6));
    }, 2200);
    return () => clearInterval(interval);
  }, []);

  function toggle(key: GrantKey) {
    setGrants((g) => ({ ...g, [key]: !g[key] }));
  }

  return (
    <div className="grid gap-px overflow-hidden rounded-sm border border-ink-700 bg-ink-700 md:grid-cols-[16rem_1fr]">
      <div className="bg-ink-900 p-5">
        <p className="text-[0.8125rem] text-muted">Policy</p>
        <div className="mt-3 space-y-2">
          {(Object.keys(GRANT_LABELS) as GrantKey[]).map((key) => (
            <button
              key={key}
              onClick={() => toggle(key)}
              className="flex w-full items-center gap-2.5 rounded-sm border border-ink-700 px-3 py-2 text-left transition-colors hover:border-ink-600"
            >
              <span
                className={
                  "h-2 w-2 shrink-0 rounded-full transition-colors " +
                  (grants[key] ? "bg-grant" : "bg-ink-600")
                }
              />
              <span className="truncate font-mono text-[0.8125rem] text-paper">
                {GRANT_LABELS[key]}
              </span>
            </button>
          ))}
        </div>
        <p className="mt-4 text-[0.75rem] leading-[1.5] text-muted">
          Toggle a grant and watch the matching attempt change on the right.
          Two of the five attempts are never in the policy — they stay
          denied no matter what.
        </p>
      </div>

      <div className="flex h-[19rem] flex-col bg-ink-950 p-5">
        <p className="text-[0.8125rem] text-muted">Access attempts</p>
        <div className="mt-3 flex-1 space-y-2 overflow-hidden">
          <AnimatePresence initial={false}>
            {log.map((entry) => (
              <motion.div
                key={entry.id}
                initial={reduceMotion ? false : { opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.3 }}
                className="flex items-center justify-between gap-3 border-b border-ink-800 pb-2 font-mono text-[0.8125rem]"
              >
                <span className="truncate text-paper">
                  <span className="text-muted">{entry.action} </span>
                  {entry.target}
                </span>
                <span
                  className={
                    "shrink-0 px-2 py-0.5 text-[0.6875rem] " +
                    (entry.granted
                      ? "rounded-sm bg-grant-subtle text-grant"
                      : "rounded-none border border-deny/40 bg-deny-subtle text-deny")
                  }
                >
                  {entry.granted ? "granted" : "denied"}
                </span>
              </motion.div>
            ))}
          </AnimatePresence>
          {log.length === 0 && (
            <p className="font-mono text-[0.8125rem] text-muted">
              Waiting for the first attempt…
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
