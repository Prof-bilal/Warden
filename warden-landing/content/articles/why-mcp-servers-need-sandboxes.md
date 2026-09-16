---
title: "Why MCP Servers Need Sandboxes"
description: "MCP servers run as local processes with your full user permissions. Here's why deny-by-default sandboxing is the missing trust boundary between AI tooling and your machine."
date: "2026-09-14"
author: prof-bilal
category: security
tags: ["mcp", "sandboxing"]
image: /diagrams/sandbox-boundary.webp
draft: false
---

An MCP (Model Context Protocol) server is a program that an AI client launches on your machine and talks to over stdio. From the operating system's point of view, that server is just another process you started. It runs as your user, with your permissions: it can read every file you can read, open connections to any host you can reach, and see every environment variable in your shell.

For first-party tools that trust boundary is fine. For third-party MCP servers pulled from a registry, it is not. The server author decides what the process does, and the AI client decides what it asks the server to dobut nothing on your side constrains either of them.

## The permission gap

A typical MCP server needs very little: a few files in one project directory, network access to one API host, and maybe a token or two. Almost no MCP server needs your whole home directory, your SSH keys, or your `.env` files. Yet by default, that is exactly what it gets.

Warden closes this gap by running MCP servers inside an OS-native sandbox with **deny-by-default** grants. Nothing is accessiblefiles, network hosts, environment variablesunless the policy explicitly allows it.

![Diagram of a sandbox boundary separating an AI client from a sandboxed MCP server](/diagrams/sandbox-boundary.webp)

## Deny by default, not allow by default

Most sandboxing tooling starts from "allow everything, then restrict." Warden inverts that: a server starts with no capabilities at all. If the policy file doesn't grant a path, a host, or a variable, the attempt is deniedand the denial is recorded.

This matters because misconfiguration then fails safe. A policy typo denies access instead of leaking it. Warden calls this **fail-closed**: if the sandbox itself cannot start correctly, the server does not run at all.

## What gets constrained

A Warden policy can restrict four things per server:

- **Filesystem**specific read or read-write paths, everything else denied
- **Network**allowed hosts and ports, enforced at the kernel level rather than by the server itself
- **Environment**the exact variables the server may see, filtered before the process starts
- **Resources**CPU and memory limits so a runaway server can't take the machine down

## Auditability

Every allow and deny decision is written to a JSONL audit log. When an agent behaves strangely, you can answer "what did this server actually try to do?" after the fact instead of guessing.

## The takeaway

AI tooling is moving toward running more third-party code locally. The protocol gives you a clean interface; the sandbox gives you the missing trust boundary. If you run MCP servers you didn't write, running them unsandboxed means trusting them with everything your user can reach.

Warden is open sourcesee the [quickstart](/docs/quickstart) to try it, or read [how Warden enforces policies](/blog/how-warden-enforces-policies) for the mechanics.
