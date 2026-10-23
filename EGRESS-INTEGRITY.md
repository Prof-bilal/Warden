# Egress and Enforcement Integrity

Evidence report for Warden's deny-by-default network policy across the Linux
bubblewrap and Docker backends, with capability/CI wiring for macOS Seatbelt
and Windows AppContainer/WFP.

- **Date:** 2026-10-23
- **Branch:** `ctx/egress-integrity`
- **Warden revision tested:** `ba53732` (`fix(docker): fail closed when uid/gid
  mapping is unavailable`), on top of `55cac76` (mapped host user) and suite
  commit `2c40cab`
- **Host:** Linux x86_64; bubblewrap 0.13.1; strace 7.2; Docker 29.9.2;
  `alpine:3.20` (`sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc`)
- **Harness:** `warden-starter/warden/testdata/egress/` (external Go runner +
  probe; the harness itself does not change product code)
- **Product change:** the Docker backend now resolves
  `/proc/self/{uid,gid}_map` before passing `--user`; see §4.1
- **Matrix:** 83 cases in groups A–G (`testdata/egress/matrix.yaml`)

This report follows the evidence-first rule: **WARN, UNSUPPORTED, BLOCKED and
NOT_RUN are never counted as PASS**, and no platform is reported as PASS unless
it was actually executed. No real secrets, production endpoints, or host
network/firewall changes were used. All collectors and DNS are local.

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
- **Docker.** The container runs with `--network none`, `--cap-drop ALL`, a
  read-only root, and a tmpfs `/tmp`. The only egress path is the Warden
  proxy bridge over a bind-mounted Unix socket. The Docker matrix was run in
  the same netns harness. The backend resolves
  `/proc/self/{uid,gid}_map` and runs the container as the mapped host user;
  the earlier test-only socket-permission workaround has been removed. F1
  (write-grant control) and F13 (Python) pass.

### 1.3 Matrix groups

| Group | Cases | Focus |
|---|---:|---|
| A | 10 | Allow/deny by hostname and IP, redirects, unlisted/rebinding names |
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
| Linux bubblewrap (netns) | 83 | 79 | 0 | 3 | 0 | 1 | 0 | **YES** (scoped; WARN/UNSUPPORTED excluded) |
| Docker (netns) | 83 | 76 | 0 | 4 | 0 | 3 | 0 | **YES** (scoped; WARN/UNSUPPORTED excluded) |
| macOS Seatbelt | — | — | — | — | — | — | — | **UNTESTED locally / INCONCLUSIVE** |
| Windows AppContainer/WFP | — | — | — | — | — | — | — | **UNTESTED locally / INCONCLUSIVE** |

Final Linux run: `r1792780446dcb302` (2026-10-23T18:34:07Z →
18:34:32Z), mode `netns`, privilege `uid=0` inside the unprivileged user
namespace, Warden revision `ba53732`. Evidence:
`/tmp/opencode/egress-linux-final/`.

Final Docker run: `r179278037639fd5b` (2026-10-23T18:32:57Z →
18:34:06Z), mode `netns`, Warden revision `ba53732`, with
`-require-complete`. Evidence: `/tmp/opencode/egress-docker-final/`.

Each evidence directory contains `run.json`, `results.jsonl`, `summary.json`,
`report.md`, `policies/<case>.yaml`, `cases/<case>/{policy.yaml,result.json,steps.json,probe-events.json,collector-hits.json,dns-hits.json,audit-events.json,checks.json,*.stdout.log,*.stderr.log}`,
and `collectors/{collector-hits.jsonl,dns.jsonl}`.

### 2.2 Linux bubblewrap — group breakdown

| Group | PASS | WARN | UNSUPPORTED | Notes |
|---|---:|---:|---:|---|
| A | 9 | 0 | 1 | A4: schema has no ports |
| B | 10 | 0 | 0 | DNS/DoT/DoH/rebinding/normalization/wildcards |
| C | 10 | 0 | 0 | Raw TCP/UDP, IPv6, subprocess, new netns |
| D | 12 | 2 | 0 | D11 Proxy-Authorization; D14 AF_UNIX |
| E | 11 | 1 | 0 | E8 TLS < 1.2 |
| F | 15 | 0 | 0 | Tamper, helpers, buffers, supervisor kill |
| G | 12 | 0 | 0 | Load, failure, timeout |

Key positive evidence:

- **A1/A2:** allowed HTTP/HTTPS reached the intended collector with a 2xx,
  a collector hit, and an audit `action=connect`.
- **A3/A5–A10:** denied, unlisted, rebinding, direct-IP, and redirect cases
  produced explicit denials (403/502 or the expected sandbox error) and no
  forbidden collector hits.
- **B1–B10:** DNS queries followed the intended resolver path; denied names
  produced no usable resolution; DoT/DoH to unauthorized endpoints failed;
  mixed-case/trailing-dot and wildcard/apex/lookalike variants matched policy.
- **C1–C10:** direct TCP/UDP, IPv4/IPv6, loopback, private, metadata, and
  subprocess-created sockets failed; `unshare` from inside the sandbox was
  denied on the Linux backend.
- **D1–D10, D12–D13:** direct connections, alternate proxies, CONNECT,
  absolute-form, SOCKS-style requests, two-hop redirects, URL userinfo,
  numeric IP formats, second proxies, and the Unix control socket were
  denied. D11 and D14 are the two WARN limitations below (proxy auth-header
  leak; AF_UNIX reachable in a granted path).
- **E1–E7, E9–E12:** POST bodies, WebSocket, SSE, MCP stdio/HTTP/SSE,
  certificate validation, Host/SNI mismatch denials, HTTP/2 (when
  negotiated), and chunked/streamed bodies stayed policy-conforming. E8 is
  the TLS-floor WARN; E10 passes after the harness Host/port normalization
  described in §4.2.
- **F1–F15:** policy read/write/replace, audit write, proxy env changes,
  resolver/hosts tampering, second processes, fresh helper binaries, buffer
  pressure, and supervisor kill all failed closed. F5/F6 show the workload was
  terminated (exit 137/255) when the enforcement process received the signal;
  F15/G9 show the in-flight request was lost when the supervisor was killed.
  Every case also passed `active_policy_unchanged`.
- **G1–G12:** high-rate DNS, short/concurrent connections, long streams,
  large bodies, connection exhaustion, DNS/collector failure, and timeouts
  stayed fail-closed; recovery worked when DNS/collectors returned.

### 2.3 Docker — group breakdown

| Group | PASS | WARN | BLOCKED | UNSUPPORTED | NOT_RUN | Notes |
|---|---:|---:|---:|---:|---:|---|
| A | 9 | 0 | 0 | 1 | 0 | A4 schema |
| B | 10 | 0 | 0 | 0 | 0 | DNS/DoT/DoH/rebinding |
| C | 9 | 0 | 0 | 1 | 0 | C10 Linux-only |
| D | 12 | 2 | 0 | 0 | 0 | D11, D14 |
| E | 11 | 1 | 0 | 0 | 0 | E8 |
| F | 13 | 1 | 0 | 1 | 0 | F1/F13 PASS, F5 WARN, F11 Linux-only |
| G | 12 | 0 | 0 | 0 | 0 | Load/failure/timeout |

The Docker container→bridge→proxy→policy path passed 76 cases, including
hostname allow/deny, DNS/DoT/DoH, IPv4/IPv6, redirects, CONNECT, WebSocket,
SSE, MCP, TLS/cert, tamper, the F1 write-grant control, the F13 Python
interpreter, load, and supervisor-kill cases. The non-PASS items are listed in
§4; none is a network-policy bypass.

---

## 3. Per-backend answer

The exact question: **“Can Warden currently demonstrate that every available
network path obeys policy on the tested backend?”**

### Linux bubblewrap — YES, for the tested network-policy contract

- 79/83 cases PASS; 0 FAIL; 0 BLOCKED; 0 NOT_RUN.
- The tested paths are the ones the suite can exercise: HTTP/HTTPS, CONNECT,
  redirects, POST/streaming, WebSocket/SSE, MCP, DNS/DoT/DoH, raw TCP/UDP,
  IPv4/IPv6, loopback/private/metadata, subprocesses, new netns, proxy bypass,
  policy/audit tampering, and enforcement shutdown.
- The 3 WARN cases and 1 UNSUPPORTED case are **excluded** from this claim and
  are not counted as PASS. If “every available path” is read to include
  TLS-version enforcement, SNI binding for CONNECT/HTTPS,
  Proxy-Authorization stripping, AF_UNIX policy, or port-granular grants, the
  answer is **INCONCLUSIVE** for those properties; see §4.

### Docker — YES, for the tested network-policy contract

- 76/83 cases PASS; 0 FAIL; 0 BLOCKED; 0 NOT_RUN; 3 UNSUPPORTED
  (A4, C10, F11); 4 WARN (D11, D14, E8, F5).
- The 76 PASS cases demonstrate the container→bridge→proxy→policy path for the
  tested network paths, including the F1 write-grant control and the F13 Python
  interpreter. F1 verifies that the container runs as the mapped host user and
  can write the granted control path while the active policy remains unreadable;
  F13 verifies that an in-sandbox Python interpreter can use the allowlisted
  URL and cannot open the denied socket.
- The Docker backend now resolves `/proc/self/{uid,gid}_map` before passing
  `--user` to Docker. The previous `--user 0:0`/host-root defect is fixed, and
  the harness no longer relaxes the test proxy socket permissions. The
  container keeps `--network none`, `--cap-drop ALL`, and the bridge-only
  egress path.
- The full backend claim is **YES for the tested network-policy contract**:
  every Docker-applicable case ran, with 0 FAIL/BLOCKED/NOT_RUN. The 3
  UNSUPPORTED cases are schema/backend applicability limits (A4 has no port
  field; C10/F11 are Linux-bubblewrap-only), and the 4 WARN cases are excluded
  from the claim and are not counted as PASS.
- If “every available path” is read to include TLS-version enforcement, SNI
  binding for CONNECT/HTTPS, Proxy-Authorization stripping, AF_UNIX policy,
  port-granular grants, or Docker in-container PID-1 signal semantics, the
  answer is **INCONCLUSIVE** for those properties; see §4.

### macOS Seatbelt — INCONCLUSIVE / not executed locally

- No macOS host was available in this environment. The runner and probe
  cross-compile for `darwin/arm64`; the CI workflow runs the Seatbelt backend
  in host mode on `macos-latest`.
- Host mode uses the allowlisted IP literal `127.0.0.1` and the LAN IP as the
  denied target. Hostname/DNS/IPv6 cases require the Linux netns harness and
  are reported **UNSUPPORTED** on macOS; they are never PASS.
- Do not claim macOS Seatbelt success until the CI evidence is reviewed.

### Windows AppContainer/WFP — INCONCLUSIVE / not executed locally

- No Windows host was available in this environment. The runner and probe
  cross-compile for `windows/amd64`; the CI workflow runs the Windows backend
  in host mode on `windows-latest`.
- GitHub-hosted Windows runners ship `fwpuclnt.dll` but refuse WFP engine
  sessions (see `.github/workflows/ci.yml`), so the Windows backend refuses to
  run and the suite records **BLOCKED/NOT_RUN**, not PASS.
- Real WFP enforcement requires self-hosted Windows hardware. The workflow
  contains a `windows-wfp-self-hosted` job gated on
  `vars.WARDEN_WFP_SELF_HOSTED == 'true'`, with `-require-complete` so a real
  run cannot silently degrade into a capability report.

### Overall

**INCONCLUSIVE across platforms.** Linux bubblewrap and Docker are **YES for
the tested network-policy contract** on this host, with the documented WARN and
UNSUPPORTED exclusions; macOS and Windows are untested locally and must not be
reported as PASS. No cross-platform YES is claimed.

---

## 4. Findings, blockers, and remediation

### 4.1 Docker under an unprivileged user namespace — mapped host user (FIXED)

**Observed before the fix.** When Warden ran inside
`unshare --user --map-root-user`, its `os.Getuid()`/`os.Getgid()` returned 0.
The Docker backend passed `--user 0:0` to `docker run`. The host proxy socket
directory was created 0700 and owned by the mapped host user; the granted write
path was also owned by that user. A root container with `--cap-drop ALL` could
not open the 0700 socket or write the user-owned grant. The harness opened the
**test** proxy socket to 0777 so the container could reach the bridge; F1’s
control write still failed by uid, so F1 was **BLOCKED** rather than PASS or
FAIL. This was a security/functional defect (container running as host root),
not a network-policy bypass.

**Fix.** `BuildDockerArgs` now calls `hostUIDGID` instead of using
`os.Getuid()`/`os.Getgid()` directly:

- `internal/sandbox/docker/userns.go` parses the kernel’s
  `inside outside length` map format and maps a namespace ID to its host ID.
- `internal/sandbox/docker/userns_linux.go` reads
  `/proc/self/uid_map` and `/proc/self/gid_map` and returns an error if the
  maps cannot be read/parsed or do not contain the current ID.
  `BuildDockerArgs` propagates that error, so the backend fails closed rather
  than silently launching the container as namespace root.
- `internal/sandbox/docker/userns_other.go` keeps the existing identity
  behavior on non-Linux platforms and omits `--user` on Windows.
- `internal/sandbox/docker/userns_linux_test.go` covers the successful
  identity/user-namespace mappings and the missing, malformed, and unmapped-ID
  error paths.
- The Docker integration test’s 0700 write directory remains a regression
  trap: if the mapping regresses, the container cannot write it.

**Verification.** Inside `newgrp docker` plus
`unshare --user --map-root-user --net --mount --propagation private`, the
mapping logged `1000:966` (namespace `0:0` to host user/group), and the full
`go test ./internal/sandbox/docker` suite passed, including the Docker
integration test. The egress harness’s test-only socket chmod was removed.
The final Docker matrix (`r179278037639fd5b`, `-require-complete`) reports
**76 PASS, 0 FAIL, 0 BLOCKED, 0 NOT_RUN, 4 WARN, 3 UNSUPPORTED, verdict YES**;
F1 passes with `control_write` and `policy_read_closed`, and F13 passes on
Docker. Evidence: `cases/F1/checks.json` and `cases/F13/checks.json` under
`/tmp/opencode/egress-docker-final/`.

**Remaining boundary.** This fix does not change the Docker network boundary:
the container still has `--network none`, `--cap-drop ALL`, and only the
bridge-mounted proxy socket. The F5 PID-1 signal WARN remains (see §4.7).

### 4.2 E10 — mismatched Host header: harness false positive corrected (PASS)

**Observed.** The first E10 run reported WARN because the check compared the
collector’s `Host` header, which includes the ephemeral destination port
(`e10.<run>.egress.test:<port>`), to the hostname-only policy value
(`e10.<run>.egress.test`). The upstream collector actually received the
**authorized** URL host, not the client’s `Host: denied-e10...`; the request
seen upstream carried the authorized host. The denied step returned 403.

**Correction.** `hostMismatchObserved` now normalizes case, a trailing dot,
and an optional port before comparison; `runner/checks_test.go` covers the
normalization. E10 is PASS on Linux and Docker; the E10 correction itself did
not require a product change. The check still warns if the collector ever
receives a Host that differs from the allowlisted host.

**Remaining boundary.** E10 covers plain HTTP request-target authorization.
HTTPS/CONNECT is an opaque tunnel: Warden does not inspect SNI or the inner
Host, so SNI binding for CONNECT is not claimed. Document that boundary; if
SNI-aware policy is required, it needs TLS interception or SNI inspection.

### 4.3 D11 — Proxy-Authorization forwarded upstream (WARN)

**Observed.** The proxy forwarded the client’s `Proxy-Authorization` header to
the upstream collector. The proxy does not use authentication, so the header
has no authorization effect, but it can leak credentials.

**Remediation.** Strip `Proxy-Authorization` (and hop-by-hop proxy headers)
before forwarding; never log the value. Add a regression test.

### 4.4 E8 — TLS below 1.2 negotiated (WARN)

**Observed.** The TLS collector negotiated `0x0302` (TLS 1.1) through the
proxy. The proxy is a blind CONNECT tunnel, so the TLS version is chosen by
the client and server; Warden does not enforce a TLS floor.

**Why it matters.** The repository context expects TLS 1.2 or higher. That is
not a property the proxy currently enforces. The destination remains
allowlisted.

**Remediation.** Either document that the TLS floor is a client-side property
and remove/qualify any claim that Warden enforces it, or implement TLS
interception/SNI-aware policy (a large design change with CA management).
At minimum, make the expectation explicit in the docs and keep E8 as a WARN.

### 4.5 D14 — AF_UNIX socket in a granted path is reachable (WARN)

**Observed.** An AF_UNIX socket placed in a granted filesystem path was
reachable from the sandbox. Network policy does not govern AF_UNIX; the
filesystem grant is the boundary. The proxy’s Unix control socket itself
denied the non-allowlisted host with 403 (`unix_proxy_denied`).

**Remediation.** Document that AF_UNIX is outside network policy and that a
writable/readable grant on a directory containing host service sockets is a
trust boundary. Optionally, deny AF_UNIX connections outside explicitly
granted paths via a syscall filter or a Unix-socket proxy; this is a design
change.

### 4.6 A4 — policy grants hostnames, not ports (UNSUPPORTED)

**Observed.** A second service on the same allowlisted hostname but a
different port was reachable. The schema has no port field, so “allowed
hostname on a denied port” cannot be expressed or enforced.

**Remediation.** If port-granular policy is required, add optional
`host:port`/`ports` syntax to the policy schema. Per `AGENTS.md`, this requires
updating `ARCHITECTURE.md` and `examples/policy.example.yaml` in the same
change, plus tests. Otherwise, document that grants are host-wide.

### 4.7 F5 — Docker in-container bridge ignores unhandled signals (WARN)

**Observed.** The in-container bridge is PID 1 of the container’s PID
namespace. Signals it does not handle (notably SIGKILL from inside) do not
terminate it; traffic stayed proxied and policy-conforming. Supervisor kill
fail-closed behavior is covered by F15/G9, which passed on Docker.

**Remediation.** Document the PID-1 semantics. SIGKILL cannot be handled;
SIGTERM can be handled with an explicit shutdown path. No security bypass is
demonstrated.

### 4.8 F13 — Python case now runs on Docker (PASS)

**Observed.** F13 was previously restricted to `linux`, `seatbelt`, and
`windows`; Docker recorded NOT_RUN because `alpine:3.20` has no Python. The
Docker backend mounts the host `/usr` runtime base, so the host `python3` is
available inside the container. The matrix now includes Docker for F13, and
the final Docker run passes `interpreter_allowed` and `interpreter_denied`.

**Remaining boundary.** If the host has no `python3`, F13 is NOT_RUN and
`-require-complete` fails. The Docker CI job therefore needs `python3` on the
runner; GitHub `ubuntu-latest` provides it.

### 4.9 C10/F11 — Linux-bubblewrap-only cases on Docker (UNSUPPORTED)

`C10` (child cannot create a new network namespace) and `F11` (unshare cannot
create an unrestricted path) require the Linux bubblewrap backend’s namespace
setup. They are not applicable to Docker and are reported UNSUPPORTED, not
PASS. They passed on Linux.

### 4.10 macOS/Windows host-mode limits

Host mode has no controllable test DNS or TEST-NET collectors. The allowed
destination is the IP literal `127.0.0.1`; hostname, DNS, and IPv6 cases are
UNSUPPORTED. Windows WFP enforcement additionally requires self-hosted
hardware. These are environment limitations, not product failures; they are
reported honestly and never counted as PASS.

---

## 5. Adoption recommendations for meaningful PASSes

| Area | Evidence | Recommendation |
|---|---|---|
| A/B allowlist + DNS | Linux 19 PASS (A1–A3, A5–A10; B1–B10); A4 UNSUPPORTED | Make a CI gate. Covers HTTP/HTTPS allow/deny, unlisted/rebinding names, direct IP, DoT/DoH, normalization, wildcard/apex/lookalike. |
| C raw sockets | Linux 10/10 PASS | Keep as a gate. Proves no direct TCP/UDP/IPv6/loopback/private/metadata egress, subprocess sockets are covered, and new netns is denied on Linux. |
| D proxy bypass/tamper | Linux 12 PASS + 2 WARN | Keep as a gate; track D11/D14 as hardening items. |
| E protocols | Linux 11 PASS + 1 WARN | Keep as a gate; track E8 (TLS floor). |
| F tamper/shutdown | Linux 15/15 PASS | Keep as a gate. The active-policy hash invariant and F15/G9 supervisor-kill cases are high-value regressions. |
| G load/failure | Linux 12/12 PASS | Keep as a gate. Covers DNS/collector failure, timeouts, exhaustion, and recovery. |
| Docker container path | 76 PASS; F1/F13 PASS | Adopt as a Docker regression gate now. The userns/uid defect is fixed; `-require-complete` rejects BLOCKED/NOT_RUN. The 4 WARN and 3 UNSUPPORTED cases are excluded from the claim. |
| macOS/Windows host mode | CI only | Adopt only the host-mode subset the CI actually executes; do not claim netns/DNS/IPv6 paths. Windows WFP requires self-hosted hardware. |

---

## 6. Reproduction

### 6.1 Local Linux bubblewrap

```bash
cd warden-starter/warden
go build ./...
go vet ./...
go test -p 1 ./...
go vet ./testdata/egress/...
go test ./testdata/egress/...       # includes the host/port normalization test
gofmt -l testdata/egress

CGO_ENABLED=0 go build -o /tmp/opencode/egress-runner ./testdata/egress/runner
/tmp/opencode/egress-runner \
  -backend linux -mode netns -build -require-complete \
  -out /tmp/opencode/egress-linux-final
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
newgrp docker -c 'cd "$PWD" && /tmp/opencode/egress-runner -backend docker -image alpine:3.20 -mode netns -build -require-complete -out /tmp/opencode/egress-docker-final'
```

`newgrp docker` is needed when the current shell predates docker-group
membership; CI uses the same pattern. The Docker run reports 76 PASS, 0 FAIL,
0 BLOCKED, 0 NOT_RUN, 4 WARN (D11, D14, E8, F5), and 3 UNSUPPORTED
(A4, C10, F11); `-require-complete` enforces the absence of BLOCKED/NOT_RUN.
F1 and F13 pass. The backend resolves `/proc/self/{uid,gid}_map`, so the
container runs as the mapped host user; `python3` on the host supplies F13.

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
and do not claim PASS. For real enforcement, run on self-hosted Windows
hardware and use `-require-complete`:

```powershell
& "$env:RUNNER_TEMP\egress-runner.exe" `
  -backend windows -mode host -build -require-complete `
  -out "$env:RUNNER_TEMP\egress-windows-wfp"
```

### 6.5 Focused subsets

```bash
# One case, group, or regexp
runner -backend linux -mode netns -build -filter '^A1$' -out /tmp/egress-a1
runner -backend linux -mode netns -build -group F -out /tmp/egress-f
runner -backend docker -image alpine:3.20 -mode netns -build -filter '^(A|B)[0-9]+$' -out /tmp/egress-ab
```

### 6.6 CI

`.github/workflows/egress-integrity.yml` runs:

- Linux bubblewrap: full netns matrix with `-require-complete`;
- Docker: full netns matrix with `-require-complete`; the backend resolves
  `/proc/self/{uid,gid}_map`, F1/F13 pass, and the Docker package tests run;
- macOS Seatbelt: host-mode matrix (netns cases UNSUPPORTED);
- Windows AppContainer/WFP: hosted capability report (BLOCKED/NOT_RUN);
- Windows WFP self-hosted: gated on `vars.WARDEN_WFP_SELF_HOSTED == 'true'`,
  with `-require-complete`.

All jobs upload `run.json`, `results.jsonl`, `summary.json`, `report.md`, and
per-case evidence as artifacts. The step summary includes the human-readable
report. A FAIL makes the runner exit non-zero; WARN/UNSUPPORTED/BLOCKED/NOT_RUN
are never converted to PASS.

---

## 7. Commands run for this evidence

```bash
cd warden-starter/warden
go build ./...
go vet ./...
go test -p 1 ./...                  # all packages pass serially
go vet ./testdata/egress/...
go test ./testdata/egress/...       # host/port normalization test
gofmt -l testdata/egress            # no output
CGO_ENABLED=0 go build -o /tmp/opencode/egress-runner ./testdata/egress/runner

/tmp/opencode/egress-runner -backend linux -mode netns -build \
  -require-complete -out /tmp/opencode/egress-linux-final

newgrp docker -c 'cd .../warden-starter/warden && /tmp/opencode/egress-runner -backend docker -image alpine:3.20 -mode netns -build -require-complete -out /tmp/opencode/egress-docker-final'

# Userns regression: the mapping must be the host ID (1000:966 on this host),
# not 0:0.
newgrp docker -c 'cd .../warden-starter/warden && unshare --user --map-root-user --net --mount --propagation private -- go test ./internal/sandbox/docker -count=1'

GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./...
```

An initial parallel `go test ./...` hit an environmental
`fork/exec ... resource temporarily unavailable`; the isolated failing test
passed, and the full serial suite (`-p 1`) passed. This was a local resource
limit, not a test failure.

---

## 8. Files

New harness (all under `warden-starter/warden/testdata/egress/`):

- `matrix.yaml` — 83-case matrix.
- `policies/*.yaml.tmpl` — nine policy fixtures.
- `runner/` — matrix loader, case execution, collectors, DNS, evidence,
  policy rendering, checks, special cases, netns setup, platform process
  helpers.
- `probe/` — HTTP/HTTPS/TCP/UDP/DNS/DoT/DoH/WebSocket/SSE/MCP/flood/tamper/
  signal/interpreter probe actions.

Documentation and CI:

- `EGRESS-INTEGRITY.md` — this report.
- `warden-starter/warden/testdata/egress/README.md` — maintainer usage, matrix
  schema, evidence layout, and known limitations.
- `.github/workflows/egress-integrity.yml` — Linux, Docker, macOS, Windows,
  and self-hosted WFP jobs.

Product code changed (under `warden-starter/warden/`; Docker userns fix; no public API or policy schema change):

- `internal/sandbox/docker/userns.go` — kernel uid/gid map parser and mapper.
- `internal/sandbox/docker/userns_linux.go` — `/proc/self/{uid,gid}_map`
  resolution that fails closed if the map is unavailable or the ID is unmapped.
- `internal/sandbox/docker/userns_other.go` — non-Linux identity behavior.
- `internal/sandbox/docker/userns_test.go` — parser/mapping regression tests.
- `internal/sandbox/docker/userns_linux_test.go` — `mapCurrentID` success and
  missing/malformed/unmapped-ID error paths.
- `internal/sandbox/docker/docker.go` — `BuildDockerArgs` uses `hostUIDGID`
  and propagates mapping errors.
- `internal/sandbox/docker/docker_test.go`,
  `internal/sandbox/docker/integration_test.go` — mapped-user assertions and
  the 0700 write-grant regression trap.

---

## 9. Final statement

**Linux bubblewrap:** YES for the tested network-policy contract — 79 PASS,
0 FAIL, 0 BLOCKED, 0 NOT_RUN — with three documented WARN limitations
(D11 Proxy-Authorization, D14 AF_UNIX, E8 TLS < 1.2) and one schema
UNSUPPORTED (A4 ports) explicitly excluded and never counted as PASS. E10 was
a harness false positive (the check compared the collector’s Host with its
ephemeral port to the hostname-only policy value); after normalization it
passes, and the proxy correctly forwarded the authorized Host.

**Docker:** YES for the tested network-policy contract — 76 PASS, 0 FAIL,
0 BLOCKED, 0 NOT_RUN — with four documented WARN limitations (D11
Proxy-Authorization, D14 AF_UNIX, E8 TLS < 1.2, F5 Docker PID-1 signal
semantics) and three UNSUPPORTED cases (A4 ports, C10/F11 Linux-only)
explicitly excluded and never counted as PASS. F1 passes with
`control_write`/`policy_read_closed`, and F13 passes with
`interpreter_allowed`/`interpreter_denied`; the backend resolves
`/proc/self/{uid,gid}_map` and the test-only socket chmod has been removed.

**macOS Seatbelt:** INCONCLUSIVE / not executed locally. **Windows
AppContainer/WFP:** INCONCLUSIVE / not executed locally; hosted CI cannot WFP.
Do not claim a cross-platform YES.
