# Contributing to Warden

Thank you for helping make MCP servers safe to run. This guide covers everything you need: repo layout, environment setup, what to work on, and how we review.

Warden is a **security tool with a fail-closed contract**: if a sandbox primitive is unavailable, Warden must refuse to runnever fall back to executing unsandboxed. Every contribution is reviewed against that contract.

## Ways to contribute

You don't need to write Go to help:

| | Contribution | Good first issue? |
|---|---|---|
| 📝 | **Example policies** for MCP servers you use (see [examples/](warden-starter/warden/examples/)) | ✅ Yes |
| 🧪 | **Platform testing**run the [proof harness](warden-starter/warden/testdata/proof/) on your hardware (especially macOS) and report results | ✅ Yes |
| 📚 | **Docs**fix stale claims, improve guides, add worked examples | ✅ Yes |
| 🐛 | **Bug reports**use the issue templates; compat problems have a dedicated [template](.github/ISSUE_TEMPLATE/compat_report.md) | ✅ Yes |
| 🧩 | **Compatibility matrix**test a new MCP server and add its fixture (see [docs/compatibility.md](warden-starter/warden/docs/compatibility.md)) | ✅ Yes |
| 🔧 | **Backends & core code**bubblewrap/Seatbelt/AppContainer/Docker backends, proxy, policy engine | After a discussion |

## Repository layout

```
.
├── README.md                  # this project's front page
├── ROADMAP.md                 # milestone history + current direction
├── ARCHITECTURE.md            # how Warden works, end to end
├── SECURITY.md                # vulnerability reporting
├── .github/
│   ├── actions/warden-action/ # GitHub Action to sandbox CI steps
│   ├── workflows/             # CI, release, action tests
│   └── ISSUE_TEMPLATE/        # bug / feature / compat report
└── warden-starter/warden/     # the Go module (the only Go module)
    ├── cmd/warden/            # CLI entrypoint
    ├── internal/
    │   ├── policy/            # YAML policy parse/validate/normalize
    │   ├── sandbox/           # linux / darwin / windows / docker backends
    │   ├── proxy/             # egress allowlist proxy + bridge
    │   ├── envfilter/         # env allowlist filtering
    │   ├── audit/             # JSONL audit log (+ strace importer)
    │   └── compat/            # 18-server compatibility fixtures
    ├── docs/                  # user docs (deployed to GitHub Pages)
    ├── examples/              # copy-paste policy examples
    ├── testdata/proof/        # reproducible proof harness
    └── testdata/compat/       # compat matrix fixtures
```

> **Note the nesting:** the Go module lives at `warden-starter/warden/`, not the repo root. There is exactly **one** `go.mod`, and CI enforces it (`.github/workflows/ci.yml` → `repo-hygiene` job).

## Development setup

Requirements: **Go ≥ 1.24** (see `warden-starter/warden/go.mod`), `git`, and on Linux `bubblewrap` + `strace` for the integration tests. Docker is optional (the docker-backend tests skip without a daemon).

```bash
git clone https://github.com/Prof-bilal/Warden.git
cd Warden/warden-starter/warden

go build ./...          # compiles everything
go vet ./...            # static analysis
go test ./...           # full suite (escape tests run where the backend exists)
```

Run the CLI you just built:

```bash
go build -o warden ./cmd/warden
./warden doctor                      # is this machine sandbox-ready?
./warden run --policy ../examples/policy.example.yaml -- echo hello
```

Before changing shared code, verify it cross-buildsbackends are platform-gated, so your OS only compiles some of them:

```bash
GOOS=darwin  GOARCH=arm64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```

### Testing your changes

```bash
go test ./...                                        # everything
go test ./internal/sandbox/... -v                    # backend behavior
go test ./internal/policy/ -run TestNormalize -v     # one test
./testdata/proof/run-proof.sh                        # end-to-end proof harness (writes evidence/)
```

The proof harness is the fastest way to check that sandbox semantics still hold end-to-end after a change. Its `evidence/` output is gitignoreddon't commit run artifacts.

## What to work on

1. **Pick from the issue tracker**issues labeled `good first issue` are scoped for newcomers.
2. **Platform verification** is always valuable: run the proof harness on macOS (native + Docker fallback) and Windows, and open an issue with your `evidence/` summary.
3. **Bigger changes** (backend work, schema changes, new subcommands): open an issue or discussion first so we can agree on the approach before you invest time. The [architecture doc](ARCHITECTURE.md) and [design notes](warden-starter/warden/docs/design.md) explain the constraints.

### The rules that keep Warden safe

These are non-negotiable in review:

- **Fail closed.** If an enforcement primitive (namespace, filter, token, audit session) can't be initialized, the run must refuse. A plain-process fallback is never acceptable.
- **Deny by default.** New features must not widen default access. Anything not granted is invisible or blocked.
- **Claims match code.** Docs and READMEs may only state behavior that exists and is tested. If you find a stale claim, fixing it is a real contribution.
- **No secrets in tests or examples.** Policies reference env var *names*, never values.

## Pull requests

1. Fork, then create a branch from `main`: `git checkout -b fix/env-leak`
2. Make your change with focused commits
3. Verify locally: `gofmt`, `go vet`, `go test ./...`, and cross-builds for the platforms above
4. Push and open a PR describing **what** changed and **why**; link the issue it closes

**PR checklist:**

- [ ] `go test ./...` passes locally (not just the package you touched)
- [ ] `gofmt -l .` is empty; `go vet ./...` is clean
- [ ] Cross-builds pass (`GOOS=darwin`, `GOOS=windows`)
- [ ] New behavior has tests
- [ ] Docs updated if behavior or user-facing surface changed
- [ ] One logical change per PReasier to review, easier to roll back

Review promise: maintainers aim to respond to new PRs within a few days. Security-relevant changes get the deepest reviewexpect questions about the failure modes of every code path.

## Reporting security issues

**Do not open a public issue for a vulnerability.** See [SECURITY.md](SECURITY.md) for private reportingespecially anything that looks like a sandbox escape or a fail-open path.

## Community

- [Discussions](https://github.com/Prof-bilal/Warden/discussions)questions, ideas, show-and-tell
- [Issue tracker](https://github.com/Prof-bilal/Warden/issues)bugs and scoped work

By participating, you agree to keep the collaboration respectful and on-topic.
