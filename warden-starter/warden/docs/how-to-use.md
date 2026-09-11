# How to use Warden

A complete walkthrough from install to a verified sandbox. Every step shows
the command, why it exists, what you should see, and what just happened.

## 1. Install

```bash
npm install -g warden-sandbox-cli
warden version
```

Expected output:

```
warden version 0.1.11
```

**Why:** the npm wrapper downloads the prebuilt Warden binary for your
platform from GitHub Releases and caches it in `~/.cache/warden/`. Prefer
another route? See [Install](install.md) for GitHub Release downloads and
building from source.

**What just happened:** you have a single static `warden` binary on your
PATH. It is not yet a sandbox — the next step checks your OS can actually
isolate processes.

## 2. Check your machine is sandbox-ready

```bash
warden doctor
```

Expected output (Linux, ready):

```
✓ Sandbox active
```

Exit code `0` means ready, `1` means NOT READY (with the missing piece named
— e.g. `bwrap` or `strace`), `2` means you used the command wrong.

**Why:** Warden refuses to run a server unsandboxed. `doctor` tells you
*before* you try whether this host can enforce a policy — on Linux it checks
bubblewrap and user namespaces, and whether `strace` is present (required for
`warden run` auditing on the native backend; [why](security.md#known-limitations)).

## 3. Write a policy

```bash
cat > policy.yaml <<'EOF'
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["NODE_ENV", "GITHUB_TOKEN"]
limits:
  memory_mb: 512
  timeout_s: 300
EOF
```

**Why:** the policy is the entire security contract. Whatever is not granted
here does not exist for the server: unlisted files are invisible, unlisted
hosts are unreachable, unlisted env vars are stripped. Don't guess what the
server needs — [trace it first](#5-dont-guess-trace-instead) or copy a fixture from
the [compatibility matrix](compatibility.md).

**What just happened:** nothing yet — but every path, host, and variable in
this file is now an explicit allowlist entry. Relative paths resolve against
the policy file's own directory.

## 4. Run the server sandboxed

```bash
warden run --policy ./policy.yaml
```

Expected: your MCP client (Claude Desktop, an IDE, `mcp-remote`, etc.)
connects over stdio exactly as before. The server cannot tell it's sandboxed.

**Why:** `warden run` starts the server inside the OS sandbox — bubblewrap
namespaces on Linux, Seatbelt on macOS, AppContainer on Windows, Docker as a
fallback — with the policy grants and nothing more. If the sandbox cannot be
initialized, Warden exits with an error instead of running unsandboxed.

## 5. Don't guess — trace instead

```bash
warden trace -- /usr/bin/node ./server/dist/index.js
warden init
```

**Why:** `trace` runs the server *unsandboxed* once and records every file,
network, and environment access. `warden init` turns that record into a
conservative starter policy — it only grants accesses that were observed to
succeed. Review the generated policy (trace ran with your full permissions),
tighten anything too broad, then `warden run` with it.

## 6. Try something forbidden — and watch it fail

Point your MCP client at a file outside the grant (say `~/notes.md`), or from
inside the server try to read a secret:

```bash
cat ~/.ssh/id_rsa   # from within the sandboxed server: BLOCKED
```

Then look at the audit log:

```bash
warden logs --tail 5
```

Expected output (real records, from the proof harness):

```
{"timestamp":"...","type":"network","action":"GET","resource":"blocked-w4rd3n.invalid:80","allowed":false,"reason":"host is not in network.allow"}
```

**What just happened:** the access attempt was denied at the sandbox
boundary and — critically — **recorded**. Blocked attempts name the exact
path or host, so widening a policy is mechanical: add the grant, re-run.
Silent failures are the enemy; Warden's audit log is the antidote.

## 7. Understand the policy you just wrote

Every field in `policy.yaml` maps to a real enforcement mechanism:

- `filesystem.read/write` → sandbox mount/view rules (bwrap `--ro-bind` /
  `--bind`, Seatbelt file rules, AppContainer capabilities)
- `network.allow` → egress proxy allowlist + network namespace isolation
- `env.allow` → environment filter applied before the process starts
- `limits` → memory cap and timeout where the platform supports it (not on
  macOS/Docker — see the [limitations table](security.md#known-limitations))

Field-by-field reference: [Schema Reference](schema.md).

## 8. Run the full security proof

```bash
go build ./cmd/warden
bash testdata/proof/run-proof.sh ./warden
```

Expected ending:

```
🎉 proof verified: contract holds with the positive control satisfied
   artifacts: .../evidence/linux/<stamp>
```

**Why:** this is the same harness used for Warden's own release claims —
eight allow/block checks against the sandbox, gated on a positive control
that proves the target really started. It writes `results.jsonl`,
`audit.jsonl`, `summary.json`, and `evidence.md`. Details:
[Proof & Test Results](proof.md).

## Next steps

- [Schema Reference](schema.md) — every policy field and its enforcement notes
- [CLI Reference](cli.md) — every command and flag
- [Compatibility Matrix](compatibility.md) — which real MCP servers work, with exact policies
- [Security Review](security.md) — the threat model and known gaps