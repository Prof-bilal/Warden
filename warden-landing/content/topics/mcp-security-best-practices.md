---
title: MCP security best practices
description: Practical rules for running local MCP servers, from what the protocol does not protect to how to grant files, hosts, and secrets.
section: guides
slug: mcp-security-best-practices
date: "2026-09-30"
answer: Treat every MCP server as untrusted local code. Give it a directory, a host list, and named secrets. Log what it tries. Do not run it with your full user account just because the client made install a one-liner.
tldr:
  - "The protocol is a transport. Isolation is a separate decision."
  - "Grant paths, hosts, and env names. Do not grant a home directory or a raw shell environment."
  - "Review denials, and do not keep a server that only works with the Docker socket or arbitrary outbound hosts."
faqs:
  - question: Are official MCP servers safe to run unsandboxed?
    answer: '"Official" describes who published the package, not what the process can do. A filesystem server from a known org can still read every file your user can read if you launch it that way. Sandbox it and grant the directory you mean.'
  - question: Should I block all network access?
    answer: Yes when the server is local-only. A search or GitHub server needs specific hosts. List those hosts. Do not leave outbound access open because one tool needs one API.
related:
  - href: /guides/mcp-server-security
    label: How to securely run an MCP server
  - href: /guides/mcp-server-permissions
    label: MCP server permissions, field by field
  - href: /guides/protect-api-keys-from-mcp-servers
    label: Keeping API keys out of an MCP server
  - href: /docs/security
    label: Warden's security review
---

These practices assume a local stdio server started by Claude, an IDE, or another MCP client on your computer. Remote HTTP MCP endpoints are a different trust decision: you are sending prompts to someone else's host, not handing them your disk.

## Before you install

Read the package enough to answer three questions: which directories it touches, which hosts it calls, and which environment variables it reads. If the README cannot answer those, assume the worst until a trace shows otherwise.

Prefer a server you can pin and re-test. A floating `npx -y` install runs whatever the registry serves today, as you.

## While it runs

- **One project, not `$HOME`.** Home-directory grants pull in SSH keys, shell history, and other projects.
- **Hosts, not "the internet."** Name `api.github.com` if that is the API. A browser-style server that must visit any URL cannot be expressed as a static allowlist. That is a reason to refuse it, or to accept a much larger blast radius on purpose.
- **Names, not a dumped environment.** Copying the parent environment into the server gives it every token loaded for other tools.
- **No Docker socket.** Mounting `/var/run/docker.sock` is host control. There is no scoped version of that socket.
- **Logs you can read.** A denial should name the path or host. That is how you tell a missing grant from a server that is probing.

## After something looks wrong

Check the audit log before you widen the policy. A server that "needs" `~/.aws` on a task that should only read `./data` is telling you about its behavior. Tighten the policy or stop using the server.

Re-run the same policy on the operating systems you actually use. A profile that was only reasoned about from Linux is not evidence about Windows.

## Where Warden fits

Warden encodes the list above as a deny-by-default YAML file and refuses to start when the sandbox backend is unavailable. `warden trace` shows what a server touches, and `warden init` turns that trace into a starter policy you still have to read. Trace mode itself is unsandboxed, so treat the trace log as sensitive and delete it after you draft the policy. That limitation is documented in the [FAQ](/docs/faq).

The threat model, including credential exposure and what Warden does not cover, is the [security review](/docs/security).
