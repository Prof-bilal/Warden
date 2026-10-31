# Proof & Test Results

Warden's security claims are backed by machine-checked evidence, not
hand-written promises. This page lists what is verified, how it is verified,
and how to reproduce it on your own machine.

## Verification state by platform

| Platform | Backend | State | Verified by |
|---|---|---|---|
| Linux | bubblewrap (native) | **Verified on real hardware** | escape tests + positive control in CI, full proof harness run locally |
| Linux | Docker fallback | Verified on this host | proof harness + integration tests (skipped automatically without a daemon) |
| Windows | AppContainer + WFP + ETW | **Verified via CI** | escape and lifecycle tests execute on GitHub-hosted Windows runners (elevated, so WFP/ETW run for real) |
| macOS | Seatbelt (`sandbox-exec`) | **CI-green; real-hardware harness run pending** | Seatbelt escape tests pass in CI on GitHub macOS runners; the end-user-machine proof-harness run has not happened yet |

CI hard-fails when it would otherwise lie: every Linux/macOS job fails if any
test `--- SKIP:`s or if the sandboxed-target positive control
(`TestSandboxPositiveControlStartup` / `TestSeatbelt*`) never passes, so a
silent skip cannot masquerade as a verified backend.

## What the proof harness checks

`testdata/proof/run-proof.sh` runs a harmless fixture target under
`warden run` and verifies the deny-by-default contract end to end:

| Step | Expected | Mechanism |
|---|---|---|
| read allowed file | SUCCESS | granted path |
| write allowed file | SUCCESS | granted path |
| read secret file (`~/.ssh/id_rsa`) | BLOCKED | not granted |
| read unlisted file | BLOCKED | not granted |
| HTTP to `127.0.0.1` (allowlisted) | SUCCESS | egress proxy allows |
| HTTP to `blocked-w4rd3n.invalid` | BLOCKED (HTTP 403) | egress proxy denies; audit records `allowed=false` |
| allowlisted env var | VISIBLE | `env.allow` |
| unlisted env var | BLOCKED | filtered before spawn |

Every result is voided unless the target first proves it started inside the
sandbox (the `WARDEN_SANDBOX_UP` marker file)a positive control against
"the sandbox blocked everything because nothing ran".

## Run it yourself

```bash
go build ./cmd/warden
bash testdata/proof/run-proof.sh ./warden
```

Requires Linux with `bwrap` + `strace` (the native backend is the one
exercised). The harness writes its artifacts to `evidence/<platform>/<stamp>/`:

- `results.jsonl`one record per step with expected/observed/verdict
- `audit.jsonl`the Warden audit records from this run only, including the
  real network-denial event for the blocked host
- `summary.json`machine-readable verdict
- `evidence.md`the human-readable table shown at the end

The fixture target only touches harness-created temp files and loopback
addresses; it cannot reach your real files or the internet.

## Hostile repository security proof

The Linux proof at `testdata/agent-security/run-linux.sh` uses a committed
hostile-repository fixture containing `AGENTS.md`. It first runs safe controls
without Warden against temporary fake data, then repeats the attacks inside
the bubblewrap backend three times by default.

| Attack | Expected | Result | Evidence |
|---|---|---|---|
| Direct, traversal, absolute, symlink, and subprocess filesystem access | blocked | machine-checked | `results.jsonl` and `audit.jsonl` |
| HTTP/HTTPS request to a non-allowlisted domain | blocked | machine-checked | proxy deny records |
| Subprocess network request and direct proxy bypass | blocked | machine-checked | proxy records and destination hit counters |
| Non-allowlisted DNS lookup | blocked | machine-checked | `results.jsonl` |

Run it from the Go module directory:

```bash
go build -o /tmp/warden-security-proof ./cmd/warden && \
WARDEN_SECURITY_REPEATS=3 bash testdata/agent-security/run-linux.sh /tmp/warden-security-proof
```

The artifact directory contains per-attack JSONL records with the attack,
policy, expected and actual outcomes, backend, reason/error, and PASS/FAIL.
The harness fails closed when the target does not start, a prerequisite is
missing, a fake secret appears in retained evidence, or the controlled
destination receives a request. This proof covers the Linux bubblewrap path;
it does not claim equivalent enforcement by macOS, Windows, or Docker.

## Known gaps that the tests do not paper over

The platform-specific audit blind spots and enforcement limits are documented
honestly in the [Security Review](security.md#known-limitations)e.g. Windows
denied file opens produce no ETW event by OS design, and macOS/Docker do not
enforce memory or timeout limits. The compatibility matrix
([18 servers](compatibility.md)) documents exactly which real-world MCP
servers pass, conditionally pass, or fail.

## Test suite

Unit, integration, and escape tests: see [Testing](testing.md) and
[`TESTING.md`](https://github.com/Prof-bilal/Warden/blob/main/warden-starter/warden/TESTING.md).
CI runs the full suite on Linux, macOS, and Windows plus cross-platform
buildsno platform is tested only by cross-compilation.
