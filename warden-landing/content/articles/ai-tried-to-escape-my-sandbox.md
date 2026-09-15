---
title: "AI Tried to Escape My Sandbox. It Couldn't."
description: "I built Warden to sandbox MCP servers, then gave AI a scenario to break out. On Linux 8/8 proof-harness steps held; on Windows 5/5 attack scenarios were blocked. Here's the evidence."
date: "2026-09-15"
author: prof-bilal
category: security
tags: ["mcp", "sandboxing", "escape-tests", "evidence"]
image: /diagrams/deny-by-default-filesystem.webp
draft: false
---

I was scared to use MCP servers. Not because they don't work — because they work too well. An MCP server runs as a plain process with your full permissions: SSH keys, `.env` files, AWS credentials, everything. I asked AI tools to restrict themselves while scanning my repos. They read the sensitive files anyway.

So I built Warden — OS-native sandboxing with deny-by-default policies. Then I did the obvious adversarial test: I gave AI a scenario and asked it to escape.

## The experiment

On Linux, the scenario was simple: read files outside the granted directory, connect to blocked hosts, reach for the `.env` file. On Windows, a friend ran five full attack scenarios: filesystem exfiltration (SSH keys, `.env`, AWS creds), network exfiltration (system fingerprint to three hosts), an env-stealer (three secrets hidden among 59 env vars), a process spawner, and symlink traversal with path tricks and UNC paths.

## The result

The boundary held on both platforms.

**Linux:** the proof harness passed 8/8 steps with verdict `ok` — allowed reads and writes succeeded, secret reads, unlisted paths, blocked hosts, and denied env vars were all contained, and the positive control confirmed the sandbox actually ran the target:

```bash
cd warden-starter/warden
go build ./cmd/warden
bash testdata/proof/run-proof.sh ./warden
# evidence/linux/<timestamp>/summary.json →
# steps_passed: 8, steps_failed: 0, verdict: "ok"
```

**Windows:** all five attack scenarios BLOCKED in 27–38ms each. Protection rate: 100%. The most telling detail: in one configuration the AppContainer backend couldn't initialize without admin rights — and Warden refused to run at all rather than executing unsandboxed. Fail-closed isn't a slogan; it's the code path (`sandboxerr.RefuseToRun`).

## Why it held

Asking AI to restrict itself operates at the application layer — instructions it can ignore. Sandboxing operates at the OS layer:

- **Linux:** bubblewrap bind-mounts only granted paths into a user namespace. Ungranted paths aren't "permission denied" — they're invisible. The integration test asserts exactly that: `ls` on an unlisted directory prints `NOT_VISIBLE`.
- **Windows:** AppContainer tokens strip filesystem capabilities, WFP filters block every outbound connection except the loopback proxy bridge, and ETW captures the audit trail.
- **Network:** a private namespace with no external route. The only way out is a local proxy that checks the hostname allowlist *before* DNS resolution — blocked hosts never reach a resolver.

The process either has the permission or it doesn't. There is nothing to argue with.

## What this doesn't prove

Honesty matters more than a good headline. This proves the tested escape paths are contained on the tested platforms — not that no bypass exists anywhere. Warden also doesn't cover AI with full PC access, remote MCP servers (only local subprocesses), or installs that run directly on your machine. Those limits are documented, not buried.

## Try to break it yourself

That is genuinely the feedback the project needs most:

```bash
npm install -g warden-sandbox-cli

# See what your server actually touches
warden trace -- node your-server.js

# Generate a starter policy from the trace
warden init

# Run sandboxed
warden run --policy policy.yaml -- node your-server.js

# Read the audit log
warden logs --tail 50
```

If something gets through that shouldn't have, file it as a compatibility report. An escape nobody reports is an escape nobody fixes.
