---
title: How to protect API keys from MCP servers
description: MCP servers inherit your shell environment unless you filter it. How to pass one token without exposing the rest, and which setups still leak keys.
section: guides
slug: protect-api-keys-from-mcp-servers
date: "2026-09-30"
answer: An MCP server should see only the API key names it needs. Filter the environment before the process starts, and do not grant filesystem access to the files where other keys live.
tldr:
  - "A normal launch copies your shell environment into the server, tokens included."
  - "Allow a variable name. Keep the value out of the policy file and out of logs you commit."
  - "Filesystem access to ~/.ssh, ~/.aws, or a project .env leaks keys even if the environment was filtered."
faqs:
  - question: If I only use a GitHub MCP server, can it see my OpenAI key?
    answer: Yes, if that key is in the environment of the process that started the server and you did not filter env. The server does not need to be malicious. It only needs to be able to read the environment, which every process can.
  - question: Does Warden store my token in policy.yaml?
    answer: No. The policy lists the variable name, such as GITHUB_TOKEN. The value stays in the parent environment and is forwarded only when that name is allowed.
related:
  - href: /guides/mcp-server-permissions
    label: What permissions an MCP server should have
  - href: /guides/mcp-security-best-practices
    label: MCP security best practices
  - href: /use-cases/run-untrusted-mcp-servers
    label: Running an MCP server you do not trust
  - href: /docs/security
    label: Credential exposure in the security review
---

API keys leak into MCP servers in two ways that people mix up.

The first is the environment. Clients often start the server as a child of a shell or desktop app that already loaded secrets for other tools. Child processes receive a copy of that block unless something filters it.

The second is the filesystem. Keys also live in `~/.aws`, `~/.ssh`, `.env`, and CI config. An environment filter does nothing for a server that can read those files.

You need both controls.

## Environment

Decide the names. A GitHub server might need `GITHUB_TOKEN` and nothing else. Launch it so the process environment contains that name and not `OPENAI_API_KEY`, `AWS_SECRET_ACCESS_KEY`, or a cloud credential you use for deploy.

Do not write the secret into a policy file, a dotfile you commit, or a trace log you paste into a ticket. If you trace a server to see what it uses, remember that a trace may record values. Delete the trace after you have the names.

## Filesystem

Grant the project directory the server is supposed to edit. Do not grant `$HOME`. Do not grant the directory that holds your `.env` unless that file is the one credential this server must read, and you accept that it can read it.

A common mistake is allowing `HOME` through the environment and also allowing read on a parent of `~/.ssh`. The process can then open the key files by path. Remove one side of that pair or, better, both.

## What still will not save you

A server that must hold a token can still send that token to an allowed host, or to an extra host if the network allowlist is empty-by-accident and your tool fails open. Pair the env filter with a host allowlist.

A server you run with the Docker socket, or unsandboxed "just this once," can read the files anyway. The exception is the leak.

## Where Warden fits

Warden's `env.allow` list is the filter: only those names are forwarded, and the YAML never contains values. Filesystem grants are separate, so a token name does not imply read access to `~/.ssh`. Network grants are separate again, so possession of a token does not imply permission to call an arbitrary host.

`warden trace` is the exception to treat carefully. It runs unsandboxed so it can observe real accesses, and observed values can appear in the log. The [FAQ](/docs/faq) says to treat that log as sensitive. Use it to learn names, then switch to `warden run` under the policy.
