import { NextResponse } from "next/server";
import {
  parseGitHubStars,
  parseNpmDownloads,
} from "@/lib/stats";

export const runtime = "nodejs";

// Cache the response for 1h: GitHub's unauthenticated API is rate-limited
// (60 req/h per IP, shared across Vercel egress), and star/download counts
// don't need to be fresher than hourly.
export const revalidate = 3600;

const GITHUB_URL =
  process.env.WARDEN_STATS_GITHUB_URL ||
  "https://api.github.com/repos/Prof-bilal/Warden";
// npm has no all-time endpoint; sum a range from package creation to today.
const NPM_CREATED = "2026-09-06";
function npmDownloadsUrl(): string {
  if (process.env.WARDEN_STATS_NPM_URL) return process.env.WARDEN_STATS_NPM_URL;
  const today = new Date().toISOString().slice(0, 10);
  return `https://api.npmjs.org/downloads/range/${NPM_CREATED}:${today}/warden-sandbox-cli`;
}

const UPSTREAM_TIMEOUT_MS = 10000;

async function getJson(url: string): Promise<unknown | null> {
  for (let attempt = 0; attempt < 2; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), UPSTREAM_TIMEOUT_MS);
    try {
      const res = await fetch(url, {
        signal: controller.signal,
        next: { revalidate: 3600 },
        headers: { "User-Agent": "warden-landing-stats" },
      });
      if (!res.ok) return null;
      return (await res.json()) as unknown;
    } catch {
      if (attempt === 0) continue;
      return null;
    } finally {
      clearTimeout(timer);
    }
  }
  return null;
}

export async function GET() {
  const [gh, npm] = await Promise.all([
    getJson(GITHUB_URL),
    getJson(npmDownloadsUrl()),
  ]);
  return NextResponse.json({
    stars: parseGitHubStars(gh),
    downloads: parseNpmDownloads(npm),
  });
}
