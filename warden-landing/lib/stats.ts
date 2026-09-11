// Pure helpers for the landing's social-proof stats (/api/stats + SocialProof).
// No Next.js imports here so this module stays unit-testable with plain node.
export function formatCount(n: unknown): string | null {
  if (typeof n !== "number" || !Number.isFinite(n) || n < 0) return null;
  const v = Math.floor(n);
  if (v < 1000) return String(v);
  const units: Array<[number, string]> = [
    [1e9, "B"],
    [1e6, "M"],
    [1e3, "k"],
  ];
  for (const [threshold, suffix] of units) {
    if (v >= threshold) {
      const x = v / threshold;
      const str = x >= 100 ? String(Math.round(x)) : String(Math.round(x * 10) / 10);
      return str + suffix;
    }
  }
  return String(v);
}

export function parseGitHubStars(data: unknown): number | null {
  if (typeof data !== "object" || data === null) return null;
  const v = (data as Record<string, unknown>).stargazers_count;
  if (typeof v !== "number" || !Number.isFinite(v) || v < 0) return null;
  return Math.floor(v);
}

export function parseNpmDownloads(data: unknown): number | null {
  if (typeof data !== "object" || data === null) return null;
  const v = (data as Record<string, unknown>).downloads;
  if (typeof v !== "number" || !Number.isFinite(v) || v < 0) return null;
  return Math.floor(v);
}
