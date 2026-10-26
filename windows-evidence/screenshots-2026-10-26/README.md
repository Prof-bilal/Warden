# Warden Windows end-to-end evidence (2026-10-26)

Real Windows 11 machine, native backend (AppContainer + WFP + ETW + Job Object). Checklist: `WINDOWS-TESTING.md`.
Each step has `NN-name.txt`, the full raw output and the source of truth. Steps 01-30 also have `NN-name.png`, a capture of the console window that ran the command; long outputs show only their tail. `28b` has text only, because screenshots were stopped on request.

## Follow-up: failures fixed in b9394f7

Both failures below were fixed in commit `b9394f7` and re-verified on the same machine the same day:

- **Step 10:** the executable-directory grant is now optional. A non-elevated `WARDEN_WINDOWS_UNPRIVILEGED=1` run of `C:\Program Files
odejs
ode.exe` starts. The granted `output\` write succeeds, and every read/write outside the policy is still EPERM.
- **Step 28b:** ETW file requests are matched to their `OperationEnd` NTSTATUS. Denied opens of `C:\warden-outside\*` are now logged `"allowed":false` with reason `NTSTATUS 0xC0000022 STATUS_ACCESS_DENIED`. The new elevated test `TestEtwDeniedFileAudited` passes, as do all Windows backend tests.
- Still open: direct egress blocked by WFP has no audit record. Only proxy denials are logged.

The table below is the original run at commit 6bdf8a5.

## Machine

| Item | Value |
|---|---|
| Windows | Windows 11 Home (EditionID Core), 26H2, build 10.0.26300.9711 |
| Arch | x64 (AMD64) |
| User | HUSNAIN\husna, member of Administrators (deny-only when not elevated). Steps 01-14 ran NON-elevated. Steps 15-30 ran ELEVATED after one UAC approval (two approvals in total). |
| Go / Node / npm / Git | go1.27.0 windows/amd64 / v24.12.0 / 11.6.2 / 2.52.0.windows.1 |
| Published Warden | warden-sandbox-cli 0.1.16 (`warden version v0.1.16`). Upgraded from 0.1.6 by step 03. |
| Source Warden | `warden version dev`, branch fix/windows-sandbox-e2e, commit 6bdf8a5a6e4d67a7c8bf8ddb1d9eaea49568e801, exe SHA256 2182CBD9...D72C |
| Docker | Not running (optional comparison skipped) |
| Test data | C:\warden-test (grants), C:\warden-outside (ungranted control). Only fake tokens. |

## Results

| Step | Command (short) | Expected | Actual | Result | Screenshot |
|---|---|---|---|---|---|
| 01 | `ver`, Get-ComputerInfo | record OS | Win11 Home 26H2 build 26300.9711 x64 | PASS | [01](01-winver.png) |
| 02 | go/node/npm/git/whoami | prereqs present | all present; not elevated | PASS | [02](02-go-node-versions.png) |
| 03 | `npm install -g warden-sandbox-cli; warden --version` | installs | 0.1.16 installed, v0.1.16 | PASS | [03](03-published-install.png) |
| 04 | `warden doctor` (published, non-admin) | fail closed without admin | NOT READY, "needs administrator", exit 1, no fallback | PASS (published doctor was not re-run elevated) | [04](04-published-doctor.png) |
| 05 | `go build -o warden-windows-amd64.exe .\cmd\warden` | builds | exit 0 | PASS | [05](05-build.png) |
| 06 | `warden-windows-amd64.exe --version` | prints version | `warden version dev` @ 6bdf8a5 | PASS | [06](06-version.png) |
| 07 | `doctor` (source, non-admin) | fail closed | NOT READY, exit 1 | PASS | [07](07-doctor.png) |
| 08 | `go test -v ./internal/sandbox/windows/...` non-admin | pass; skips give reasons | ok; 2 SKIPs with reasons (no AppContainer without admin / unprivileged-only test) | PASS | [08](08-go-test-windows-nonadmin.png) |
| 09 | `warden run --backend windows` non-admin, no opt-in | refuse, ask for admin | "needs administrator permission; set WARDEN_ELEVATE=1", exit 1 | PASS | [09](09-run-nonadmin-needs-elevation.png) |
| 10 | `WARDEN_WINDOWS_UNPRIVILEGED=1` run with `C:\Program Files\nodejs\node.exe` | runs in unprivileged mode | doctor says READY, but run fails: `grant filesystem access to "C:\Program Files\nodejs": SetNamedSecurityInfoW: Access is denied`. Without stdin redirect it waits forever at the `[y/N]` prompt | **FAIL** | [10](10-unprivileged-run-programfiles-node.png) |
| 11 | unprivileged run, node copied to C:\warden-test\bin | fs/env enforced, all network blocked | reads/writes in grant OK; 8 out-of-grant ops EPERM; fake secrets ABSENT; direct TCP EACCES, DNS ENOTFOUND, proxy unreachable; audit says ETW is off | PASS | [11](11-unprivileged-run.png) |
| 12 | unprivileged + policy with network.allow | refuse | refused: "network.allow needs WFP egress filtering...", exit 1 | PASS | [12](12-unprivileged-network-allow-refused.png) |
| 13 | `bash test-mcp-proxy.sh <exe>` | proxy contract | 6 passed, 0 failed | PASS | [13](13-mcp-proxy.png) |
| 14 | `docker info` | optional Docker comparison | Docker Desktop engine not running | SKIP (Docker not running) | [14](14-docker-fallback.png) |
| 15 | `doctor` (elevated) | READY, fail-closed | READY: AppContainer, fs/env/network policy, fail-closed enforced | PASS | [15](15-doctor-admin.png) |
| 16 | `go test -v ./internal/sandbox/windows/...` elevated | pass | ok; 36 PASS, 1 SKIP (unprivileged-only test, reason given) | PASS | [16](16-go-test-windows-admin.png) |
| 17 | `go test -v -run 'TestEtw\|TestEscape\|TestAppContainer\|TestWFP'` | discovered and pass | 13 tests ran: ETW lifecycle/reclaim/file audit, escape read/network/timeout, WFP, all PASS (1 unprivileged-only SKIP) | PASS | [17](17-go-test-targeted-admin.png) |
| 18 | `go test -count=1 ./...` elevated | all pass | 22 packages ok, exit 0 | PASS | [18](18-go-test-all-admin.png) |
| 19 | native run, checklist policy, all probes | grants work, everything else blocked | all 24 probe results as expected, exit 0 | PASS | [19](19-native-run.png) |
| 20 | probe fs-read | in-grant read OK, outside blocked | data/hello.txt OK; C:\warden-outside, user profile, Desktop, repo file EPERM (host control can read secret) | PASS | [20](20-fs-read-blocked.png) |
| 21 | probe fs-write | write in output\ only | output\ OK; read-only grant, outside dir, profile, C:\Windows\Temp EPERM; host check: no stray files | PASS | [21](21-fs-write-blocked.png) |
| 22 | probe net-blocked | direct egress and unlisted host blocked | 1.1.1.1:443 and GitHub IP EACCES (WFP); DNS ENOTFOUND; proxy CONNECT evil.example.net HTTP 403 | PASS | [22](22-network-blocked.png) |
| 23 | probe net-allowed | allowlisted host works via proxy | CONNECT api.github.com 200, TLS GET /zen HTTP/1.1 200 OK | PASS | [23](23-network-allowed.png) |
| 24 | probe env with fake secrets set on host | only allowlisted vars | PATH, TEMP present; FAKE_SECRET_TOKEN, GITHUB_TOKEN, AWS_SECRET_ACCESS_KEY, USERPROFILE ABSENT | PASS (note: LOCALAPPDATA/TMP/SYSTEMROOT/WINDIR/proxy vars also visible by design) | [24](24-env-filtered.png) |
| 25 | probe mem, memory_mb 256 | stopped near 256 MB | allocation failed at 224 MB; "resource limit exceeded: memory (256MB)", exit 1 | PASS | [25](25-memory-limit.png) |
| 26 | probe timeout, timeout_s 6 | tree killed at 6 s | "wall-clock timeout (6s)" after 6.6 s; parent, child and grandchild PIDs gone | PASS | [26](26-timeout-kill.png) |
| 27 | probe children (parent exits, leaves child + grandchild) | Job Object kills the tree | child and grandchild PIDs gone; 0 leftover node.exe | PASS | [27](27-job-object-cleanup.png) |
| 28 | inspect audit.jsonl | ETW file events, network/limit decisions, no secrets | 1860 ETW file records; proxy allow/deny and limit-terminate records; 0 fake-secret matches | PASS | [28](28-audit-log.png) |
| 28b | grep audit for the denied paths | denied accesses not shown as allowed | every EPERM-denied open of C:\warden-outside\* is logged `"allowed":true`; WFP-blocked direct egress has no audit record | **FAIL** (audit accuracy) | text only |
| 29 | run with root grant / write on runtime dir / memory_mb -5 | refuse before target starts | refused with clear errors (exit 1/1/2); 0 probe processes | PASS | [29](29-fail-closed.png) |
| 29b | UNPRIVILEGED=1 while elevated, network.allow policy | no weakening | flag ignored; full WFP mode (direct blocked, unlisted 403, allowed 200) | PASS | [29b](29b-unprivileged-flag-while-elevated.png) |
| 30 | logman / netsh wfp / process / ACL checks after all runs | nothing left behind | 0 warden ETW sessions; 0 "warden" WFP filters; 0 leftover processes; SDDL of C:\, C:\warden-test, output, Program Files\nodejs identical to before; 0 AppContainer ACEs | PASS | [30](30-cleanup-after-runs.png) |

**Totals: 29 PASS, 2 FAIL, 1 SKIP** (32 rows)

Checklist summary: published install PASS, doctor PASS, Windows Go tests PASS, AppContainer PASS, WFP PASS, ETW audit PASS (accuracy issue in 28b), Job/process cleanup PASS, filesystem PASS, env PASS, network PASS, resource limits PASS, MCP proxy PASS (gateway not exercised), Docker SKIP.

## Failures and diagnosis

1. **10, unprivileged mode cannot run a normally installed node.**
   - The Windows backend adds a DACL grant on the command's runtime directory (`C:\Program Files\nodejs`). A non-admin user has no WRITE_DAC on that admin-owned directory, so `SetNamedSecurityInfoW` returns Access denied. This happens even though AppContainers can usually already read Program Files through ALL APPLICATION PACKAGES.
   - `doctor` reports `Status: READY` in this mode anyway. It also lists "Network policy: explicit hosts only", although this mode refuses network.allow.
   - On an interactive console the run then waits forever at the `Request it now? [y/N]` prompt.
   - Workaround used in step 11: copy node into a user-owned directory. WINDOWS-TESTING.md "Known limits" does not mention this. Likely fix: skip the runtime-base grant when it already grants RX to the AppContainer, or skip it in unprivileged mode.
2. **28b, the audit log marks denied file access as allowed.**
   - `internal/sandbox/windows/etw.go:449` hard-codes `Allowed: true` ("observed via ETW kernel trace") on every kernel FileIo event. ETW records the open attempt, not its result.
   - So reads and writes that the sandbox blocked with EPERM (C:\warden-outside\secret.txt, probe-write.txt) show up as `"allowed":true`.
   - Direct-egress connections dropped by WFP (1.1.1.1, 140.82.112.5) leave no audit record at all. Only proxy-level denials are logged.
   - Enforcement itself is correct. The audit trail is misleading for incident review.

## Other observations (not failures)

- Steps 15-18 ran with `XDG_STATE_HOME=C:\warden-test\state`, so `go test` added 3 `blocked-w4rd3n.invalid` records to the same audit log.
- ETW records use the time the logger wrote them, so dozens of events share one timestamp.
- Step 29 C: a relative `node` command was resolved through PATH and ran normally.
- `warden-starter/warden/warden-windows-amd64.exe` is left in the repo as an untracked build output. Nothing was committed.
