# Warden Windows backend — AI test runbook

Instructions for an AI coding agent (or a careful human) running on a **real
Windows machine**. Linux and Docker are already verified; macOS and Windows are
not. This runbook exists to produce the first real-Windows evidence for the
**AppContainer / WFP / ETW / Job Object** backend.

Read `WINDOWS-TESTING.md` for the manual checklist. This file is the
machine-followable version, and it carries one specific job: **confirm or refute
a WFP filter-install fix.**

---

## 0. What this is testing, and why it matters

Warden's Windows egress filters could never install. BFE (`FwpmFilterAdd0`)
rejected every filter with:

```
WFP FwpmFilterAdd(deny-all): code 0x6f7     # RPC_X_BAD_STUB_DATA
```

The cause was the Go mirror of the WFP structs disagreeing with the Windows SDK
(`FWPM_FILTER0` layout, action types, `FWP_DATA_TYPE` values, and every
identifier GUID). That has been fixed, and the offsets are now pinned by
`wfp_layout_test.go`.

**Your run either:**

- **confirms the fix** — filters install, the escape tests *execute*, and they
  pass; or
- **refutes it** — a specific WFP error is reported, which is still useful and
  must be sent back verbatim.

There is no third outcome. A **skipped** escape test is **not** a pass.

---

## 1. Requirements

| Item | Requirement |
|---|---|
| OS | Windows 10 or 11 **Pro/Enterprise**, or Windows Server 2019+ |
| Architecture | x64 (arm64 is not covered) |
| Shell | PowerShell **run as Administrator** |
| Go | 1.24 or newer |
| Git | Git for Windows |
| Node.js | Optional; only for the `warden doctor`/client steps |
| Machine | A **spare / test PC**. Do not run this on a daily driver with production data |

> Windows **Home** lacks parts of the AppContainer/WFP story. If that is all you
> have, still run it — but report the edition, because a skip there is an
> environment limit, not a pass.

Check the basics first:

```powershell
winver
go version
git --version
whoami /groups | Select-String -Pattern "S-1-16-12288"   # High Mandatory Level == elevated
```

If the elevation check prints nothing, reopen PowerShell as Administrator and
start again.

---

## 2. Get the code

The fix lives on branch `ctx/egress-integrity`, commit `e0a4c61` (plus the
earlier commits on that branch). **If the branch has not been pushed**, use the
tarball you were given instead of `git clone`.

```powershell
# Option A — branch is pushed
git clone https://github.com/Prof-bilal/Warden.git
Set-Location Warden
git checkout ctx/egress-integrity

# Option B — you received warden.tar.gz
New-Item -ItemType Directory -Force -Path C:\src | Out-Null
tar.exe -xzf .\warden.tar.gz -C C:\src
Set-Location C:\src\warden
```

Confirm you have the fix — this must print the correct action type `0x1001`:

```powershell
Select-String -Path .\internal\sandbox\windows\wfp_spec.go -Pattern "fwpmActionBlock"
```

Expect: `fwpmActionBlock = 0x0001 | fwpActionFlagTerminating // 0x1001`.
If it shows `0x00000000`, you have the **old** code — stop and get the right
revision.

---

## 3. Static gates (fast, must pass before anything else)

```powershell
go build ./...
go vet ./...
go test -count=1 -run "TestWFPSpec|TestBuildPlan|TestDerive" -v .\internal\sandbox\windows\...
go test -count=1 -run "TestWFPMFilterLayout|TestFWPMSublayerLayout|TestFWPMNestedLayout" -v .\internal\sandbox\windows\...
```

Expected: everything passes, including the three **layout** tests. A layout
failure means the structs do not match the SDK ABI — **stop and report it**; do
not "fix" the expected numbers.

---

## 4. The decisive check — can this host install a WFP filter?

This is the whole point. Run it and capture the exact output:

```powershell
go test -count=1 -v -run TestDiagUnsupportedReason .\internal\sandbox\windows\...
```

If that test is not present, create `internal\sandbox\windows\zz_diag_test.go`:

```go
//go:build windows

package windows

import "testing"

func TestDiagUnsupportedReason(t *testing.T) {
	if err := procNtCreateLowBoxToken.Find(); err != nil {
		t.Logf("NtCreateLowBoxToken.Find ERROR: %v", err)
	} else {
		t.Logf("NtCreateLowBoxToken.Find OK")
	}
	if err := wfpSupported(); err != nil {
		t.Logf("wfpSupported ERROR: %v", err)
	} else {
		t.Logf("wfpSupported OK")
	}
	t.Logf("Supported()=%v", Supported())
}
```

**Interpretation:**

| Output | Meaning |
|---|---|
| `wfpSupported OK` and `Supported()=true` | **Fix confirmed.** Continue to §5. |
| `code 0x6f7` | Still a struct/ABI mismatch. Report verbatim; send `go env GOOS GOARCH`, Windows build (`winver`), and full output. |
| `code 0x32` / `ERROR_NOT_SUPPORTED (50)` | This host refuses WFP engine sessions (some hardened/virtualised images do). Report it; the backend correctly fails closed. |
| `procedure ... not found in <dll>` | Missing export in `fwpuclnt.dll`/`iphlpapi.dll` on this image. Report the DLL name. |

Also confirm the two WFP services are running — a stopped BFE makes the engine
open fail:

```powershell
Get-Service BFE, mpssvc | Format-Table Name, Status, StartType -AutoSize
```

---

## 5. Execute the Windows enforcement tests

If §4 said `Supported()=true`, these must now **run**, not skip.

```powershell
go test -count=1 -v .\internal\sandbox\windows\... 2>&1 |
  Tee-Object -FilePath C:\warden-evidence\windows-tests.txt
```

Then confirm none of the boundary tests skipped:

```powershell
Select-String -Path C:\warden-evidence\windows-tests.txt -Pattern "^--- SKIP|not usable on this host"
```

**Expected:** no output. Any `SKIP` on `TestEscape*`, `TestAppContainer*`,
`TestEtw*`, `TestWFPDLL*` means the boundary was never exercised — record it as
**NOT A PASS**.

The tests that matter:

| Test | Boundary |
|---|---|
| `TestEscapeUngrantedReadDenied` | AppContainer filesystem denial |
| `TestGrantedPathReadable` | positive control (grants actually work) |
| `TestEscapeNetworkBlockedAudited` | WFP egress denial + ETW audit |
| `TestEscapeExceedsTimeoutTerminatesTree` | Job Object process-tree cleanup |
| `TestEtwSessionLifecycle` | ETW session create/stop |
| `TestWFPDLLProbeResilient` | fail-closed when WFP is unavailable |

---

## 6. End-to-end product run

```powershell
go build -o C:\warden-evidence\warden.exe .\cmd\warden
C:\warden-evidence\warden.exe version
C:\warden-evidence\warden.exe doctor    | Tee-Object C:\warden-evidence\doctor.txt
```

`doctor` must report the Windows backend available and fail-closed enabled —
**no unsandboxed fallback**.

Then a real sandboxed run using the disposable test server:

```powershell
New-Item -ItemType Directory -Force -Path C:\warden-test\output | Out-Null
@'
command: ["C:/Windows/System32/cmd.exe", "/c", "echo hello > C:/warden-test/output/ok.txt"]
filesystem:
  read:  ["C:/warden-test"]
  write: ["C:/warden-test/output"]
network:
  allow: ["example.com"]
env:
  allow: ["PATH", "TEMP"]
limits:
  memory_mb: 256
  timeout_s: 30
'@ | Set-Content C:\warden-test\policy.yaml

C:\warden-evidence\warden.exe run --backend windows `
  --policy C:\warden-test\policy.yaml -- C:\Windows\System32\cmd.exe /c "echo hello > C:\warden-test\output\ok.txt"

Test-Path C:\warden-test\output\ok.txt   # must be True
```

Verify the negative side too — a write **outside** the grant must fail:

```powershell
C:\warden-evidence\warden.exe run --backend windows `
  --policy C:\warden-test\policy.yaml -- C:\Windows\System32\cmd.exe /c "echo nope > C:\warden-escape.txt"
Test-Path C:\warden-escape.txt           # must be False
```

---

## 7. Egress matrix (host mode)

Host mode cannot exercise hostname/DNS/IPv6 cases — those report
**UNSUPPORTED** and are never PASS.

```powershell
cd .\warden-starter\warden          # if you cloned the repo
$env:CGO_ENABLED = "0"
go build -o C:\warden-evidence\egress-runner.exe .\testdata\egress\runner
C:\warden-evidence\egress-runner.exe -backend windows -mode host -build `
  -out C:\warden-evidence\egress-windows
```

Then report the counts exactly as printed. `-require-complete` is deliberately
**not** used here because host mode legitimately reports UNSUPPORTED; WARN,
UNSUPPORTED, BLOCKED and NOT_RUN are never converted to PASS.

---

## 8. Evidence to send back

Create `C:\warden-evidence` and collect:

```
windows-tests.txt     # §5, full -v output
diag.txt              # §4 output, and Get-Service BFE,mpssvc
doctor.txt            # §6
version.txt           # warden version + go version
egress-windows\       # §7 run.json, results.jsonl, summary.json, report.md
environment.txt       # winver, whoami /groups, Get-CimInstance Win32_OperatingSystem
screenshots\          # optional: the terminal output, for review
```

**Redact before sharing:** user names, machine names, install paths, any
`C:\Users\<name>\...` fragments, and audit logs that contain local paths.

Then fill in:

```text
Tester / AI agent:
Date:
Windows edition + build:
Architecture:
Elevated: YES / NO
Go version:

§3 static gates:            PASS / FAIL
§4 WFP engine install:      PASS (Supported=true) / FAIL (code ...)
§4 BFE/mpssvc:              Running / Stopped
§5 escape tests executed:   YES / NO (skipped: <names>)
§5 escape tests:            PASS / FAIL
§6 doctor:                  PASS / FAIL
§6 filesystem boundary:     PASS / FAIL     (write inside ok, outside blocked)
§7 egress host mode:        PASS=n FAIL=n WARN=n UNSUPPORTED=n BLOCKED=n NOT_RUN=n
   (attach summary.json)

Blocking errors, verbatim:
Notes / anything unexpected:
```

---

## 9. Rules for the agent running this

1. **Never report a WARN, UNSUPPORTED, BLOCKED, NOT_RUN or SKIP as PASS.**
2. **Never** disable Windows Firewall, BitLocker, Defender or Secure Boot to
   make a test pass. If something needs that, the result is INCONCLUSIVE.
3. Do not modify `wfp_layout_test.go` / `wfp_spec_test.go` expected values to
   make them green. They encode the Windows ABI; a failure is the finding.
4. Copy error codes **verbatim** (`0x6f7`, `0x32`, `FWP_E_*`). They are the
   actionable part.
5. This is someone's real machine: no destructive commands, no changes to
   firewall state, and keep everything inside `C:\warden-test` and
   `C:\warden-evidence`.

---

## 10. After you report

If §4 and §5 are green on real hardware, the Windows backend moves from
INCONCLUSIVE to a scoped **YES** in `EGRESS-INTEGRITY.md`, and the machine can be
registered as the repo's self-hosted CI runner so it runs on every push:

- Runner labels: `self-hosted, windows, x64, wfp`
- Repository variable: `WARDEN_WFP_SELF_HOSTED=true`
- The `windows-wfp-self-hosted` job in `.github/workflows/egress-integrity.yml`
  then runs the matrix with `-require-complete`.

If §4 still fails, send the verbatim error and stop — the next fix depends on
that code.
