---
title: "I Tested 18 MCP Servers Against a Sandbox. Here's What Happened."
description: "14 pass, 2 conditional, 2 fail — the full compatibility matrix for running real-world MCP servers under Warden, with pinned policies and honest failure classifications."
date: "2026-09-15"
author: prof-bilal
category: engineering
tags: ["mcp", "compatibility", "policies"]
image: /diagrams/ci-pipeline.webp
draft: false
---

Security claims are cheap. Compatibility data is not. So Warden is tested against 18 real MCP servers with exact pinned policies — and the results are published with failures included:

**Totals: 14 pass · 2 conditional · 2 fail.**

## The 14 that just work

Filesystem, GitHub, Slack, PostgreSQL, SQLite, Brave Search, Google Drive, Git, Memory, Time, Sequential Thinking, plus community Notion, Linear, and Tavily servers. Each ships with a pinned regression fixture under `testdata/compat/`, enforced by `go test ./internal/compat/` — so an update can't silently break what used to work.

A typical passing policy is small. The filesystem server needs exactly this and nothing else:

```yaml
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./data"
  write: []
network:
  allow: []
env:
  allow: []
```

## The 2 conditional

**Fetch** retrieves arbitrary user-supplied URLs by design — no static allowlist can cover it. It works once you supply a per-deployment host list. **Kubernetes** needs its cluster API host configured. Both are "works with setup", not "broken".

## The 2 that fail — and why that's the point

**Docker-mcp** needs `/var/run/docker.sock`. Granting the socket hands over full host control; there is no scoped version of it. The failure is inherent to the server's design, not a Warden gap.

**Playwright** visits arbitrary domains, which the static hostname schema couldn't express when it was classified. That failure drove a schema improvement — the failure classification *is* the roadmap input.

## What the triage process looks like

Every failure gets one of three labels: Warden's bug (fix the code), schema gap (extend the policy language), or inherent (document it and move on). The Slack read/write overlap bug was the first kind — same path in both lists was rejected as ambiguous until `policy.Normalize` learned write-wins coalescing. Bare launcher names (`npx`, `uvx`) failing with "not an absolute path" was fixed the same way, via `PATH` lookup.

## Before you install your next MCP server

Check the matrix first. Copy the closest fixture policy, tighten it, and run:

```bash
cp testdata/compat/github/policy.yaml ./policy.yaml
warden run --policy ./policy.yaml -- /usr/bin/node server.js
warden logs --tail 50
```

If your server isn't listed, `warden trace` it and file what you find. That's how the matrix grows from 18 to whatever comes next.
