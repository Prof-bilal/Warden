# Remaining Work

**Status:** M0–M8 shipped. Release pipeline green (v0.1.3–v0.1.5 published),
`warden version`/`--version` flag live. This document tracks what the **first
real CI run** (root `.github/workflows/ci.yml` landed in
[`885ab2f`](https://github.com/Prof-bilal/Warden/actions/runs/34028064382)
after v0.1.5) exposed, plus loose ends from the release work.

**Evidence:** CI run
[`34028064382`](https://github.com/Prof-bilal/Warden/actions/runs/34028064382)
— `linux` ✅, `cross-build` ✅, `docs` ✅, `windows` ❌ (see below).

---

## P0 — Fix the production bug exposed by the Windows job (was masked until now)

A real Windows runner is the only place the TinyGo/COM proc bindings get
exercised, and it caught a **wrong DLL binding that would panic `warden run`
instead of failing closed** on a real Windows machine.

**What failed:** `TestEtwSessionLifecycle` panicked with

```
panic: Failed to find OpenTraceW procedure in kernel32.dll:
        The specified procedure could not be found.
```

because `syscalls.go` bound `OpenTraceW`, `ProcessTrace`, `CloseTrace` to
`kernel32.dll`, but those live in **advapi32.dll** on modern Windows.

**Impact:** before this fix, `warden run --backend windows` with ETW enabled
would panic at session startup on real Windows (advapi32 exports the symbol,
kernel32 does not), rather than failing closed cleanly. The ETW session's
`ProcessTrace` goroutine would have died too.

**Where:** `internal/sandbox/windows/syscalls.go:42-44` (proc bindings).
Verify/confirm on device and also in docs:
- this may also need a human sanity check that `ProcessTrace`/`CloseTrace`
  are advapi32 on the actual target, since the plan's original "kernel32" was
  wrong and I haven't independently confirmed the full trio via the device docs
  this conversation referenced (MS evntcons.h).

**Acceptance:** the Windows CI job goes green on this test, including the
assertion step that confirms the ETW/escape/AppContainer/WFP tests actually
ran (not skipped).

**Acceptance criteria (language-sensitive):**
- `warden version` and `warden --version` both print the stamped version.
- Release binary checksum matches the published SHA256SUMS.
- `TestEtwSessionLifecycle` passes on the Windows runner, and the assertion
  step in the Windows job confirms ETW/escape/AppContainer/WFP tests ran.

---

## P1 — Make the Windows CI job robust (so it fails loudly on skip, not silently)

The Windows job assertion step I added (`windows-latest`) is a good start, but
it's currently gated behind `TestEtw|TestEscape|TestAppContainer|TestWFP` in a
pwsh grep. Once the P0 DLL fix lands, this job should go green on the escape
tests — and stay green.

**TBD:**
- Confirm the assertion regex still matches the real test names after any rename.
- If any escape test is skipped on the runner (elevated privilege issue), the
  assertion step must fail so the skip is caught rather than silently ignored.

---

## P2 — Fix cross-platform test portability debt (Latent — only surfaced because CI ran)

Several tests were written assuming POSIX paths and a Linux $PATH, so they
failed on the first Windows run. These aren't backend bugs — the backends
(including Windows-specific ones) passed. These are **test fixtures and
assertions** that need to be OS-aware.

**What failed by package:**

- **`internal/compat`** — `TestFixturePoliciesGrantWhatManifestClaims` (18
  subtests), `TestResolveExecutable`. The manifest fixture policies and test
  code use `/usr/bin/node`, `/bin/true`, etc. as absolute commands, which
  aren't absolute on Windows and don't resolve via PATH there.
- **`internal/policy`** — `grants_test.go`, `starter_test.go` use `/srv/ro`,
  `/srv/data`, `/work/...` paths that aren't absolute on Windows.
- **`internal/sandbox/darwin`** — `profile_test.go` uses `/usr/bin/true` and
  `/opt/custom/bin/server` as fixtures. These are macOS-only tests anyway, but
  they still fail when the package is compiled for `GOOS=windows`.
- **`internal/sandbox/docker`** — `docker_test.go` uses `/bin/true`.

**Likely root cause pattern:** the failing tests construct fixtures with
bare POSIX paths (e.g. `"/usr/bin/node"`, `"/srv/data/file.txt"`,
`"/work/input.txt"`) and either (a) feed them into `filepath.IsAbs`-aware
assertions that reject them on Windows, or (b) expect `ResolveExecutable`
(PATH lookup) to succeed on names like `/usr/bin/node` which Windows treats
as a single literal token.

**Acceptance:** the Windows job goes green on these packages' tests — without
breaking the same tests on Linux/macOS. This almost certainly means gating the
problematic test cases by runtime OS (e.g. `if runtime.GOOS != "windows"` skip)
or switching them to tempdir-relative paths / `exec.LookPath` probes / platform
helper binaries.

**Recommended scope:**
- Be conservative: don't rewrite the compat fixtures (they're intentionally
  POSIX since that's the server world). Skip/filter just the Windows-failing
  assertions so the matrix-validity tests still run on Windows.
- For policy grants tests, switch the POSIX fixture paths to tempdir-relative
  paths (these tests are pure-logic anyway).
- For `internal/sandbox/darwin` tests: they're macOS-only by nature; verify
  whether they should be `GOOS`-filtered rather than failing on every non-macOS
  build.

**Acceptance criteria (language-sensitive):**
- The Windows CI job passes on `internal/compat`, `internal/policy`,
  `internal/sandbox/darwin`, `internal/sandbox/docker`, and
  `internal/sandbox/windows` (no failures, no silent skips of the escape tests).
- The Windows job still fails loudly if any escape/ETW test is skipped.
- Linux and macOS (cross-build) jobs continue to pass.

---

## P3 — Loose ends from release work (minor)

These are things left over from moving the release pipeline to root and
shipping v0.1.3–v0.1.5.

**3.1 — Subproject `ci.yml` still lives at `warden-starter/warden/.github/`**

GitHub **only reads root `.github/workflows/`**, so the subproject's `ci.yml`
(including the `windows-latest` job) never actually executed until I moved it
to root. That `ci.yml` file still exists in the subproject. Decide whether to:
- delete it (the root one is authoritative now), or
- keep it as a historical snapshot / migration artifact.

**3.2 — npm publish isn't rerun-safe (cosmetic, but real)**

The v0.1.4 rerun tripped `403 "cannot publish over the previously published
versions: 0.1.4"` because npm versions are immutable. The first pass succeeded,
so nothing was lost — but if a release job job fails late and is rerun, the
publish step fails even though the package is already published.

**Acceptance (optional):** publish step skips if the version already exists on
the registry, so failed-job reruns go green without burning a new version.

**3.3 — `build/npm-wrapper/package.json` still says `0.1.3`**

The release job versions it from the tag at publish time (`npm version
"${GITHUB_REF_NAME#v}" --no-git-tag-version`), so the registry gets the right
value — but the repo's copy stays stale at `0.1.3`. Cosmetic, but it's the
source of truth for the wrapper's docs.

**Acceptance (optional):** bump it to the latest shipped version, or leave it as
the dev-stamp baseline with a comment explaining the release job overrides it.

**3.4 — v0.1.4's release run shows ❌ on the rerun (cosmetic)**

The v0.1.4 run's `publish-npm` job is red because the rerun attempted a
duplicate publish. Nothing was lost, but the run history is noisy. Can't edit
GitHub run status.

**3.5 — Homebrew tap not actually published (manual step)**

`build/brew.sh` generates a formula, but there's no evidence of an actual
Homebrew tap repo being set up for distribution. If tap distribution is wanted,
that's a separate manual step (create tap repo, commit the generated formula,
update the landing page if it references a tap URL).

**3.6 — Verify OpenTraceW/ProcessTrace/CloseTrace DLL on the actual target**

The P0 fix is based on the documented fact that those are advapi32 exports.
Before treating it as fully verified on the Windows target, I'd ideally confirm
via the device's own SDK / docs this conversation referenced, since the prior
"kernel32" assumption was wrong and I haven't independently re-confirmed the
full trio from the authoritative source (MS evntcons.h / the SDK headers) on
this device. Treat P0 as "likely correct, needs one external confirmation pass"
if production-Windows behavior matters here.

---

## Done / verified (for the record)

- M0–M8 roadmap checklist complete (ROADMAP.md).
- Release pipeline green and automated: root `.github/workflows/release.yml`
  + root `.github/workflows/ci.yml` both registered and active on GitHub.
- npm: `warden-sandbox-cli@0.1.3`, `0.1.4`, `0.1.5` live; `0.1.5` includes
  the `--version` flag.
- GitHub Releases: `v0.1.3`, `v0.1.4`, `v0.1.5` published with all 5 binaries
  + SHA256SUMS.
- GitHub Pages: `https://prof-bilal.github.io/Warden/` serving (mkdocs on
  release trigger).
- CLI: `warden version` / `warden --version` / `warden -version` / `warden -v`
  all print the stamped build version and exit 0; unstamped builds report `dev`.
- Docs site renders `docs/cli.md` (including the new `warden version` section)
  directly, so README + landing + docs stay in sync.
