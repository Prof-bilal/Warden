---
title: How to run local MCP servers securely
description: Local stdio MCP servers are the common case. How to keep the client setup you already have and stop the server from inheriting your account.
section: use-cases
slug: secure-local-mcp-servers
date: "2026-09-30"
answer: Keep the MCP client. Change the command it launches so a sandbox starts first and the server starts second. Grant one project directory, the hosts that server calls, and the env names it needs.
tldr:
  - "Local MCP means the server is a child process on your machine, usually over stdio."
  - "Point the client at a wrapper command instead of at npx directly."
  - "Warden is that wrapper. The client still speaks MCP."
faqs:
  - question: Do I have to change my prompts or tools?
    answer: No. The client still sends the same MCP messages. You change the command line that starts the server so it runs under a policy.
  - question: What about remote MCP servers hosted by a vendor?
    answer: This page is about local processes. A remote server never sees your disk, but you are sending the conversation to that vendor. Sandboxing the local process does not apply.
related:
  - href: /guides/how-to-sandbox-mcp-server
    label: How to sandbox an MCP server
  - href: /use-cases/run-untrusted-mcp-servers
    label: Running a server you did not write
  - href: /docs/quickstart
    label: Warden quickstart
  - href: /docs/client-proxy
    label: Warden's client proxy
---

Most MCP setups are local. The desktop app or IDE spawns `npx`, `uvx`, or a Node entrypoint and speaks JSON-RPC on stdin and stdout. There is no separate machine to configure. The security job is entirely about that child process.

## A configuration that stays familiar

Leave the client, the tools, and the prompts alone. Replace the command:

```bash
warden run --policy ./github.yaml -- npx -y @modelcontextprotocol/server-github
```

The policy file sits next to the project and is what you review in pull requests. The client config stores the command, the same way it stored `npx` before.

## What to put in the policy for a local server

- The directory that server is allowed to see. A repo, not your home folder.
- Hosts only if it calls an API. A filesystem or memory server often needs none.
- One token name if it calls an API. Not the rest of the environment.
- A memory limit so a stuck server is killed with its children.

Then read `warden logs` the first time a tool fails. Add the path the log names, or decide the tool should keep failing.

## What local sandboxing does not cover

It does not inspect the model's judgment. A granted directory can still be exfiltrated to a granted host. The policy makes that path explicit so you can see it.

It does not protect a server you installed by bypassing the wrapper "to test." The client config is the control. If a second config still calls `npx` directly, that copy is unsandboxed.

## Where Warden fits

This is the case Warden is built for: local stdio servers, one policy each, fail closed when the OS sandbox is unavailable. The [quickstart](/docs/quickstart) is the shortest path. If several servers are declared in a gateway file, `warden gateway` can prefix the local ones with `warden run` instead of you editing every command by hand. Remote endpoints in that file are skipped, because there is no local process to sandbox. That behavior is documented in the [CLI reference](/docs/cli).
