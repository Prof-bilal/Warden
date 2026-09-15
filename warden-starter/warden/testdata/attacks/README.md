# Attack-Simulation Harness

Reproducible attack simulations that verify Warden's containment claims with
committed, machine-checkable evidence. Every scenario runs **twice**:

1. **Control phase** — the attack runs *unsandboxed* and must actually land.
   If the attack can't succeed without Warden, the scenario is **VOID**
   (it proves nothing either way).
2. **Sandbox phase** — the identical attack runs under `warden run` with a
   deny-by-default policy and must be contained.

Containment verdicts are confirmed **host-side** from artifacts the sandboxed
process cannot forge — not by trusting its output:

| Check | Artifact |
|---|---|
| Network exfil never arrived | collector request log (loopback) |
| Vault untouched by "ransomware"/escape probe | before/after sha256 of the decoy vault |
| No write outside grants | escape-probe file must not exist |
| Secrets not copied out | sandbox output dir scan |
| Runaway CPU actually terminated | cpu-bomb iteration count frozen after exit |

## Scenarios

| # | Name | What the attack does | Containment proof |
|---|---|---|---|
| 01 | `fs_exfil` | Copies decoy SSH keys + AWS credentials out of the sandbox | Zero copies in the output dir |
| 02 | `net_exfil` | POSTs a decoy fingerprint to an exfil collector | Collector log shows no sandbox POST |
| 03 | `env_steal` | Harvests decoy secrets from the environment | 0 of 3 planted vars visible |
| 04 | `process_spawn` | Probes the host, writes an escape probe into the vault | Probe file never appears host-side |
| 05 | `symlink_traverse` | Reads through a symlink pointing outside all grants | Read-through returns nothing |
| 06 | `ransomware` | XOR-"encrypts" decoy docs, deletes originals | Vault byte-identical, originals intact |
| 07 | `cpu_bomb` | Busy-loops unbounded | Timeout-killed; iteration count frozen |

## Run it

```bash
cd warden-starter/warden
go build -o warden ./cmd/warden
bash testdata/attacks/run-attacks.sh            # or: run-attacks.sh /path/to/warden
```

Exit 0 only when **every** scenario's control landed **and** the sandbox
contained it. Any VOID or FAIL exits non-zero.

Evidence lands in `evidence/<platform>/<stamp>/attacks/`:

```
evidence.md                  human-readable report (verdict tables + metrics)
summary.json                 machine summary (verdict, posts, hashes, iters)
results.jsonl                per-scenario verdict stream
control-steps.txt            sandbox-steps.txt        raw phase views
control-metrics.txt          sandbox-metrics.txt      measured quantities
collector.log                exfil collector request log (phase-tagged)
vault-integrity-*-*.txt      before/after vault sha256 per phase
audit.jsonl                  the sandboxed run's own audit stream
run-*-stdout.log             run-*-stderr.txt         what each phase printed
```

## Safety

- **Decoy data only.** Every "secret", key, and document is created by the
  harness with obviously-fake content (`DECOY-...-NOT-REAL`). The harness
  never reads real user files.
- **No real crypto.** "Ransomware" is an XOR with `0x5A` — an irreversible
  *looking* transform for the demo, trivially reversible, and applied only to
  harness-created decoys.
- **Loopback network only.** The exfil collector is a local HTTP server on
  `127.0.0.1`; no external host is contacted.
- **No persistent effects.** All state lives under the evidence run directory
  and is removed at exit (`evidence/` itself is gitignored).

## Environment knobs

| Variable | Default | Meaning |
|---|---|---|
| `WARDEN_ATTACKS_OUT` | `<repo>/evidence` | evidence root directory |
| `WARDEN_ATTACKS_TIMEOUT` | `60` | sandbox wall-clock timeout for the main run (s) |
| `WARDEN_BACKEND` | _(unset)_ | pin the sandbox backend (`docker`, `linux`, …) instead of warden's auto-detection; recorded in evidence |

The CPU-bomb scenario uses its own shorter timeout (5 s) to keep the run quick.
