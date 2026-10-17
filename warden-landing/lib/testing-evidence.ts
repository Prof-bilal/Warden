// Snapshot of reproduced Warden test results shown on /testing.
// Source: `go build`/`go vet`/`go test` run in warden-starter/warden, and the
// Linux proof harness (testdata/proof/run-proof.sh) evidence directory
// evidence/linux/20261016-203044/ (summary.json, results.jsonl, audit.jsonl).
// Regenerate this file when a new run replaces the snapshot.

export const RUN_DATE = "2026-10-16";

export const GO_CHECKS = [
  { check: "go build ./...", exit: 0, result: "Builds cleanly" },
  { check: "go vet ./...", exit: 0, result: "No issues" },
  {
    check: "go test ./... -count=1 -p 1",
    exit: 0,
    result: "23 packages pass · 3 with no test files · 0 failures",
  },
] as const;

export const TEST_FUNCTION_COUNT = 265;

export const HARNESS = {
  platform: "Linux x86_64",
  backend: "BubbleWrap (bwrap)",
  runId: "20261016-203044",
  runExit: 0,
  positiveControl: "PASS",
  stepsPassed: 8,
  stepsFailed: 0,
  verdict: "ok",
  steps: [
    { step: "read_allowed", expected: "SUCCESS", observed: "SUCCESS", what: "Read a file the policy allows" },
    { step: "write_allowed", expected: "SUCCESS", observed: "SUCCESS", what: "Write a file the policy allows" },
    { step: "read_secret", expected: "BLOCKED", observed: "BLOCKED", what: "Read a secret file outside the policy" },
    { step: "read_unlisted", expected: "BLOCKED", observed: "BLOCKED", what: "Read an unlisted file" },
    { step: "net_allowed", expected: "SUCCESS", observed: "SUCCESS", what: "HTTP to an allowlisted host" },
    { step: "net_blocked", expected: "BLOCKED", observed: "BLOCKED", what: "HTTP to an unlisted host (egress proxy 403)" },
    { step: "env_allowed", expected: "VISIBLE", observed: "VISIBLE", what: "Allowlisted env var is visible" },
    { step: "env_denied", expected: "BLOCKED", observed: "BLOCKED", what: "Unlisted env var is absent" },
  ],
} as const;

// Counts from evidence/linux/20261016-203044/audit.jsonl (535 entries).
export const AUDIT_SUMMARY = {
  total: 535,
  file: { allowed: 315, blocked: 201 },
  network: { allowed: 13, blocked: 6 },
} as const;
