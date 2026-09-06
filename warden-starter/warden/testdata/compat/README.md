# `testdata/compat` — M8 compatibility fixtures

One directory per MCP server in the [compatibility matrix](../../docs/compatibility.md),
each holding the exact `policy.yaml` that server needs. The manifest
[`matrix.yaml`](./matrix.yaml) is the single source of truth; the
[compat regression test](../../internal/compat/compat_test.go) enforces that
every matrix entry has a matching fixture and that every fixture policy
loads, validates, and grants exactly what the manifest probes claim.

## Layout

```text
testdata/compat/
  matrix.yaml            # manifest: verdict, failure class, probes per server
  <server>/policy.yaml   # exact policy the server needs
```

## CI behavior (deterministic, no sandbox needed)

`go test ./internal/compat/` only exercises the policy engine
(`policy.Load`, `Normalize`, `CoversFile`, host validation) — no `bwrap`,
no network, no MCP server install. It fails if:

- a matrix entry has no fixture directory (or vice versa),
- a fixture policy does not load / validate,
- an `probe_allow_*` entry is not actually granted (or a `probe_deny_*`
  entry leaks through),
- a `fail`/`conditional` entry lacks a failure class and notes,
- the documented schema gaps (wildcard hosts) stop being rejected without
  the matrix being updated first.

## Manual backend verification (needs Linux + bwrap)

Policy-level checks cannot prove end-to-end sandboxing. To reproduce the
matrix verdicts against a real backend:

```bash
# 1. Install the server under test, e.g.:
#    npm install -g @modelcontextprotocol/server-filesystem
#
# 2. Point the fixture policy's command at the installed entry point, e.g.:
#    command: ["/usr/bin/node", "/usr/lib/node_modules/@modelcontextprotocol/server-filesystem/dist/index.js"]
#
# 3. Run it sandboxed and exercise the server's tools from your MCP client:
warden run --policy testdata/compat/filesystem/policy.yaml

# 4. Confirm denials are logged, not silent:
warden logs --tail 20
```

Record the outcome (pass/fail + exact policy diff) back into `matrix.yaml`
and `docs/compatibility.md` — never change a fixture policy to make the
test pass without updating both.
