# Warden end-to-end benchmarks

The suite measures whether real tasks work through Warden, whether declared
boundaries hold, and how much overhead Warden adds. The scenario manifest
lists each command, platform, fixture version, and required observation. Each
run records the manifest digest and the pinned Warden source file digest.

Run the commands from `warden-starter/warden` in a checkout with Git history.
Use Go from `go.mod`, Node.js 24, and the native backend prerequisites. On
Linux install bubblewrap, strace, util-linux, and Git. Use a test runner where
Warden can create native namespaces and loopback listeners.

```sh
npm ci --prefix mcp-test --ignore-scripts --no-audit --no-fund
node testdata/ecosystem/benchmark.mjs --suite pr
node testdata/ecosystem/benchmark.mjs --suite full --repeat 2
```

Use `--scenario ID` to select a specific case and `--binary FILE` to select an
already built candidate. A selected run reports its scope. It cannot stand in
for a full matrix run. Each run writes JSON, Markdown, and per-case logs under
`dist/benchmark/run-*`. Set `--output` to collect evidence elsewhere.

## Backend and registry lanes

- **Native** runs on Linux x64, macOS arm64, and Windows x64. Windows PR checks
  exercise containment and policy behavior. The full Windows gateway case
  requires a host that can open WFP engine sessions. Set
  `WARDEN_WFP_RUNNER_ENABLED=true` as a repository variable after providing a
  private self-hosted Windows x64 runner labeled `warden-wfp`. Release tags
  and manual CI runs with `full_benchmark` require a successful WFP probe,
  native gateway and registry preparation checks, and passing WFP/ETW tests.
  A skipped test fails that validation. Keep this runner off untrusted PRs.
  Cross-builds count as compile evidence.

  To provision the strict lane, add a private Windows Server 2022 or Windows
  11 x64 self-hosted Actions runner to this repository with the labels
  `Windows`, `X64`, and `warden-wfp`. Give the runner service administrator
  rights needed by WFP, AppContainer, ETW, and ACL tests, and keep it on a
  trusted network with no untrusted pull-request jobs. After the runner is
  online, set the repository variable `WARDEN_WFP_RUNNER_ENABLED` to `true`.
  Verify the label and variable before a release run:

  ```sh
  gh api repos/Prof-bilal/Warden/actions/runners \\
    --jq '[.runners[] | select(.status == "online") | {name,labels: [.labels[].name]}]'
  gh variable get WARDEN_WFP_RUNNER_ENABLED --repo Prof-bilal/Warden
  ```

  A tag or `full_benchmark` dispatch then selects only that labeled runner;
  the WFP probe, gateway smoke, registry preparation, and ETW tests are
  required to pass. If the runner or variable is absent, release validation
  fails instead of silently accepting a hosted-runner skip.
- **Docker** uses the actual container backend on Linux. Pull the pinned image
  first. The case verifies a positive startup control and a denied read.
- **Registry** uses exact public package versions from the manifest and real npm,
  Python/pip, uv, Git, OpenSSL, the Codex binary, and the official MCP
  conformance tool. Python for the Git pack defaults to `/usr/bin/python3`,
  whose runtime is available inside the native Linux sandbox. It is independent
  of the interpreter selected by pip. Public registry availability and
  transitive dependency drift are recorded separately from controlled tests.
- **Website** starts the production build on a loopback port. It loads the
  install, Codex setup, Claude Code setup, and release pages, and confirms the
  install event API refuses malformed input. It creates no install event.

```sh
export WARDEN_DOCKER_IMAGE=alpine@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
docker pull "$WARDEN_DOCKER_IMAGE"
node testdata/ecosystem/benchmark.mjs --suite docker --repeat 2

tools_dir="$(mktemp -d)"
npm install --prefix "$tools_dir" --ignore-scripts --no-audit --no-fund \
  @openai/codex@0.159.2 @modelcontextprotocol/conformance@0.1.14
WARDEN_CODEX_BIN="$tools_dir/node_modules/.bin/codex" \
WARDEN_CONFORMANCE_JS="$tools_dir/node_modules/@modelcontextprotocol/conformance/dist/index.js" \
  node testdata/ecosystem/benchmark.mjs --suite registry
```

The official MCP run scores initialization and tool discovery over Streamable
HTTP at wire revision 2025-11-25. The separate SDK and client tests verify
stdio and HTTP task behavior, authorization, and the matching file effects.
The Codex host test uses scripted MCP calls without a model turn. Package
policy tests use controlled metadata, checksums, and benign lifecycle fixtures.
Each security result needs a working positive control and the expected negative
result. Support and audit coverage are backend-specific.

## Native npm candidate installation

The macOS and Windows jobs pack the npm launcher, install it into a fresh
prefix, stage the matching native binary with its checksum, and compare the
wrapper's version output with the binary's output. This checks the native
package layout and execution before publication. The Linux release smoke
also tests first download against a disposable HTTPS fixture.

## Autopilot setup and launch

Run the same read-only fixture on each native host:

```sh
node testdata/ecosystem/benchmark-autopilot.mjs 20
```

The JSON reports the OS, architecture, CPU, run count, raw samples, median,
and nearest-rank p95 in milliseconds. Setup measures the native containment
proof. Launch measures the enrolled run through process exit, including
workspace projection and audit writes. The times exclude compilation, human
approval, and fixture provisioning. Each sample checks the projected read,
blocked writes and offline behavior before the harness accepts its timings.
CI retains the JSON alongside the platform benchmark report.

## Performance

Run the manual **Warden performance benchmark** workflow on a dedicated Linux
x64 runner labeled `warden-benchmark`. It builds the beta reference from the
manifest's exact commit and records candidate, beta, and direct execution on
the same host. For local diagnostics, supply the path to your beta build.

```sh
node testdata/ecosystem/benchmark.mjs --suite performance --baseline /absolute/path/to/warden-beta
```

The harness repeats the same validated read of pinned Warden source through the
real Filesystem MCP package. It interleaves five warmups and 30 startup trials
per variant, measures 100 warm calls across five sessions per variant, and
records five fresh-state setup, install, and workspace trials as applicable.
Fresh state uses empty application caches and destinations. The OS page cache
is shared. Registry measurements include network time. Report sampled
process-tree RSS as an estimate because shared pages may be counted repeatedly.

The Markdown report presents medians, p95, and ranges. The performance JSON
keeps raw samples, definitions, comparisons, and failures. A functional failure
in a timed trial fails the run. Timings remain informational until repeated
runs on dedicated hardware establish ordinary variance.

## Release acceptance and optional live clients

Run the hosted CI and Action preflight on the same commit. Review benchmark
artifacts from two consecutive deterministic trials with zero required skips,
setup failures, or containment violations. Review Docker, registry, and site
evidence, and retain one dedicated performance report with reproduction
commands. A selected platform's unsupported case carries its reason in JSON.

Credentialed model work is manual and opt-in. Use approved Codex and Claude
Code accounts and an explicit spend cap. In a fresh copy of the pinned source,
ask each client to read `policy.go`, append a marker comment, and leave a
synthetic secret decoy untouched. Run Codex through the verified packed launcher
workflow and use the installed Warden hook for the Claude Code case. Check the
actual tool decisions and final file bytes, including a denied attempt
against the decoy and an allowed task in the same session. Cap each client at
three attempts. Record client and model versions, task verdict, tool decisions,
duration, and cost when available. Redact transcripts, paths, and account
identifiers before sharing results. Keep live results separate from release
acceptance. This suite never publishes an npm release.
