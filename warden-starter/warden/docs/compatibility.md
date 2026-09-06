# MCP Server Compatibility

Will Warden work with *your* MCP server? This page is the public,
tested answer. Every row below has a permanent regression fixture under
[`testdata/compat/`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat)
— the exact `policy.yaml` the server needs — enforced by
`go test ./internal/compat/`, so a future Warden change cannot silently
break a server that used to work.

> **How to read the verdicts:** ✅ **pass** works with the linked policy as
> written. ⚠️ **conditional** works once you fill in the deployment-specific
> grants noted. ❌ **fail** cannot be sandboxed today; the reason column says
> whether that is a Warden bug, a policy-schema gap, or inherent to the
> server's design.

## Matrix (18 servers)

| Server | Upstream | Verdict | Policy |
|---|---|---|---|
| Filesystem | `@modelcontextprotocol/server-filesystem` | ✅ pass | [`filesystem/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/filesystem/policy.yaml) |
| GitHub | `@modelcontextprotocol/server-github` | ✅ pass | [`github/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/github/policy.yaml) |
| Slack | `@modelcontextprotocol/server-slack` | ✅ pass | [`slack/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/slack/policy.yaml) |
| PostgreSQL | `@modelcontextprotocol/server-postgres` | ✅ pass | [`postgres/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/postgres/policy.yaml) |
| SQLite | `@modelcontextprotocol/server-sqlite` | ✅ pass | [`sqlite/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/sqlite/policy.yaml) |
| Brave Search | `@modelcontextprotocol/server-brave-search` | ✅ pass | [`brave-search/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/brave-search/policy.yaml) |
| Google Drive | `@modelcontextprotocol/server-gdrive` | ✅ pass | [`gdrive/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/gdrive/policy.yaml) |
| Git | `@modelcontextprotocol/server-git` | ✅ pass | [`git/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/git/policy.yaml) |
| Memory | `@modelcontextprotocol/server-memory` | ✅ pass | [`memory/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/memory/policy.yaml) |
| Time | `@modelcontextprotocol/server-time` | ✅ pass | [`time/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/time/policy.yaml) |
| Sequential Thinking | `@modelcontextprotocol/server-sequential-thinking` | ✅ pass | [`sequential-thinking/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/sequential-thinking/policy.yaml) |
| Notion (community) | `notion-mcp` | ✅ pass | [`notion/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/notion/policy.yaml) |
| Linear (community) | `linear-mcp` | ✅ pass | [`linear/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/linear/policy.yaml) |
| Tavily search (community) | `tavily-mcp` | ✅ pass | [`tavily/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/tavily/policy.yaml) |
| Fetch | `@modelcontextprotocol/server-fetch` | ⚠️ conditional | [`fetch/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/fetch/policy.yaml) |
| Kubernetes (community) | `kubernetes-mcp` | ⚠️ conditional | [`kubernetes/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/kubernetes/policy.yaml) |
| Docker (community) | `docker-mcp` | ❌ fail (inherent) | [`docker/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/docker/policy.yaml) |
| Playwright (community) | `playwright-mcp` | ❌ fail (schema gap) | [`playwright/policy.yaml`](https://github.com/Prof-bilal/Warden/tree/main/warden-starter/warden/testdata/compat/playwright/policy.yaml) |

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
