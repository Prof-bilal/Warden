"use client";
import { useState } from "react";
import { clients } from "@/lib/ecosystem";
const events = [
  "setup-completed",
  "setup-failed",
  "first-protected-task",
  "repeat-use",
  "rollback",
  "badge-activation",
];
export default function LocalPilotReport() {
  const [summary, setSummary] = useState<{
    counts: Record<string, number>;
    median: number;
    failure: number;
    records: number;
  } | null>(null);
  const [error, setError] = useState("");
  const load = async (f?: File) => {
    setSummary(null);
    setError("");
    try {
      if (!f || f.size > 1048576)
        throw Error("Choose a local report smaller than 1 MiB.");
      const lines = (await f.text()).trim().split("\n");
      if (lines.length > 10000) throw Error("Report exceeds 10,000 events.");
      const counts: Record<string, number> = {};
      const durations: number[] = [];
      for (const line of lines) {
        const r = JSON.parse(line);
        if (
          !events.includes(r.event) ||
          !clients.some((c) => c.id === r.client) ||
          !Number.isInteger(r.seconds) ||
          r.seconds < 0 ||
          r.seconds > 86400 ||
          !Number.isFinite(Date.parse(r.time)) ||
          Object.keys(r).some(
            (k) => !["time", "client", "event", "seconds"].includes(k),
          )
        )
          throw Error("Report must contain only valid local event records.");
        counts[r.event] = (counts[r.event] ?? 0) + 1;
        if (r.event === "setup-completed") durations.push(r.seconds);
      }
      durations.sort((a, b) => a - b);
      const n = durations.length;
      const median = n
        ? n % 2
          ? durations[Math.floor(n / 2)]
          : (durations[n / 2 - 1] + durations[n / 2]) / 2
        : 0;
      const attempts =
        (counts["setup-completed"] ?? 0) + (counts["setup-failed"] ?? 0);
      setSummary({
        counts,
        median,
        failure: attempts ? (counts["setup-failed"] ?? 0) / attempts : 0,
        records: lines.length,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Report could not be read.");
    }
  };
  return (
    <section className="rounded-2xl border border-ink-700 p-6 sm:p-8">
      <h2 className="text-2xl font-semibold">Your local pilot report</h2>
      <p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted">
        Import events you explicitly recorded with the CLI. This page totals
        them locally. Event counts do not represent unique users or verified
        activations.
      </p>
      <input
        type="file"
        accept=".jsonl,.json"
        onChange={(e) => void load(e.target.files?.[0])}
        className="mt-5 block w-full text-sm text-muted"
        aria-label="Local pilot event report"
      />
      {error && (
        <p role="status" className="mt-5 text-progress">
          {error}
        </p>
      )}
      {summary && (
        <>
          <div className="mt-7 grid gap-4 sm:grid-cols-3">
            {[
              [summary.records, "Recorded events"],
              [`${summary.median}s`, "Median completed setup"],
              [
                `${(summary.failure * 100).toFixed(1)}%`,
                "Recorded setup failure rate",
              ],
            ].map(([value, label]) => (
              <div key={label} className="rounded-xl bg-ink-800 p-5">
                <p className="text-2xl text-blueprint">{value}</p>
                <p className="mt-2 text-xs text-muted">{label}</p>
              </div>
            ))}
          </div>
          <dl className="mt-6 grid gap-3 sm:grid-cols-2">
            {events.map((event) => (
              <div
                key={event}
                className="flex justify-between border-b border-ink-700 py-3 text-sm"
              >
                <dt className="text-muted">{event.replaceAll("-", " ")}</dt>
                <dd>{summary.counts[event] ?? 0}</dd>
              </div>
            ))}
          </dl>
        </>
      )}
      <p className="mt-6 text-xs text-muted">
        Nothing is uploaded, saved, or collected automatically.
      </p>
    </section>
  );
}
