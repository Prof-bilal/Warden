---
title: How to securely run an MCP server
description: An MCP server is a local process with your permissions. Here is the attack surface, what to constrain, and how a sandbox policy limits the damage.
section: guides
slug: mcp-server-security
date: "2026-09-30"
answer: An MCP server is secure only to the extent the operating system stops it from using your account. Treat every server as a local program that can read files, open connections, and see environment variables, then grant it just the paths, hosts, and variable names that job needs.
tldr:
  - "MCP does not sandbox the server. The process runs as the user who launched it."
  - "Constrain filesystem paths, network hosts, and environment variables before the process starts."
  - "If the sandbox cannot be built, do not run the server unsandboxed."
faqs:
  - question: Does the MCP protocol sandbox servers?
    answer: No. MCP defines how a client talks to a server, usually over stdio. Permissions are an operating-system problem. A server launched on your machine inherits your user unless a sandbox wraps it.
  - question: What should an MCP security policy cover?
    answer: At minimum, which directories the server may read or write, which hosts it may contact, and which environment variable names it may see. Resource limits and an audit log of denied attempts make the policy checkable.
related:
  - href: /guides/how-to-sandbox-mcp-server
    label: How to sandbox an MCP server
  - href: /guides/mcp-server-permissions
    label: What permissions an MCP server should have
  - href: /guides/protect-api-keys-from-mcp-servers
    label: How to keep API keys away from an MCP server
  - href: /use-cases/run-untrusted-mcp-servers
    label: Running an MCP server you did not write
---

The [Model Context Protocol](https://modelcontextprotocol.io) lets an AI client start a tool server and exchange JSON-RPC over stdio. That design is convenient, and it is also the security boundary people skip. The client asks for tools. The operating system decides what those tools can touch. If you launched the server, the answer is usually "everything you can touch."

## What the server can do by default

A typical install is `npx` or `uvx` in the same shell as your editor. That process can:

- Read `~/.ssh`, cloud credentials, and `.env` files, because those files are readable by your user.
- Open outbound connections to any host your network allows.
- See every environment variable exported in the parent shell, including tokens you meant for a different tool.
- Spawn child processes that inherit the same rights.

None of that requires a bug in the MCP client. It is how local processes work. The protocol specification does not insert a permission check between "tool call" and "read this path."

## The attack surface that matters

Most incidents people worry about fall into four buckets.

| Risk | What happens | What actually limits it |
| --- | --- | --- |
| Filesystem | The server, or the model driving it, reads or writes outside the project | A mount or profile that does not include the rest of the disk |
| Network | Data leaves to a host you did not intend | An allowlist enforced outside the server process |
| Environment | A token in the parent shell is visible inside the server | Pass names, not the whole environment |
| Confused deputy | The model asks a legitimate tool to do an illegitimate thing | A policy that makes the illegitimate path impossible, plus a log of what was attempted |

"The model promised not to" is not one of those limits. Instructions are application-layer. A sandbox is kernel-layer.

## A practical baseline

1. Run the server as its own process, not inside a shell that already holds production secrets.
2. Give it one project directory, not your home directory.
3. List the API hosts it needs. Block DNS and connections for everything else.
4. Forward only the variable names that server documents, such as one API token.
5. Keep a log of denied file and network attempts so the next grant is based on a real denial, not a guess.
6. If the mechanism that enforces those rules fails to start, stop. Do not fall through to a normal process.

That last step is the one most DIY setups miss. A broken container flag or a missing `bwrap` binary often means the server still starts, just without the protection you thought you had.

## Where Warden fits

[Warden](/) is an open-source sandbox runtime that applies this baseline to local MCP servers. A `policy.yaml` names the command and the grants. Warden compiles that file into the host sandbox — bubblewrap on Linux, Seatbelt on macOS, AppContainer on Windows — and starts the server only after the sandbox exists. If no backend can enforce the policy, Warden refuses the run.

The published compatibility set covers 18 MCP servers with pinned policies: 14 pass, 2 work only with extra per-deployment grants, and 2 fail for reasons the matrix records (a Docker socket, and a browser server that needs arbitrary hosts). Those numbers are the test record, not a promise that every future server or every bypass has been tried. The write-up is [I tested 18 MCP servers](/blog/i-tested-18-mcp-servers), and the table is the [compatibility matrix](/docs/compatibility).

Install and the policy fields are in the [install guide](/docs/install) and the [policy schema](/docs/schema).
