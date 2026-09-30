---
title: How to run an untrusted MCP server
description: A practical way to try an MCP server you did not write, without giving it your SSH keys, your .env files, or a general network.
section: use-cases
slug: run-untrusted-mcp-servers
date: "2026-09-30"
answer: Run an untrusted MCP server inside a sandbox that starts empty. Grant the single directory and hosts you are willing to lose. Trace it once if you do not know what it needs, then delete the trace. If it only works with your home directory or the Docker socket, do not run it.
tldr:
  - "Untrusted means a registry package, a gist, or any server you did not audit."
  - "Start with no grants and add only what a denial log justifies."
  - "Some servers cannot be sandboxed honestly. The Docker socket and arbitrary browsing are the usual reasons."
faqs:
  - question: Is a popular MCP server trusted?
    answer: Popularity is not a permission boundary. A widely installed filesystem server still runs as you unless you sandbox it. Trust the policy you can read, not the download count.
  - question: Can Warden make every MCP server safe?
    answer: No. Servers that need the Docker socket, or that must contact arbitrary hosts, do not become safe because a wrapper launched them. The compatibility matrix records those as failures instead of granting the dangerous thing.
related:
  - href: /guides/mcp-server-security
    label: How to securely run an MCP server
  - href: /guides/protect-api-keys-from-mcp-servers
    label: How to protect API keys from MCP servers
  - href: /use-cases/secure-local-mcp-servers
    label: Securing the local servers you already use
  - href: /blog/why-mcp-servers-need-sandboxes
    label: Why MCP servers need a sandbox at all
---

"Untrusted" is the normal case. The interesting MCP servers are the ones someone else published. You want the tool. You do not want a one-line install to become a copy of your account.

## A sequence that does not start with full access

1. Install the package the way you would, but do not point the client at it yet.
2. Write a policy with the command and empty grants.
3. Start it under the sandbox. Expect denials.
4. Read the log. If the denial is the project folder you meant, add that folder. If the denial is `~/.ssh` or a credentials file, stop and decide whether this server is worth it.
5. Add hosts only when the server cannot do its stated job without them.
6. Put the wrapped command into the client config after the policy is something you would approve in a review.

If you truly do not know what it will touch, a trace records accesses. On Warden, trace mode is unsandboxed so the observation is real. That log can contain secret values. Generate the policy, then delete the log. Do not commit it.

## Servers that should stay refused

The published matrix is useful here because it includes failures.

- A server that needs `/var/run/docker.sock` is asking for the host. Do not grant it to "make the test pass."
- A server that fetches arbitrary URLs cannot be pinned to a host list. Running it means you accept open outbound network.
- A server whose first run demands your entire home directory is telling you it was written for the unsandboxed default. Sandboxing it properly may mean not using it.

Fourteen of the eighteen tested servers pass with a small pinned policy, including filesystem, GitHub, Slack, Postgres, and several community servers. Two more pass only after you add a deployment-specific host. The point of the two failures is that a sandbox tool should be allowed to say no. The list is [I tested 18 MCP servers](/blog/i-tested-18-mcp-servers).

## Where Warden fits

Warden is the runner for this sequence: empty-by-default policy, denial log, fail closed, and a matrix that records servers which do not fit. It will not vouch for a package's author. It will stop that package from starting life as your user.
