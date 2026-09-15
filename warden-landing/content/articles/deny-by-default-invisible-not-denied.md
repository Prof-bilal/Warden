---
title: "Deny-by-Default: Why 'Permission Denied' Isn't Enough"
description: "Warden makes ungranted paths invisible instead of unreadable — bind mounts, not permission bits. Why invisibility is the stronger boundary, and how it's tested."
date: "2026-09-15"
author: prof-bilal
category: security
tags: ["sandboxing", "deny-by-default", "architecture"]
image: /diagrams/filesystem-capabilities.webp
draft: false
---

Most developers treat "permission denied" as security. It isn't — it's a notification. When a process gets `EACCES`, it learns the resource exists. It can retry. It can race. It can probe for escalation. The OS said no *this time*.

Warden's filesystem model says something stronger: the resource does not exist. A sandboxed process listing an ungranted directory doesn't get "permission denied" — it gets nothing. The project's integration test asserts exactly this: `ls` on a sibling directory outside the grant prints `NOT_VISIBLE`.

## How invisibility is built

On Linux, only granted paths are bind-mounted into the bubblewrap sandbox's user namespace. Everything else simply isn't in the mount table. There is no permission check to bypass because there is no path to check against.

On Windows, the AppContainer LowBox token strips the capabilities that would let the process reach outside its box. On macOS, the Seatbelt profile denies the read at the policy layer.

Same guarantee, three mechanisms: the process cannot name what it cannot see.

## Why this distinction matters for AI tooling

An AI agent that hits "permission denied" does what agents do: it tries another path, rephrases the request, looks for a symlink, checks the parent directory. Each attempt is fine in isolation and dangerous in aggregate. Against invisibility, there is no second attempt worth making — the directory isn't there.

This is also why the audit log matters more than the block. The log shows what the server *tried* to touch: the exact paths, hosts, and variables. That record is how you learn what your MCP server actually wants, and it's how policies get tightened from observed behavior (`warden trace` → `warden init` → `warden run`).

## The policy that expresses it

```yaml
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
```

Four stanzas. No roles, no capabilities, no inheritance. The schema even rejects unknown keys, so a typo fails loudly instead of silently widening access. Review your MCP server's actual permissions tonight: are they "denied", or are they "invisible"? Only one of those survives contact with a motivated process.
