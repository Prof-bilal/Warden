---
title: "How Warden Enforces Policies"
description: "A walkthrough of the Warden enforcement path: how a policy.yaml becomes OS-level sandbox rules, how deny-by-default checks run, and what happens on failure."
date: "2026-09-14"
author: prof-bilal
category: engineering
tags: ["policy", "internals"]
image: /diagrams/policy-flow.webp
draft: false
---

A Warden policy is a small YAML file that declares exactly what an MCP server is allowed to do. This article walks the path that file takes from `warden run` to the kernel, and why each step is designed to fail closed.

![Diagram of the Warden policy enforcement flow from policy file to sandbox](/diagrams/policy-flow.webp)

## The policy file

A minimal policy declares a command and its grants:

```yaml
command: npx -y @modelcontextprotocol/server-filesystem ./workspace

filesystem:
  read:
    - ./workspace
  write:
    - ./workspace/tmp

network:
  allow:
    - host: api.example.com
      port: 443

env:
  allow:
    - API_TOKEN

limits:
  maxMemoryMb: 1024
```

Everything not listed is denied. You don't write deny rules; absence of an allow rule *is* the deny rule.

`warden init` generates a first draft of this file by inspecting the server you want to run, and `warden trace` records what the server actually attempts so you can tighten the policy against real behavior instead of guessing.

## From YAML to OS primitives

At launch, Warden compiles the policy into the sandbox primitive of the host platform:

- **Linux**bubblewrap namespaces and bind mounts: only granted paths exist inside the mount namespace, and a network proxy mediates egress
- **macOS**Seatbelt (sandbox-exec) profiles compiled from the same policy
- **Windows**AppContainer LowBox tokens plus Windows Filtering Platform (WFP) egress filters and Job Objects for resource limits

The same policy file produces equivalent enforcement on each platform. If the native backend can't provide the requested guarantee, Warden refuses to start the server rather than silently running with weaker isolation.

## Fail-closed startup

Ordering matters at startup. The sandbox is fully constructedmounts, filters, env filtering, limits*before* the server process is spawned. If any step fails, Warden aborts the launch and writes the reason to the audit log. There is no mode where a policy error results in an unsandboxed run.

## Runtime denial

During execution, every blocked access is recorded:

```json
{"ts":"2026-09-14T10:00:00Z","verdict":"deny","kind":"net","target":"unknown.example.com:443"}
```

The stream is JSONL, one event per line, so it can be piped into any log tooling. `warden logs` tails it locally.

## Why not just use Docker?

Container images add an indirection layer, but they don't solve the grant problem: a container with default settings still has broad network access and whatever volumes you mount. Warden's deny-by-default policy, kernel-level network filtering, and per-platform enforcement are a tighter boundary for the specific job of running MCP serversand they work on the machine you already have, without a daemon. The [about page](/about) covers this trade-off in more detail.

## Try it

Install with `npm install -g warden-sandbox-cli`, then follow the [quickstart](/docs/quickstart) or read about the [sandboxing backends](/blog/sandboxing-backends-bubblewrap-seatbelt-appcontainer) in depth.
