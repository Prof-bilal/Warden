"use client";
import { useState } from "react";
import Link from "next/link";
import { ArrowRight, Search, FolderLock, Globe, KeyRound } from "lucide-react";
import { policyPacks } from "@/lib/ecosystem";

export default function PolicyCatalog() {
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("All profiles");
  const visible = policyPacks.filter(
    (p) =>
      (category === "All profiles" || p.category === category) &&
      `${p.name} ${p.artifact} ${p.profile}`
        .toLowerCase()
        .includes(query.toLowerCase()),
  );
  return (
    <>
      <div className="mb-8 rounded-xl border border-progress/30 bg-progress-subtle p-5 text-sm leading-relaxed text-muted">
        <span className="font-semibold text-progress">Candidate profiles.</span>{" "}
        Upstream identity and version metadata are recorded. Runtime preparation
        and allowed/blocked workflow checks are still required before a profile
        can be called verified.{" "}
        <Link
          href="/protection"
          className="text-blueprint underline underline-offset-4"
        >
          See the evidence boundary
        </Link>
        .
      </div>
      <div className="mb-8 flex flex-col gap-4 sm:flex-row">
        <label className="relative flex-1">
          <span className="sr-only">Search policy packs</span>
          <Search size={18} className="absolute left-4 top-3.5 text-muted" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search by server or task…"
            className="w-full rounded-xl border border-ink-700 bg-ink-900 py-3 pl-11 pr-4 text-sm"
          />
        </label>
        <label>
          <span className="sr-only">Filter by category</span>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="w-full rounded-xl border border-ink-700 bg-ink-900 px-4 py-3 text-sm sm:w-56"
          >
            {[
              "All profiles",
              ...new Set(policyPacks.map((p) => p.category)),
            ].map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </label>
      </div>
      <p role="status" className="mb-4 text-xs text-muted">
        {visible.length} profiles · metadata reviewed September 30, 2026
      </p>
      <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
        {visible.map((p) => (
          <article
            key={p.id}
            className="flex flex-col rounded-2xl border border-ink-700 bg-ink-900 p-6 transition-colors hover:border-blueprint/50"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted">{p.category}</span>
              <span className="rounded-full border border-progress/30 px-2.5 py-1 text-[10px] uppercase tracking-wide text-progress">
                Candidate
              </span>
            </div>
            <h2 className="mt-6 text-2xl font-semibold">{p.name}</h2>
            <p className="mt-2 min-h-12 text-sm leading-relaxed text-muted">
              {p.profile}
            </p>
            <div className="my-6 space-y-3 border-y border-ink-700 py-5 text-xs text-muted">
              <p className="flex items-center gap-2">
                <FolderLock size={15} />
                {p.pathMode === "none"
                  ? "Prepared runtime only"
                  : p.pathMode === "read"
                    ? "Selected directory · read only"
                    : "Selected storage · read & write"}
              </p>
              <p className="flex items-center gap-2">
                <Globe size={15} />
                {p.hosts.length
                  ? `${p.hosts.length} explicit API host`
                  : "Network denied"}
              </p>
              <p className="flex items-center gap-2">
                <KeyRound size={15} />
                {p.env.length
                  ? `${p.env.length} allowed variable name`
                  : "No environment variables"}
              </p>
            </div>
            <p className="break-all font-mono text-[10px] text-muted">
              {p.artifact}@{p.version}
            </p>
            <Link
              href={`/policies/${p.id}`}
              className="mt-6 flex items-center justify-between text-sm font-medium text-blueprint"
            >
              Review profile <ArrowRight size={16} />
            </Link>
          </article>
        ))}
      </div>
      {!visible.length && (
        <div className="rounded-xl border border-dashed border-ink-700 p-10 text-center text-muted">
          No profiles match. Try a different server or category.
        </div>
      )}
    </>
  );
}
