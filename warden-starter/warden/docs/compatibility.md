# MCP Server Compatibility

Will Warden work with *your* MCP server? This page is the public,
tested answer. Every row below has a permanent regression fixture under
`testdata/compat/`
— the exact `policy.yaml` the server needs — enforced by
`go test ./internal/compat/`, so a future Warden change cannot silently
break a server that used to work. Each pinned policy is reproduced in
[full below](#pinned-policies), so you can copy it without leaving this site.

> **How to read the verdicts:** ✅ **pass** works with the linked policy as
> written. ⚠️ **conditional** works once you fill in the deployment-specific
> grants noted. ❌ **fail** cannot be sandboxed today; the reason column says
> whether that is a Warden bug, a policy-schema gap, or inherent to the
> server's design.

## Matrix (18 servers)

| Server | Upstream | Verdict | Policy |
|---|---|---|---|
| Filesystem | `@modelcontextprotocol/server-filesystem` | ✅ pass | [policy](#filesystem) |
| GitHub | `@modelcontextprotocol/server-github` | ✅ pass | [policy](#github) |
| Slack | `@modelcontextprotocol/server-slack` | ✅ pass | [policy](#slack) |
| PostgreSQL | `@modelcontextprotocol/server-postgres` | ✅ pass | [policy](#postgresql) |
| SQLite | `@modelcontextprotocol/server-sqlite` | ✅ pass | [policy](#sqlite) |
| Brave Search | `@modelcontextprotocol/server-brave-search` | ✅ pass | [policy](#brave-search) |
| Google Drive | `@modelcontextprotocol/server-gdrive` | ✅ pass | [policy](#google-drive) |
| Git | `@modelcontextprotocol/server-git` | ✅ pass | [policy](#git) |
| Memory | `@modelcontextprotocol/server-memory` | ✅ pass | [policy](#memory) |
| Time | `@modelcontextprotocol/server-time` | ✅ pass | [policy](#time) |
| Sequential Thinking | `@modelcontextprotocol/server-sequential-thinking` | ✅ pass | [policy](#sequential-thinking) |
| Notion (community) | `notion-mcp` | ✅ pass | [policy](#notion-community) |
| Linear (community) | `linear-mcp` | ✅ pass | [policy](#linear-community) |
| Tavily search (community) | `tavily-mcp` | ✅ pass | [policy](#tavily-search-community) |
| Fetch | `@modelcontextprotocol/server-fetch` | ⚠️ conditional | [policy](#fetch) |
| Kubernetes (community) | `kubernetes-mcp` | ⚠️ conditional | [policy](#kubernetes-community) |
| Docker (community) | `docker-mcp` | ❌ fail (inherent) | [policy](#docker-community) |
| Playwright (community) | `playwright-mcp` | ❌ fail (schema gap) | [policy](#playwright-community) |

**Totals: 14 pass · 2 conditional · 2 fail.**

## What each server needs

- **Filesystem** — grant exactly the directories the server may serve
  (`filesystem.read`); no network, no env.
- **GitHub** — `network.allow: ["api.github.com"]` covers REST + GraphQL;
  pass `GITHUB_TOKEN` through `env.allow`, plus a cache/output dir.
- **Slack** — `slack.com` + `api.slack.com`, `SLACK_BOT_TOKEN` /
  `SLACK_TEAM_ID`, and a write-only cache dir (a write grant subsumes
  reads; don't list the same dir twice).
- **PostgreSQL** — the DB host(s) in `network.allow`, client credential
  files as read grants, `PG*` connection names in `env.allow`.
- **SQLite** — `write` on the database *directory* (WAL and journal
  sidecars live next to the db file), no network.
- **Brave Search / Notion / Linear / Tavily** — single-host API egress plus
  the provider's API-key env name.
- **Google Drive** — `www.googleapis.com` + `oauth2.googleapis.com` +
  `drive.google.com`, an OAuth client file as a read grant, and the Google
  credential env names.
- **Git** — `write` on the repo checkout it operates on; no network needed
  for local operations.
- **Memory** — `write` on the directory holding the JSON store; no network.
- **Time / Sequential Thinking** — empty grants: pure compute, the
  deny-by-default sandbox fits with zero configuration.

## Failures, classified

Every non-pass row gets a class, per the M8 triage rule:

### Fixed during triage (Warden bugs — highest impact first)

1. **Same path in `read` and `write` rejected (Slack).** The shipped Slack
   example listed `./data/cache` in both lists; it passed validation but
   every backend failed it as ambiguous. Since a write grant already
   subsumes reads underneath it, `policy.Load` now coalesces the overlap
   to a single write grant (`Normalize`), and the example is canonical
   write-only. Regression test:
   `TestOverlapNormalizeIsWriteWins`.
2. **Bare `npx`/`uvx`/`node` commands rejected.** Most public servers
   launch via a PATH launcher, but `warden run` demanded an absolute path
   before the backend could bind-mount anything. `warden run` and
   `warden gateway run` now resolve bare names via `LookPath` (the same
   behavior `gateway init` already had) and still fail closed off-PATH.
   Regression test: `TestResolveExecutable`.

### Schema gaps (access the schema cannot express yet)

- **No wildcard hosts.** A browser-automation server (Playwright) visits
  arbitrary domains; `network.allow` only accepts bare hostnames, so there
  is no honest policy for it. `TestDocumentedSchemaGaps` locks this in:
  adding wildcard support must update the matrix, fixtures, and this page
  together.
- **No unix-socket grants.** The Docker daemon socket and browser IPC
  sockets have no scoped-down representation in the schema.

### Inherently incompatible (no Warden change would help)

- **Docker-mcp: needs `/var/run/docker.sock`.** Granting the socket hands
  over full host container control and voids the sandbox; there is no
  scoped grant preserving both function and isolation.
- **Fetch: arbitrary user-supplied URLs by design.** Sandboxable only with
  an explicit per-deployment host list — a fully general fetch server
  cannot be allowlisted in advance.
- **Kubernetes: cluster-specific.** Works fine once the cluster API host
  and kubeconfig grant are set, but those are per-deployment by nature.

## Pinned policies

The exact regression fixtures under `testdata/compat/`, reproduced verbatim.
Copy the one closest to your server and point `command` at your install.

### Filesystem

Upstream: `@modelcontextprotocol/server-filesystem` · Verdict: **✅ pass** · Fixture: `testdata/compat/filesystem/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-filesystem (compat: pass)
# Relative paths resolve against THIS directory (testdata/compat/filesystem).
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./data"
  write: []
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 128
  timeout_s: 60
```

### GitHub

Upstream: `@modelcontextprotocol/server-github` · Verdict: **✅ pass** · Fixture: `testdata/compat/github/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-github (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./data/cache"
  write:
    - "./data/output"
network:
  allow:
    - "api.github.com"
env:
  allow:
    - "GITHUB_TOKEN"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### Slack

Upstream: `@modelcontextprotocol/server-slack` · Verdict: **✅ pass** · Fixture: `testdata/compat/slack/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-slack (compat: pass)
# Canonical write-only cache dir: a write grant subsumes reads underneath
# it, so no duplicate read grant (see policy.Normalize).
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "slack.com"
    - "api.slack.com"
env:
  allow:
    - "SLACK_BOT_TOKEN"
    - "SLACK_TEAM_ID"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### PostgreSQL

Upstream: `@modelcontextprotocol/server-postgres` · Verdict: **✅ pass** · Fixture: `testdata/compat/postgres/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-postgres (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./config/.pgpass"
    - "./config/.postgresql"
  write: []
network:
  allow:
    - "localhost"
    - "db.internal.example.com"
env:
  allow:
    - "PGHOST"
    - "PGPORT"
    - "PGUSER"
    - "PGPASSWORD"
    - "PGSSLMODE"
    - "HOME"
limits:
  memory_mb: 512
  timeout_s: 600
```

### SQLite

Upstream: `@modelcontextprotocol/server-sqlite` · Verdict: **✅ pass** · Fixture: `testdata/compat/sqlite/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-sqlite (compat: pass)
# SQLite needs WRITE on the db directory (WAL + journal sidecars), not just
# the db file. The write grant on ./data covers ./data/db.sqlite3.
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data"
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 128
  timeout_s: 60
```

### Brave Search

Upstream: `@modelcontextprotocol/server-brave-search` · Verdict: **✅ pass** · Fixture: `testdata/compat/brave-search/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-brave-search (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "api.search.brave.com"
env:
  allow:
    - "BRAVE_API_KEY"
    - "HOME"
limits:
  memory_mb: 128
  timeout_s: 60
```

### Google Drive

Upstream: `@modelcontextprotocol/server-gdrive` · Verdict: **✅ pass** · Fixture: `testdata/compat/gdrive/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-gdrive (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./config/credentials.json"
  write:
    - "./data/cache"
network:
  allow:
    - "www.googleapis.com"
    - "oauth2.googleapis.com"
    - "drive.google.com"
env:
  allow:
    - "GOOGLE_CLIENT_ID"
    - "GOOGLE_CLIENT_SECRET"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### Git

Upstream: `@modelcontextprotocol/server-git` · Verdict: **✅ pass** · Fixture: `testdata/compat/git/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-git (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/repo"
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 256
  timeout_s: 300
```

### Memory

Upstream: `@modelcontextprotocol/server-memory` · Verdict: **✅ pass** · Fixture: `testdata/compat/memory/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-memory (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data"
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 128
  timeout_s: 60
```

### Time

Upstream: `@modelcontextprotocol/server-time` · Verdict: **✅ pass** · Fixture: `testdata/compat/time/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-time (compat: pass)
# No filesystem or network needs at all: empty grants, deny-by-default.
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write: []
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 64
  timeout_s: 60
```

### Sequential Thinking

Upstream: `@modelcontextprotocol/server-sequential-thinking` · Verdict: **✅ pass** · Fixture: `testdata/compat/sequential-thinking/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-sequential-thinking (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write: []
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 128
  timeout_s: 120
```

### Notion (community)

Upstream: `notion-mcp` · Verdict: **✅ pass** · Fixture: `testdata/compat/notion/policy.yaml`

```yaml
# Fixture policy — community notion-mcp (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "api.notion.com"
env:
  allow:
    - "NOTION_API_KEY"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### Linear (community)

Upstream: `linear-mcp` · Verdict: **✅ pass** · Fixture: `testdata/compat/linear/policy.yaml`

```yaml
# Fixture policy — community linear-mcp (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "api.linear.app"
env:
  allow:
    - "LINEAR_API_KEY"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### Tavily search (community)

Upstream: `tavily-mcp` · Verdict: **✅ pass** · Fixture: `testdata/compat/tavily/policy.yaml`

```yaml
# Fixture policy — community tavily-mcp web search (compat: pass)
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "api.tavily.com"
env:
  allow:
    - "TAVILY_API_KEY"
    - "HOME"
limits:
  memory_mb: 128
  timeout_s: 60
```

### Fetch

Upstream: `@modelcontextprotocol/server-fetch` · Verdict: **⚠️ conditional** · Fixture: `testdata/compat/fetch/policy.yaml`

```yaml
# Fixture policy — @modelcontextprotocol/server-fetch (compat: CONDITIONAL)
# Fetch retrieves arbitrary user-supplied URLs by design, so no static
# allowlist is complete. This fixture pins one host (example.com) to prove
# the mechanism; a deployment must extend network.allow per use-case.
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/cache"
network:
  allow:
    - "example.com"
env:
  allow: []
limits:
  memory_mb: 256
  timeout_s: 120
```

### Kubernetes (community)

Upstream: `kubernetes-mcp` · Verdict: **⚠️ conditional** · Fixture: `testdata/compat/kubernetes/policy.yaml`

```yaml
# Fixture policy — community kubernetes-mcp (compat: CONDITIONAL)
# Works once the two deployment-specific grants are set: the cluster API
# host and a read grant on the kubeconfig file.
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./config/kubeconfig"
  write: []
network:
  allow:
    - "k8s.internal.example.com"
env:
  allow:
    - "KUBECONFIG"
    - "HOME"
limits:
  memory_mb: 256
  timeout_s: 300
```

### Docker (community)

Upstream: `docker-mcp` · Verdict: **❌ fail (inherent)** · Fixture: `testdata/compat/docker/policy.yaml`

```yaml
# Fixture policy — community docker-mcp (compat: FAIL, inherent)
# The server needs the Docker daemon socket (/var/run/docker.sock), i.e.
# full host container control. That grant is DELIBERATELY ABSENT here:
# granting the socket would void the sandbox. This fixture pins the rest of
# the policy so the regression test locks in the fail verdict and the
# documented reason (see docs/compatibility.md).
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./data"
  write: []
network:
  allow: []
env:
  allow: []
limits:
  memory_mb: 256
  timeout_s: 300
```

### Playwright (community)

Upstream: `playwright-mcp` · Verdict: **❌ fail (schema gap)** · Fixture: `testdata/compat/playwright/policy.yaml`

```yaml
# Fixture policy — community playwright-mcp (compat: FAIL, schema-gap)
# A browser visits arbitrary domains (needs wildcard hosts, which the schema
# rejects) and needs browser IPC / nested-sandbox syscalls Warden does not
# mediate. This fixture pins a single host to prove the mechanism while the
# matrix records the fail verdict and the gap.
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: []
  write:
    - "./data/profile"
network:
  allow:
    - "example.com"
env:
  allow:
    - "PLAYWRIGHT_BROWSERS_PATH"
    - "HOME"
limits:
  memory_mb: 512
  timeout_s: 300
```


## Manifest

The machine-readable source of truth (`testdata/compat/matrix.yaml`) behind the table above — verdict, failure class, and grant probes per server, enforced by `go test ./internal/compat/`. Reproduced verbatim.

```yaml
# Warden MCP compatibility matrix (M8) — machine-readable manifest.
#
# This file is the single source of truth for the matrix published in
# docs/compatibility.md. Each entry MUST have a matching fixture directory
# testdata/compat/<name>/policy.yaml, enforced by internal/compat/compat_test.go.
#
# verdict:
#   pass        — works under the fixture policy on a host with a working backend
#   conditional — works only with per-deployment grants (documented in notes)
#   fail        — cannot be sandboxed without a schema or design change
# failure_class (for verdict != pass, plus fixed bugs):
#   warden-bug  — Warden defect (fixed where fix_version is set)
#   schema-gap  — access pattern the policy schema cannot express yet
#   inherent    — server is incompatible with sandboxing by design
verdicts:
  - name: filesystem
    upstream: "@modelcontextprotocol/server-filesystem"
    verdict: pass
    failure_class: none
    policy: filesystem/policy.yaml
    notes: "Local file I/O only. Grant exactly the dirs the server may serve."
    probe_allow_files: ["data"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: github
    upstream: "@modelcontextprotocol/server-github"
    verdict: pass
    failure_class: none
    policy: github/policy.yaml
    notes: "REST + GraphQL share api.github.com. Needs GITHUB_TOKEN."
    probe_allow_files: ["data/cache", "data/output"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["api.github.com"]
    probe_allow_env: ["GITHUB_TOKEN"]

  - name: slack
    upstream: "@modelcontextprotocol/server-slack"
    verdict: pass
    failure_class: warden-bug
    fix_note: "Shipped example listed ./data/cache in both read and write; every backend rejected it as ambiguous. Fixed by policy.Normalize (write-wins coalescing) + canonical write-only example."
    policy: slack/policy.yaml
    notes: "Needs slack.com + api.slack.com, SLACK_BOT_TOKEN, SLACK_TEAM_ID."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["slack.com", "api.slack.com"]
    probe_allow_env: ["SLACK_BOT_TOKEN", "SLACK_TEAM_ID"]

  - name: postgres
    upstream: "@modelcontextprotocol/server-postgres"
    verdict: pass
    failure_class: none
    policy: postgres/policy.yaml
    notes: "DB host per deployment; credentials via PG* env, never in policy."
    probe_allow_files: ["config/.pgpass"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["localhost", "db.internal.example.com"]
    probe_allow_env: ["PGHOST", "PGPASSWORD"]

  - name: sqlite
    upstream: "@modelcontextprotocol/server-sqlite"
    verdict: pass
    failure_class: none
    policy: sqlite/policy.yaml
    notes: "SQLite needs WRITE on the db file's directory (WAL + journal sidecars)."
    probe_allow_files: ["data/db.sqlite3", "data"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: brave-search
    upstream: "@modelcontextprotocol/server-brave-search"
    verdict: pass
    failure_class: none
    policy: brave-search/policy.yaml
    notes: "Single-host API egress. Needs BRAVE_API_KEY."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["api.search.brave.com"]
    probe_allow_env: ["BRAVE_API_KEY"]

  - name: gdrive
    upstream: "@modelcontextprotocol/server-gdrive"
    verdict: pass
    failure_class: none
    policy: gdrive/policy.yaml
    notes: "Google APIs live on www.googleapis.com + oauth2.googleapis.com + drive.google.com. OAuth client file is a read grant."
    probe_allow_files: ["config/credentials.json", "data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["www.googleapis.com", "oauth2.googleapis.com", "drive.google.com"]
    probe_allow_env: ["GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"]

  - name: git
    upstream: "@modelcontextprotocol/server-git"
    verdict: pass
    failure_class: none
    policy: git/policy.yaml
    notes: "Needs write on the repo checkout it operates on."
    probe_allow_files: ["data/repo"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: memory
    upstream: "@modelcontextprotocol/server-memory"
    verdict: pass
    failure_class: none
    policy: memory/policy.yaml
    notes: "Knowledge-graph server; local JSON store only, no network."
    probe_allow_files: ["data/memory.json"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: time
    upstream: "@modelcontextprotocol/server-time"
    verdict: pass
    failure_class: none
    policy: time/policy.yaml
    notes: "No filesystem or network needs at all — empty grants."
    probe_allow_files: []
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: sequential-thinking
    upstream: "@modelcontextprotocol/server-sequential-thinking"
    verdict: pass
    failure_class: none
    policy: sequential-thinking/policy.yaml
    notes: "Pure reasoning server; empty grants like time."
    probe_allow_files: []
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: notion
    upstream: "community: notion-mcp (@makenotion / sison1992)"
    verdict: pass
    failure_class: none
    policy: notion/policy.yaml
    notes: "Single-host API egress. Needs NOTION_API_KEY."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["api.notion.com"]
    probe_allow_env: ["NOTION_API_KEY"]

  - name: linear
    upstream: "community: linear-mcp"
    verdict: pass
    failure_class: none
    policy: linear/policy.yaml
    notes: "Single-host API egress. Needs LINEAR_API_KEY."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["api.linear.app"]
    probe_allow_env: ["LINEAR_API_KEY"]

  - name: tavily
    upstream: "community: tavily-mcp (web search)"
    verdict: pass
    failure_class: none
    policy: tavily/policy.yaml
    notes: "Single-host API egress. Needs TAVILY_API_KEY."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["api.tavily.com"]
    probe_allow_env: ["TAVILY_API_KEY"]

  - name: fetch
    upstream: "@modelcontextprotocol/server-fetch"
    verdict: conditional
    failure_class: inherent
    policy: fetch/policy.yaml
    notes: "Fetches ARBITRARY user-supplied URLs by design. Sandboxable only with an explicit per-deployment host list; a fully general fetch server cannot be allowlisted in advance. This is inherent, not a Warden bug."
    probe_allow_files: ["data/cache"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["example.com"]
    probe_allow_env: []

  - name: kubernetes
    upstream: "community: kubernetes-mcp"
    verdict: conditional
    failure_class: inherent
    policy: kubernetes/policy.yaml
    notes: "Needs the cluster API host (per deployment) plus a read grant on the kubeconfig. Works once those two deployment-specific grants are set."
    probe_allow_files: ["config/kubeconfig"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["k8s.internal.example.com"]
    probe_allow_env: ["KUBECONFIG"]

  - name: docker
    upstream: "community: docker-mcp"
    verdict: fail
    failure_class: inherent
    policy: docker/policy.yaml
    notes: "Requires the Docker daemon socket (/var/run/docker.sock), i.e. full host container control. Granting the socket voids the sandbox; there is no scoped-down grant that preserves both function and isolation. FAIL by design."
    probe_allow_files: ["data"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: []
    probe_allow_env: []

  - name: playwright
    upstream: "community: playwright-mcp (browser automation)"
    verdict: fail
    failure_class: schema-gap
    policy: playwright/policy.yaml
    notes: "Two gaps: (1) the schema has no wildcard hosts, but a browser visits arbitrary domains; (2) the schema has no unix-socket grant for the browser IPC, and nested browser sandboxes need syscalls Warden does not mediate. Tracked as schema gaps, not planned for M8."
    probe_allow_files: ["data/profile"]
    probe_deny_files: ["/etc/shadow"]
    probe_allow_hosts: ["example.com"]
    probe_allow_env: ["PLAYWRIGHT_BROWSERS_PATH"]
```

## Try it yourself

```bash
# Copy the fixture closest to your server and point `command` at your install:
cp testdata/compat/github/policy.yaml ./policy.yaml
warden run --policy ./policy.yaml

# Unsure what your server touches? Trace first, then generate:
warden trace -- /usr/bin/node ./server/dist/index.js
warden init
warden logs --tail 20
```

Found friction? File a [compatibility report](./beta.md#filing-a-compatibility-report)
— that is exactly the signal the beta program is collecting.
