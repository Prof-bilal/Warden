---
title: What permissions should an MCP server have?
description: How to decide filesystem, network, and environment permissions for a local MCP server without handing it your user account.
section: guides
slug: mcp-server-permissions
date: "2026-09-30"
answer: An MCP server should receive the minimum filesystem paths, network hosts, and environment variable names its task requires. It should not receive your home directory, a general outbound network, or the parent shell's environment.
tldr:
  - "Start from no access and add grants the server can justify."
  - "Separate read paths from write paths. A database usually needs write on its directory, not on the repo."
  - "Passing HOME plus a broad filesystem grant exposes SSH keys and other credentials."
faqs:
  - question: Why did my server fail until I granted a parent directory?
    answer: Sandboxes mount the paths you name. A file grant does not imply the parent, and a write to a SQLite file often needs the directory so the engine can create journal files beside it. The denial log names the path that was missing.
  - question: Should two MCP servers share one policy?
    answer: 'No. A filesystem server and a GitHub server need different hosts and directories. One shared "developer" policy becomes your full account again.'
related:
  - href: /guides/protect-api-keys-from-mcp-servers
    label: How to protect API keys from MCP servers
  - href: /use-cases/mcp-server-filesystem-isolation
    label: Filesystem isolation for MCP servers
  - href: /docs/schema
    label: Warden policy schema
  - href: /docs/examples
    label: Example Warden policies
---

Permissions are the policy, not a feeling about the vendor. Write them down per server.

## Filesystem

Grant the directory the task is about. A docs server that answers questions from `./notes` gets read on `./notes`. It does not get read on `~`.

Write is separate. A server that renders into `./output` gets write there. A server that only searches does not get write at all.

Watch two edge cases that show up in real servers:

- SQLite and similar engines need write on the **directory**, because journals sit next to the database file.
- Overlapping read and write on the same path should be one write grant. Listing it twice is a sign the policy was generated and not read.

## Network

Allow the hosts the server calls, with the port if your tool requires it. Local-only servers get an empty allowlist.

A server whose job is "fetch whatever URL the model supplies" cannot be locked to a fixed host list. You can still run it, but you should describe that choice as "this process may contact arbitrary hosts," not as a locked-down policy.

## Environment

Allow variable **names**. The values remain in the environment you launch from. Do not put secrets in the policy file.

Do not pass `HOME` unless you have a specific reason and a filesystem policy that still excludes credential directories. `HOME` plus a read grant on a parent of `~/.ssh` is how keys leak. The [security review](/docs/security) spells that footgun out.

## Where Warden fits

A Warden policy is that list and nothing else. Unknown keys are rejected, so a typo does not silently widen access. Relative paths resolve against the policy file, not against whatever directory your shell happened to use.

```yaml
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read: ["./data"]
  write: []
network:
  allow: []
env:
  allow: []
```

That is the pinned filesystem-server fixture from the compatibility set: local reads, no writes, no network, no env. Other servers in the [example policies](/docs/examples) add a host or a token name. Copy the closest one and delete grants you cannot explain.
