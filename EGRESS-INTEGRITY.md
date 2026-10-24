# Egress and Enforcement Integrity

Evidence report for Warden's deny-by-default network policy across the Linux
bubblewrap and Docker backends, with capability/CI wiring for macOS Seatbelt
and Windows AppContainer/WFP.

- **Date:** 2026-10-24
- **Branch:** `ctx/egress-integrity`
- **Warden revision tested:** `9d166b0` (harness F6 teardown fix) on top of
  `8e4e942` (product: port-qualified grants, CONNECT TLS floor, hop-by-hop
  header stripping, Docker `--init`), `55cac76`/`ba53732` (Docker mapped host
  user), and suite commit `2c40cab`
- **Host:** Linux x86_64, kernel `7.2.8-9-omarchy`; bubblewrap 0.13.1;
  util-linux 2.42.4; iproute2 7.3.0; strace 7.2; Go 1.27.3; Docker 29.9.2;
  `alpine:3.20`
  (`sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc`)
- **Harness:** `warden-starter/warden/testdata/egress/` (external Go runner +
  probe; it is not part of the Warden binary or its public API)
- **Matrix:** 83 cases in groups A–G (`testdata/egress/matrix.yaml`)

This report follows the evidence-first rule: **WARN, UNSUPPORTED, BLOCKED and
NOT_RUN are never counted as PASS**, and no platform is reported as PASS unless
it was actually executed. No real secrets, production endpoints, or destructive
host network/firewall changes were used. All collectors and DNS are local.

---

## 1. What was tested

### 1.1 Threat model

A sandboxed process must not reach an unauthorized destination through any
alternate path:

- direct IPv4/IPv6 TCP or UDP, including raw IP literals, loopback, private
  ranges, link-local metadata addresses, and alternate ports;
- direct DNS (UDP/TCP port 53), DoT, DoH, resolver rebinding, mixed-case/
  trailing-dot variants, wildcard apex/lookalike names;
- proxy bypass (ignoring the proxy, changing proxy environment variables,
  launching a second proxy, SOCKS-style requests, absolute-form requests,
  redirects, URL userinfo);
- subprocesses, freshly executed helpers, Python interpreters, and new
  network namespaces;
- tampering with the policy, audit log, resolver configuration, hosts file, or
  the proxy control socket;
- enforcement shutdown (signals to the in-sandbox bridge, supervisor kill,
  in-flight requests, DNS/collector failure, timeouts, load/exhaustion).

The policy is hostname-based. The proxy is the only permitted egress path on
the native Linux backend; Docker uses `--network none` plus a bind-mounted
Unix-socket bridge to the host-side proxy. Direct egress should fail with no
route, and the proxy should authorize only allowlisted destinations.

### 1.2 Harness design

- **External runner and probe.** The runner is a standalone Go program under
  `testdata/egress/runner`; the probe is under `testdata/egress/probe`. Neither
  is part of the Warden binary or its public API.
- **Isolated test network.** On Linux the runner re-execs itself under
  `unshare --user --map-root-user --net --mount --propagation private`. Test
  collectors bind TEST-NET IPv4 (`192.0.2.0/24`), documentation IPv6
  (`2001:db8::/32`), loopback, and fake metadata/private addresses on `lo`.
  A local DNS server listens on `192.0.2.53:53`; no host resolver or firewall
  configuration is changed.
- **Local collectors.** Allowed HTTP/HTTPS/alt-HTTP, denied HTTP/HTTPS/TCP/UDP,
  metadata, private, loopback, IPv6, DoT/DoH, fake proxy, Unix echo, redirects,
  SSE, WebSocket, slow/hold, and large-body endpoints. Every collector records
  source/destination, protocol, hostname/token, and payload metadata.
- **Per-case policy fixtures.** Nine templates under
  `testdata/egress/policies/` are rendered per case and validated with the real
  `policy.Load`. The **active policy is written outside the sandbox-visible
  case tree** at `<out>/policies/<case>.yaml`; a copy is kept for evidence at
  `cases/<case>/policy.yaml`. Each case records the policy SHA-256 before and
  after execution; `active_policy_unchanged` is a required check, so a
  sandbox-side write cannot masquerade as a policy change.
- **Positive controls.** Every case requires a `PROBE_UP` marker. Allow cases
  require a 2xx response **and** a collector hit **and** an audit allow
  (`action=connect`); deny cases require an explicit denial/error class **and**
  no forbidden collector hit. Timeouts alone are WARN, never PASS. DNS,
  collector, audit, and process events are cross-checked.
- **Per-case hostnames/tokens.** Every case uses unique hostnames and tokens
  under `<run-id>.egress.test`, so a hit from another case cannot satisfy a
  check.
- **Docker.** The container runs with `--init`, `--network none`,
  `--cap-drop ALL`, a read-only root, and a tmpfs `/tmp`. The only egress path
  is the Warden proxy bridge over a bind-mounted Unix socket. The Docker matrix
  runs in the same netns harness. The backend resolves
  `/proc/self/{uid,gid}_map` and runs the container as the mapped host user;
  the earlier test-only socket-permission workaround has been removed.

### 1.3 Matrix groups

| Group | Cases | Focus |
|---|---:|---|
| A | 10 | Allow/deny by hostname, IP, and port; redirects; unlisted/rebinding names |
| B | 10 | DNS path: UDP/TCP 53, DoT/DoH, rebinding, normalization, wildcards |
| C | 10 | Raw TCP/UDP, IPv4/IPv6, loopback/private/metadata, subprocess, new netns |
| D | 14 | Proxy bypass, CONNECT, absolute-form, redirects, auth headers, Unix socket |
| E | 12 | HTTP/HTTPS POST, WebSocket, SSE, MCP stdio/HTTP/SSE, TLS/cert/Host/SNI, h2, chunked |
| F | 15 | Policy/audit/env/resolver/hosts tampering, second process, helpers, buffers, supervisor kill |
| G | 12 | Load, long streams, large bodies, connection exhaustion, DNS/collector failure, timeouts |

---

## 2. Results

### 2.1 Summary

| Backend | Total | PASS | FAIL | WARN | BLOCKED | UNSUPPORTED | NOT_RUN | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| Linux bubblewrap (netns) | 83 | 83 | 0 | 0 | 0 | 0 | 0 | **YES** |
| Docker (netns) | 83 | 83 | 0 | 0 | 0 | 0 | 0 | **YES** |
| macOS Seatbelt | — | — | — | — | — | — | — | **INCONCLUSIVE** (not executed locally) |
| Windows AppContainer/WFP | — | — | — | — | — | — | — | **INCONCLUSIVE** (not executed locally) |

- **Final Linux run:** `r1792834593384b49` (2026-10-24T09:36:35Z →
  09:37:00Z), mode `netns`, revision `9d166b0`, `-require-complete`. Evidence:
  `/tmp/cceg/final-linux`.
- **Final Docker run:** `r1792834620dd73b4` (2026-10-24T09:37:01Z →
  09:38:22Z), mode `netns`, revision `9d166b0`, `-require-complete`. Evidence:
  `/tmp/cceg/final-docker`.

`-require-complete` fails the run on any BLOCKED or NOT_RUN case. Neither run
produced a WARN or UNSUPPORTED case either, so there is nothing to exclude from
the counts above. `warden_commit` in each `run.json` is
`9d166b06c5a5101107685bdacaace3670a05ed15`.

Each evidence directory contains `run.json`, `results.jsonl`, `summary.json`,
`report.md`, `policies/<case>.yaml`,
`cases/<case>/{policy.yaml,result.json,steps.json,probe-events.json,collector-hits.json,dns-hits.json,audit-events.json,checks.json,*.stdout.log,*.stderr.log}`,
and `collectors/{collector-hits.jsonl,dns.jsonl}`.

A copy of both final runs is also preserved locally under the gitignored
`evidence/egress/linux/<stamp>/` tree (`20261024T093633Z/linux` and
`20261024T093700Z/docker`) so the evidence survives `/tmp` cleanup.

### 2.2 Per-group breakdown

Both backends: every case PASS, zero FAIL/WARN/BLOCKED/UNSUPPORTED/NOT_RUN.

| Group | Linux PASS | Docker PASS |
|---|---:|---:|
| A | 10 | 10 |
| B | 10 | 10 |
| C | 10 | 10 |
| D | 14 | 14 |
| E | 12 | 12 |
| F | 15 | 15 |
| G | 12 | 12 |
| **Total** | **83** | **83** |

Key positive evidence:

- **A1–A10:** allowed host reached the intended collector with a 2xx, a
  collector hit, and an audit `action=connect`; denied/unlisted/rebinding/
  direct-IP/redirect cases produced explicit denials and no forbidden
  collector hits. A4 now exercises real port-granular grants (see §3.4).
- **B1–B10:** DNS queries followed the intended resolver path; denied names
  produced no usable resolution; DoT/DoH to unauthorized endpoints failed;
  mixed-case/trailing-dot and wildcard/apex/lookalike variants matched policy.
- **C1–C10:** direct TCP/UDP, IPv4/IPv6, loopback, private, metadata, and
  subprocess-created sockets failed; `unshare(CLONE_NEWNET)` from inside the
  sandbox was refused on Docker and produced a namespace with no route on
  Linux.
- **D1–D14:** direct connections, alternate proxies, CONNECT, absolute-form,
  SOCKS-style requests, two-hop redirects, URL userinfo, numeric IP formats,
  second proxies, proxy auth headers, and the Unix control socket were denied
  where required. D11 and D14 are now passing (see §3.1, §3.3).
- **E1–E12:** POST bodies, WebSocket, SSE, MCP stdio/HTTP/SSE, certificate
  validation, Host/SNI mismatch denials, HTTP/2 (when negotiated), chunked/
  streamed bodies, and the CONNECT TLS floor stayed policy-conforming.
- **F1–F15:** policy read/write/replace, audit write, proxy env changes,
  resolver/hosts tampering, second processes, fresh helper binaries, buffer
  pressure, signals to the enforcement process, and supervisor kill all failed
  closed. Every case also passed `active_policy_unchanged`. F1 (write-grant
  control) and F13 (Python interpreter) PASS on both backends.
- **G1–G12:** high-rate DNS, short/concurrent connections, long streams, large
  bodies, connection exhaustion, DNS/collector failure, and timeouts stayed
  fail-closed; recovery worked when DNS/collectors returned.

No required check failed and no warn-only check failed in either final run.

---

## 3. Per-backend answer and resolved findings

The exact question: **“Can Warden currently demonstrate that every available
network path obeys policy on the tested backend?”**

### Linux bubblewrap — YES, for the tested network-policy contract

- 83/83 PASS; 0 FAIL, 0 WARN, 0 BLOCKED, 0 UNSUPPORTED, 0 NOT_RUN.
- The tested paths are the ones the suite can exercise: HTTP/HTTPS, CONNECT,
  redirects, POST/streaming, WebSocket/SSE, MCP, DNS/DoT/DoH, raw TCP/UDP,
  IPv4/IPv6, loopback/private/metadata, subprocesses, new netns, proxy bypass,
  policy/audit tampering, and enforcement shutdown.
- The properties that were previously WARN/UNSUPPORTED (D11, D14, E8, A4,
  C10/F11) are now enforced or deliberately reclassified with explicit scope in
  §3.1–§3.6. Every one of them is backed by an executed case that PASSed.

### Docker — YES, for the tested network-policy contract

- 83/83 PASS; 0 FAIL, 0 WARN, 0 BLOCKED, 0 UNSUPPORTED, 0 NOT_RUN.
- The container→bridge→proxy→policy path is demonstrated end to end, including
  the F1 write-grant control, the F13 Python interpreter, C10/F11 `unshare`
  denial, and the F5/F6 enforcement-shutdown cases.
- The backend resolves `/proc/self/{uid,gid}_map` before passing `--user`, so
  the container runs as the mapped host user rather than namespace root. The
  container keeps `--init`, `--network none`, `--cap-drop ALL`, and the
  bridge-only egress path.

### macOS Seatbelt — INCONCLUSIVE / not executed locally

- No macOS host was available in this environment. The runner and probe
  cross-compile for `darwin/amd64` and `darwin/arm64`; the CI workflow runs the
  Seatbelt backend in host mode on `macos-latest`.
- Host mode uses the allowlisted IP literal `127.0.0.1` and the LAN IP as the
  denied target. Hostname/DNS/IPv6 cases require the Linux netns harness and
  are reported **UNSUPPORTED** on macOS; they are never PASS.
- Do not claim macOS Seatbelt success until the CI evidence is reviewed.

### Windows AppContainer/WFP — INCONCLUSIVE / not executed locally

- No Windows host was available in this environment. The runner and probe
  cross-compile for `windows/amd64`; the CI workflow runs the Windows backend
  in host mode on `windows-latest`.
- GitHub-hosted Windows runners ship `fwpuclnt.dll` but refuse WFP engine
  sessions, so the Windows backend refuses to run and the suite records
  **BLOCKED/NOT_RUN**, never PASS.
- Real WFP enforcement requires self-hosted Windows hardware. The workflow
  contains a `windows-wfp-self-hosted` job gated on
  `vars.WARDEN_WFP_SELF_HOSTED == 'true'` with `-require-complete`, so a real
  run cannot silently degrade into a capability report.

### Overall

**INCONCLUSIVE across platforms.** Linux bubblewrap and Docker are **YES for
the tested network-policy contract** on this host, with zero WARN/UNSUPPORTED/
BLOCKED/NOT_RUN; macOS and Windows are untested locally and must not be reported
as PASS. **No cross-platform YES is claimed.**

---

### 3.1 D11 — proxy header stripping (RESOLVED)

The proxy now strips `Proxy-Authorization` and the other hop-by-hop proxy
headers (`Connection` plus any header it names, `Keep-Alive`,
`Proxy-Authenticate`, `Proxy-Connection`, `TE`, `Trailer`, `Transfer-Encoding`,
`Upgrade`) before forwarding a request upstream
(`internal/proxy/proxy.go`, `stripHopByHopHeaders`). The value is never logged.

Evidence: D11 PASS on Linux and Docker. The warn-only check
`no_proxy_auth_upstream` reports `ok=true` ("no Proxy-Authorization header
observed upstream") on both. Regression test:
`TestProxyStripsProxyAuthAndHopByHopHeaders`.

### 3.2 E8 — CONNECT TLS floor (RESOLVED, ClientHello-only scope)

For a CONNECT tunnel whose first client byte is a TLS handshake record, the
proxy buffers and inspects the **ClientHello** before forwarding anything. If
the client's highest offered version is below TLS 1.2, the proxy returns a
fatal `protocol_version` alert and never relays the ClientHello to the upstream
(`errTLSFloor`). Non-TLS and server-first tunnels are forwarded without
inspection, bounded by a 2s first-byte timeout.

**Explicit scope — what this does and does not do:**

- It enforces a **client-offered** floor. A client that offers only TLS < 1.2
  is rejected.
- It does **not** decrypt the tunnel, inspect `ServerHello`, or bind SNI to
  policy. A client that offers 1.2+ may still negotiate a lower version with a
  legacy server, because Warden does not read the server's answer. SNI-aware
  policy would require TLS interception and CA management.

Evidence: E8 PASS on Linux and Docker; `tls_floor_denied` ok, the upstream
`allowed-https` collector shows 0 hits, and the client sees a TLS error.

### 3.3 D14 — AF_UNIX is filesystem-scoped (RECLASSIFIED, not a bypass)

AF_UNIX sockets are not namespaced by the network backend, so they are outside
the network-policy boundary; the filesystem grant is their boundary. This is a
deliberate reclassification, not a relaxation of the network claim.

Evidence: D14 PASS on Linux and Docker with two checks. `unix_proxy_denied`
proves the host proxy control socket still enforces policy (403 for a
non-allowlisted host), and `unix_granted_path_reachable` asserts that an
AF_UNIX socket inside a granted path is reachable **by design**. A writable/
readable grant on a directory that holds host service sockets remains a trust
boundary; restricting AF_UNIX to granted paths would need a syscall filter or a
Unix-socket broker (a design change, not required here).

### 3.4 A4 — port-qualified grants (RESOLVED)

`network.allow` entries may now be port-qualified. `example.test` grants every
port; `example.test:8443` grants only that port; IPv6 with a port uses
`[addr]:port` (`internal/policy/policy.go`, `splitHostPortRule`/
`validatePort`). The proxy enforces the pairing.

Evidence: A4 PASS on Linux and Docker — the allowed-port step reaches the
destination, the denied-port step on the same host is denied, and the alternate
collector shows 0 hits. Regression tests: `TestSplitHostPortRule`,
`TestAllowedHonorsPortSpecificGrants`, `TestProxyPortSpecificGrantDeniesOtherPort`.

**Explicit scope:** port-qualified grants are enforced by Warden's own egress
proxy. The generic Docker/Kubernetes manifest translation path does not preserve
a `host:port` pairing; use the Warden proxy path when port granularity matters
(documented in `docs/container-k8s.md`).

### 3.5 F5 / F6 — enforcement shutdown fails closed (RESOLVED)

The Docker backend now starts the container with `--init`
(`internal/sandbox/docker/docker.go`), so the in-container proxy bridge is an
ordinary child rather than PID 1. Signalling the bridge therefore terminates
enforcement instead of leaving a signal-immune PID 1 behind.

Evidence: F5 (SIGKILL) and F6 (SIGTERM) PASS on Linux and Docker. F5 exits 137;
F6 exits 255. In both the workload is terminated when the enforcement process
receives the signal — nothing survives to keep using the network.

F6 exposed a **harness** timing issue, not a product bypass: on Docker, the
SIGTERM teardown can kill the probe after it emitted the pre-signal phase but
before it flushed the `kill(2)` result. `killFailClosed` now accepts a
pre-signal phase event **plus a non-zero workload exit** as fail-closed
evidence, mirroring the existing post-signal fallback. It never treats a
surviving workload (zero exit) as a PASS, and a case with no signal attempt
remains a hard FAIL. Covered by `TestKillFailClosed`. Supervisor-kill
fail-closed behavior is additionally covered by F15/G9.

### 3.6 C10 / F11 — applicability broadened to netns (RESOLVED)

C10 (child cannot create a new network namespace) and F11 (unshare cannot
create an unrestricted path) previously ran only on the Linux bubblewrap
backend. Their applicability is now `netns`, and they run on Docker too.

Evidence: both PASS on Linux and Docker. On Docker, `unshare(CLONE_NEWNET)` is
refused (`CAP_SYS_ADMIN` dropped); on Linux it succeeds but yields a namespace
with no route to the denied destination (`post_unshare_ok=false`). F11 also
confirms recovery (`step_allow:recovery`).

### 3.7 Other corrections

- **Docker mapped host user (FIXED).** `BuildDockerArgs` resolves the host
  UID/GID via `/proc/self/{uid,gid}_map` and fails closed if the map is
  unavailable or the ID is unmapped, instead of passing `--user 0:0`. F1
  verifies the container can write the granted control path while the active
  policy stays unreadable; F13 verifies an in-sandbox Python interpreter can use
  the allowlisted URL and cannot open the denied socket.
- **E10 (harness false positive corrected).** `hostMismatchObserved` now
  normalizes the case, a trailing dot, and an optional port before comparing the
  collector Host to the hostname-only policy value. E10 passes on both backends;
  the proxy forwards the authorized Host. HTTPS/CONNECT remains an opaque
  tunnel, so SNI binding is not claimed.

---

## 4. Remaining boundaries (not PASS anywhere)

- **macOS Seatbelt and Windows WFP are unexecuted** on this host. Host mode
  cannot exercise hostname/DNS/IPv6 netns cases; Windows WFP needs self-hosted
  hardware. Nothing on those platforms is counted as PASS.
- **E8 is a ClientHello floor only** (§3.2): no ServerHello/SNI inspection.
- **AF_UNIX is filesystem-scoped** (§3.3): the filesystem grant is the boundary.
- **Port-qualified grants** apply to Warden's egress proxy, not to the generic
  Docker/Kubernetes manifest translation path (§3.4).
- **Host mode** (macOS/Windows) reports hostname/DNS/IPv6 cases UNSUPPORTED.

---

## 5. Adoption recommendations

| Area | Evidence | Recommendation |
|---|---|---|
| A/B allowlist + DNS | Linux/Docker 20 PASS | Make a CI gate. Covers HTTP/HTTPS allow/deny, port-qualified grants, unlisted/rebinding names, direct IP, DoT/DoH, normalization, wildcard/apex/lookalike. |
| C raw sockets | Linux/Docker 10 PASS | Keep as a gate. No direct TCP/UDP/IPv6/loopback/private/metadata egress; subprocess sockets covered; new netns denied on Linux and refused on Docker. |
| D proxy bypass/tamper | Linux/Docker 14 PASS | Keep as a gate. Includes header stripping (D11) and the AF_UNIX boundary (D14). |
| E protocols | Linux/Docker 12 PASS | Keep as a gate. Includes the CONNECT TLS floor (E8). |
| F tamper/shutdown | Linux/Docker 15 PASS | Keep as a gate. The active-policy hash invariant plus F5/F6/F15/G9 shutdown cases are high-value regressions. |
| G load/failure | Linux/Docker 12 PASS | Keep as a gate. Covers DNS/collector failure, timeouts, exhaustion, recovery. |
| macOS/Windows host mode | CI only | Adopt only the subset the CI actually executes; do not claim netns/DNS/IPv6 paths. Windows WFP requires self-hosted hardware. |

---

## 6. Reproduction

### 6.1 Linux bubblewrap

```bash
cd warden-starter/warden
go build ./...
go vet ./...
go test -p 1 ./...
go vet ./testdata/egress/...
go test ./testdata/egress/...       # host/port normalization + F6 teardown tests
gofmt -l testdata/egress

CGO_ENABLED=0 go build -o /tmp/cceg/egress-runner ./testdata/egress/runner
/tmp/cceg/egress-runner \
  -backend linux -mode netns -build -require-complete \
  -out /tmp/cceg/final-linux
```

The runner re-execs under `unshare --user --map-root-user --net --mount
--propagation private`; it needs `bwrap`, `strace`, `unshare`, and `ip`. On
Ubuntu 24.04, install a scoped AppArmor profile for `bwrap`/`unshare` (the CI
workflow does this) rather than disabling the userns restriction host-wide.

### 6.2 Docker

```bash
# The image must exist locally; Warden does not pull implicitly.
docker image inspect alpine:3.20 >/dev/null 2>&1 || docker pull alpine:3.20

cd warden-starter/warden
newgrp docker -c 'cd "$PWD" && /tmp/cceg/egress-runner \
  -backend docker -image alpine:3.20 -mode netns -build -require-complete \
  -out /tmp/cceg/final-docker'
```

`newgrp docker` is needed when the current shell predates docker-group
membership; CI uses the same pattern. Both intended runs report 83 PASS, 0 FAIL,
0 WARN, 0 BLOCKED, 0 UNSUPPORTED, 0 NOT_RUN. `python3` on the host supplies F13.

### 6.3 macOS Seatbelt

```bash
cd warden-starter/warden
CGO_ENABLED=0 go build -o "$RUNNER_TEMP/egress-runner" ./testdata/egress/runner
"$RUNNER_TEMP/egress-runner" \
  -backend seatbelt -mode host -build \
  -out "$RUNNER_TEMP/egress-macos"
```

Prerequisites: `sandbox-exec` (hosted macOS has it) and `python3` for F13.
Hostname/DNS/IPv6 cases are UNSUPPORTED in host mode; do not report them as
PASS. Review the uploaded `report.md` before making any macOS claim.

### 6.4 Windows AppContainer/WFP

```powershell
cd warden-starter/warden
go build -o "$env:RUNNER_TEMP\egress-runner.exe" ./testdata/egress/runner
& "$env:RUNNER_TEMP\egress-runner.exe" `
  -backend windows -mode host -build `
  -out "$env:RUNNER_TEMP\egress-windows"
```

Hosted Windows runners cannot open WFP engine sessions; expect BLOCKED/NOT_RUN
and do not claim PASS. For real enforcement, run on self-hosted Windows hardware
with `-require-complete`.

### 6.5 Focused subsets

```bash
runner -backend linux  -mode netns -build -filter '^(F5|F6)$' -out /tmp/cceg/f5f6
runner -backend linux  -mode netns -build -group F -out /tmp/cceg/f
runner -backend docker -image alpine:3.20 -mode netns -build \
  -filter '^(A4|C10|D11|D14|E8|F5|F6|F11)$' -out /tmp/cceg/key
```

### 6.6 CI

`.github/workflows/egress-integrity.yml` runs:

- Linux bubblewrap: full netns matrix with `-require-complete`;
- Docker: full netns matrix with `-require-complete`;
- macOS Seatbelt: host-mode matrix (netns cases UNSUPPORTED);
- Windows hosted: capability report (BLOCKED/NOT_RUN);
- Windows WFP self-hosted: gated on `vars.WARDEN_WFP_SELF_HOSTED == 'true'`,
  with `-require-complete`.

All jobs upload `run.json`, `results.jsonl`, `summary.json`, `report.md`, and
per-case evidence as artifacts. A FAIL makes the runner exit non-zero;
WARN/UNSUPPORTED/BLOCKED/NOT_RUN are never converted to PASS.

---

## 7. Commands run for this evidence

```bash
cd warden-starter/warden
go build ./...
go vet ./...
go test -p 1 ./...                  # all packages pass serially
go vet ./testdata/egress/...
go test ./testdata/egress/...       # runner tests (incl. TestKillFailClosed)
gofmt -l testdata/egress            # no output
CGO_ENABLED=0 go build -o /tmp/cceg/egress-runner ./testdata/egress/runner

/tmp/cceg/egress-runner -backend linux -mode netns -build \
  -require-complete -out /tmp/cceg/final-linux

newgrp docker -c 'cd .../warden-starter/warden && /tmp/cceg/egress-runner \
  -backend docker -image alpine:3.20 -mode netns -build \
  -require-complete -out /tmp/cceg/final-docker'

GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build ./...
```

---

## 8. Files

Harness (all under `warden-starter/warden/testdata/egress/`):

- `matrix.yaml` — 83-case matrix.
- `policies/*.yaml.tmpl` — nine policy fixtures, including
  `allow-host-denied-port.yaml.tmpl` for A4.
- `runner/` — matrix loader, case execution, collectors, DNS, evidence, policy
  rendering, checks (`killFailClosed` pre-signal teardown handling), special
  cases, netns setup, platform process helpers.
- `probe/` — HTTP/HTTPS/TCP/UDP/DNS/DoT/DoH/WebSocket/SSE/MCP/flood/tamper/
  signal/interpreter probe actions.

Documentation and CI:

- `EGRESS-INTEGRITY.md` — this report.
- `warden-starter/warden/testdata/egress/README.md` — maintainer usage, matrix
  schema, evidence layout, and known limitations.
- `.github/workflows/egress-integrity.yml` — Linux, Docker, macOS, Windows, and
  self-hosted WFP jobs.

Product code changed (under `warden-starter/warden/`; no public API change):

- `internal/policy/policy.go` — `splitHostPortRule`/`validatePort` for
  port-qualified `network.allow` entries.
- `internal/proxy/proxy.go` — hop-by-hop header stripping (D11), port-qualified
  grant enforcement (A4), ClientHello TLS 1.2 floor for CONNECT (E8).
- `internal/proxy/proxy_test.go` — D11, A4, and E8 regression tests.
- `internal/policy/policy_test.go` — port-rule parsing/validation tests.
- `internal/sandbox/docker/docker.go` — `--init` (F5); `hostUIDGID` for the
  mapped host user.
- `internal/sandbox/docker/userns*.go` — `/proc/self/{uid,gid}_map` resolution
  that fails closed when the map is unavailable or the ID is unmapped.
- `internal/sandbox/docker/{docker,integration,userns}*_test.go` — mapped-user
  assertions and the 0700 write-grant regression trap.

Docs updated in the same change: `ARCHITECTURE.md`, `docs/schema.md`,
`docs/architecture.md`, `docs/approve.md`, `docs/compatibility.md`,
`docs/container-k8s.md`, `docs/examples.md`, `examples/policy.example.yaml`.

---

## 9. Final statement

**Linux bubblewrap:** YES for the tested network-policy contract — 83 PASS,
0 FAIL, 0 WARN, 0 BLOCKED, 0 UNSUPPORTED, 0 NOT_RUN. Run
`r1792834593384b49`, revision `9d166b0`.

**Docker:** YES for the tested network-policy contract — 83 PASS, 0 FAIL,
0 WARN, 0 BLOCKED, 0 UNSUPPORTED, 0 NOT_RUN. Run `r1792834620dd73b4`, revision
`9d166b0`. The container resolves `/proc/self/{uid,gid}_map`, runs as the mapped
host user, uses `--init`, and keeps `--network none` / `--cap-drop ALL`.

**macOS Seatbelt:** INCONCLUSIVE / not executed locally. **Windows
AppContainer/WFP:** INCONCLUSIVE / not executed locally; hosted CI cannot run
WFP. **Do not claim a cross-platform YES.**

Every previously WARN or UNSUPPORTED property (D11, D14, E8, A4, C10/F11, F5,
plus F6) is either enforced now or explicitly reclassified with a documented
boundary, and each is backed by an executed PASS on both backends. The F6 change
is a harness correctness fix: the workload was genuinely terminated, and the
check still refuses to accept a surviving workload as a PASS.
