# Warden Windows Testing Guide

Use this checklist on a real Windows 10/11 machine. It verifies AppContainer
filesystem isolation, WFP network filtering, ETW audit events, Job Object
cleanup, and fail-closed startup.

Do not use WSL for the native result. WSL tests Linux behavior. Run the Windows
`.exe` directly from PowerShell or Git Bash.

## Requirements

- Windows 10/11 x64, preferably Pro or Enterprise.
- Administrator PowerShell. WFP and ETW tests may skip without elevation.
- Git for Windows with Git Bash.
- Go 1.24 or newer for source and integration tests.
- Node.js/npm for published installation and the MCP fixture.
- Docker Desktop is optional for the Docker fallback test.
- A disposable test directory with no personal or production data.

Check the machine before testing:

```powershell
winver
go version
node --version
npm --version
whoami
```

Record Windows edition/build, administrator status, architecture, and missing
prerequisites. A skipped WFP/AppContainer/ETW test is not a pass.

## 1. Published installation

Open the Warden landing page and copy its install command. The expected npm
command is:

```powershell
npm install -g warden-sandbox-cli
warden --version
warden doctor
```

Expected doctor result: Windows backend available, fail-closed enabled, and no
unsandboxed fallback.

## 2. Build and test the checkout

From PowerShell at the repository root:

```powershell
Set-Location warden-starter/warden
go build -o warden-windows-amd64.exe .\cmd\warden
.\warden-windows-amd64.exe --version
.\warden-windows-amd64.exe doctor
```

Run as Administrator:

```powershell
go test -v -count=1 .\internal\sandbox\windows\...
go test -count=1 ./...
go test -v -count=1 -run 'TestEtw|TestEscape|TestAppContainer|TestWFP' .\internal\sandbox\windows\...
```

The targeted output must show discovered tests. A skip must include a concrete
reason such as unavailable WFP engine, missing elevation, or unsupported
AppContainer.

## 3. Native sandbox run

Create a policy using real Windows paths, but only a disposable test directory:

```yaml
command: ["C:/Program Files/nodejs/node.exe", "C:/warden-test/server.js"]
filesystem:
  read: ["C:/warden-test"]
  write: ["C:/warden-test/output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["PATH", "TEMP"]
limits:
  memory_mb: 256
  timeout_s: 60
```

Run it with the native backend:

```powershell
.\warden-windows-amd64.exe run `
  --backend windows `
  --policy C:\warden-test\policy.yaml `
  -- node C:\warden-test\server.js
```

Verify allowed reads/writes succeed and reads/writes outside the grants fail.

## 4. Security boundary checks

Using the harmless MCP fixture or disposable test server, verify:

- Read inside the grant: succeeds.
- Write inside the grant: succeeds.
- Read/write outside the grant: blocked.
- Unlisted environment variables: absent.
- Allowed hostname: succeeds through Warden’s proxy.
- Unlisted hostname and direct egress: blocked by WFP/proxy.
- Child and grandchild processes terminate with the run.
- Memory and timeout limits terminate the test process.
- An unavailable Windows security primitive refuses to start the server.

Inspect and redact the Warden audit output after each run.

## 5. MCP proxy and gateway

From Git Bash, using the Windows executable:

```bash
export XDG_STATE_HOME="$PWD/windows-evidence/state"
bash test-mcp-proxy.sh "$PWD/warden-starter/warden/warden-windows-amd64.exe"
```

Verify allowed forwarding, denied JSON-RPC errors, parse error `-32700`, and
audited decisions. If Git Bash cannot bind the local port, rerun it as
Administrator and record the exact error.

For gateway testing, wrap the disposable local MCP server and repeat one
allowed and one denied tool call through the generated client configuration.

## 6. Optional Docker comparison

```powershell
docker info
docker pull alpine:3.20
.\warden-windows-amd64.exe run --backend docker --policy C:\warden-test\policy.yaml -- node C:\warden-test\server.js
```

Report native Windows and Docker as separate backend results. Docker passing
does not prove AppContainer/WFP/ETW passed.

## 7. Evidence report

Return a ZIP containing redacted `doctor.txt`, `version.txt`,
`go-test-windows.txt`, `go-test-all.txt`, `native-run.txt`, `audit.jsonl`,
`proxy-test.txt`, optional `docker-test.txt`, and `environment.txt`.

Use this summary:

```text
Tester:
Date:
Windows edition/build:
Architecture:
Administrator: YES / NO
Go version:
Node/npm version:
Warden package version:
Warden binary version:
Backend reported by doctor:

Published installation: PASS / FAIL
Doctor: PASS / FAIL
Windows Go tests: PASS / FAIL / SKIP
AppContainer: PASS / FAIL / SKIP
WFP network enforcement: PASS / FAIL / SKIP
ETW audit: PASS / FAIL / SKIP
Job/process cleanup: PASS / FAIL / SKIP
Filesystem boundary: PASS / FAIL
Environment filtering: PASS / FAIL
Network boundary: PASS / FAIL
Resource limits: PASS / FAIL
MCP proxy/gateway: PASS / FAIL / SKIP
Docker fallback: PASS / FAIL / SKIP

Failures, skips, and limitations:
Evidence ZIP path:
```

## Important interpretation

Windows hosted GitHub runners may compile the backend while refusing real WFP
engine sessions. A hosted-runner green build is not equivalent to native
Windows enforcement. Native verification requires AppContainer, WFP, ETW, and
escape tests to execute on the friend’s real machine.
