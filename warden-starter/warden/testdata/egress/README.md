# Egress and Enforcement Integrity Suite

External, evidence-first test harness for Warden's network policy. It is not
part of the Warden binary. The full evidence report is at
[`../../../../EGRESS-INTEGRITY.md`](../../../../EGRESS-INTEGRITY.md). The
release-readiness branch also includes the product fixes exercised by the
suite: port-qualified `network.allow` entries, a CONNECT TLS 1.2 ClientHello
floor, proxy hop-by-hop header stripping, and Docker `--init` so the bridge is
not PID 1.

The suite answers one question: **can a sandboxed process reach a destination
that the policy did not authorize, through any path the harness can exercise?**
It tests direct TCP/UDP, DNS/DoT/DoH, IPv4/IPv6, proxy bypass, redirects,
HTTP/HTTPS/WebSocket/SSE/MCP, subprocesses, new network namespaces, policy and
audit tampering, and enforcement shutdown.

## Classification

| Result | Meaning |
|---|---|
| `PASS` | The expected behavior was observed with a positive control and corroborating collector/DNS/audit evidence. |
| `FAIL` | An unauthorized path succeeded, or a required control did not hold. |
| `WARN` | A documented limitation was observed; it is **not** a PASS. |
| `BLOCKED` | The case could not be executed for an environmental or product reason. |
| `UNSUPPORTED` | The case is not applicable to this backend or mode. |
| `NOT_RUN` | The case was deliberately skipped (for example, missing interpreter). |

`WARN`, `UNSUPPORTED`, `BLOCKED`, and `NOT_RUN` are never counted as PASS.
Timeouts alone are WARN. Every executed case requires a `PROBE_UP` marker; allow
cases require a 2xx response plus a collector hit plus an audit allow; deny
cases require an explicit denial class and no forbidden collector hit.

## Prerequisites

**Linux bubblewrap**
- `bwrap`, `strace`, `unshare`, `ip`
- Unprivileged user+net+mount namespaces. On Ubuntu 24.04, load scoped AppArmor
  profiles for `bwrap` and `unshare` rather than disabling the restriction
  host-wide; the CI workflow shows the profile setup.

**Docker**
- A working Docker daemon and an image that already exists locally. Warden does
  not pull images implicitly; pull explicitly:
  `docker image inspect alpine:3.20 >/dev/null 2>&1 || docker pull alpine:3.20`
- `python3` on the host for F13. The Docker backend mounts the host `/usr`
  runtime base, so the host interpreter is available inside the container.
- If the shell predates docker-group membership, wrap the run in
  `newgrp docker -c '...'`.

**macOS**
- `sandbox-exec` (hosted macOS has it) and `python3` for the interpreter case.

**Windows**
- AppContainer/WFP requires self-hosted hardware that permits WFP engine
  sessions. GitHub-hosted runners record BLOCKED/NOT_RUN, never PASS.

## Run

```bash
cd warden-starter/warden

# Build the external runner.
CGO_ENABLED=0 go build -o /tmp/opencode/egress-runner ./testdata/egress/runner

# Full Linux bubblewrap matrix (fails on BLOCKED/NOT_RUN with -require-complete).
/tmp/opencode/egress-runner -backend linux -mode netns -build \
  -require-complete -out /tmp/opencode/egress-linux

# Full Docker matrix (F1 and F13 pass; -require-complete rejects BLOCKED/NOT_RUN).
newgrp docker -c 'cd "$PWD" && /tmp/opencode/egress-runner \
  -backend docker -image alpine:3.20 -mode netns -build \
  -require-complete -out /tmp/opencode/egress-docker'

# macOS host mode (netns/DNS/IPv6 cases are UNSUPPORTED).
"$RUNNER_TEMP/egress-runner" -backend seatbelt -mode host -build \
  -out "$RUNNER_TEMP/egress-macos"
```

Useful flags:

```text
-backend auto|linux|docker|seatbelt|windows
-mode auto|netns|host
-filter REGEXP        only run matching case IDs
-group A|B|C|D|E|F|G  only run one group
-list                 list the matrix
-require-complete     exit nonzero on BLOCKED or NOT_RUN
```

## Evidence layout

Each run writes:

```text
<out>/
  run.json                 run metadata and limitations
  results.jsonl            one JSON result per case
  summary.json             counts and verdict
  report.md                human-readable report
  policies/<case>.yaml     active policy used for the case
  cases/<id>/
    policy.yaml            evidence copy of the policy
    result.json            classification and reason
    steps.json             exact probe commands
    probe-events.json      probe-emitted events
    collector-hits.json    local collector observations
    dns-hits.json          local DNS observations
    audit-events.json      proxy audit events
    checks.json            named check results (absent for BLOCKED cases)
  collectors/              aggregate collector and DNS logs
  harness/                 built warden/probe binaries and test CA
```

The active policy is rendered outside the sandbox-visible case tree, and its
SHA-256 is recorded before and after each case. `active_policy_unchanged` is a
required check.

## Matrix schema

Cases live in `matrix.yaml`. A single-probe case has:

```yaml
- id: A1
  group: A
  title: Allowed hostname over HTTP succeeds
  policy: allow-host
  probe: ["http", "--url", "{{allowedURL}}"]
  expect: allow
  checks: [probe_status_2xx, collector_hit, audit_allow]
  requires: [netns]
  timeout: 30s
  notes: ...
```

A step-based case uses `kind: multi` and a `steps` list instead of a top-level
`probe`:

```yaml
- id: E10
  kind: multi
  group: E
  title: Host header mismatch does not bypass policy
  policy: allow-host
  steps:
    - name: allowed-host-mismatch
      probe: ["absolute", "--url", "{{allowedURL}}", "--host-header", "{{deniedHost}}"]
      expect: allow
      checks: ["step_allow:allowed-host-mismatch"]
      required: true
  expect: observe
  checks: [host_mismatch_observed]
  requires: [netns]
```

`requires` values are `netns`, `ipv6`, `linux`, `docker`, `seatbelt`,
`windows`, `proxy-unix`, `strace`, and `python`. Templates use Go
`text/template` variables such as `{{allowedHost}}`, `{{deniedURL}}`,
`{{allowedIP}}`, `{{proxyAddr}}`, and `{{policyPath}}`. Per-case hostnames and
tokens are generated under `<run-id>.egress.test` so a collector hit from
another case cannot satisfy a check.

Special case kinds: `mcp_stdio`, `kill_warden`, `stop_dns`, `stop_collector`,
`multi`, `schema_observe`, `unsupported`, `not_run`.

## CI

`.github/workflows/egress-integrity.yml` runs:

- Linux bubblewrap: full netns matrix with `-require-complete`
- Docker: full netns matrix with `-require-complete`; F1/F13 pass, and the
  container runs as the mapped host user
- macOS Seatbelt: host-mode matrix; netns cases are UNSUPPORTED
- Windows hosted: capability report only; BLOCKED/NOT_RUN, never PASS
- Windows self-hosted WFP: gated on `vars.WARDEN_WFP_SELF_HOSTED == 'true'`,
  with `-require-complete`

All jobs upload `run.json`, `results.jsonl`, `summary.json`, `report.md`, and
per-case evidence. A FAIL exits non-zero; WARN/UNSUPPORTED/BLOCKED/NOT_RUN are
never converted to PASS.

## Known limitations

See [`../../../../EGRESS-INTEGRITY.md`](../../../../EGRESS-INTEGRITY.md) for
the current evidence and boundaries. In short:

- Linux bubblewrap and Docker both pass all 83 cases with no WARN,
  UNSUPPORTED, BLOCKED, or NOT_RUN results on this host. The Docker run uses
  `--init`; F5 verifies that killing the bridge terminates the workload and
  does not leave a signal-immune PID 1.
- Docker resolves `/proc/self/{uid,gid}_map` and runs the container as the
  mapped host user, not namespace root. F1 (write-grant control) and F13
  (Python interpreter) pass; the previous userns defect is fixed.
- A4 now tests real port-granular grants (`host:port`). A bare host grants
  every port; a port-qualified entry grants only that port. C10 and F11 run on
  Docker and pass: the container drops `CAP_SYS_ADMIN`, so `unshare` is
  refused or produces a namespace with no route.
- D14 distinguishes the two boundaries: the host proxy control socket is
  reachable but still enforces `network.allow` (403 for a denied host), while
  an AF_UNIX socket inside a granted filesystem path is reachable by design.
  AF_UNIX is not namespaced and is outside the network-policy boundary; the
  filesystem grant is its boundary.
- E8 is enforced for CONNECT: the proxy inspects the first TLS ClientHello and
  rejects a client whose highest offered version is below TLS 1.2. It does not
  decrypt the tunnel or bind SNI to policy; non-TLS and server-first CONNECT
  traffic is forwarded without inspection.
- Host mode (macOS/Windows) cannot exercise hostname/DNS/IPv6 netns cases.
- Windows WFP enforcement requires self-hosted hardware (or a VM with a real
  Windows guest). The first real-Windows run showed the WFP filter install
  failing with `RPC_X_BAD_STUB_DATA (0x6f7)` because the Go mirror of
  `FWPM_FILTER0` disagreed with the SDK; that mirror is fixed and pinned by
  `internal/sandbox/windows/wfp_spec_test.go` and `wfp_layout_test.go`. A
  real-Windows re-run is still required — see
  [`../../../../WINDOWS-AI-TEST-GUIDE.md`](../../../../WINDOWS-AI-TEST-GUIDE.md).
- Port-qualified grants apply to Warden's own egress proxy. The Docker/Kubernetes
  manifest translation path does not preserve a host:port distinction; use the
  Warden proxy path when port granularity matters.
