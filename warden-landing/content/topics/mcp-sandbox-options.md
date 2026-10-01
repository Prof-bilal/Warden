---
title: Ways to isolate an MCP server
description: Unsandboxed processes, separate OS users, containers, hand-written OS sandboxes, and a policy runtime. What each option actually stops.
section: compare
slug: mcp-sandbox-options
date: "2026-09-30"
answer: You can isolate an MCP server with a separate OS user, a container, or a kernel sandbox. A separate user is the weakest of the three. A container or an OS sandbox works only when the filesystem, network, and environment are all narrowed, and when a failed setup refuses to start the server.
tldr:
  - "Doing nothing means the server is you, for filesystem, network, and secrets."
  - "Containers and OS sandboxes can both be strict. The policy is the flags or the profile, not the product name."
  - "Warden is the option that turns one deny-by-default file into the host sandbox and aborts if that fails."
faqs:
  - question: What is the best sandbox for MCP servers?
    answer: The best option is the one you will actually configure and notice when it fails. For a local stdio server, that is usually an OS sandbox with an explicit grant list. A container is a good option when you also need a custom image. Running as your normal user is not a sandbox.
  - question: Is there an official MCP sandbox?
    answer: The protocol does not ship one. Isolation is left to the host. Warden is one open-source runtime that fills that gap for local servers. It is not part of the protocol, and it is not the only way to build a sandbox.
related:
  - href: /compare/warden-vs-docker
    label: Warden compared with Docker
  - href: /guides/how-to-sandbox-mcp-server
    label: How to sandbox an MCP server
  - href: /guides/mcp-server-security
    label: How to securely run an MCP server
  - href: /blog/sandboxing-backends-bubblewrap-seatbelt-appcontainer
    label: How Warden maps one policy onto three OS sandboxes
---

"Best MCP security tool" is usually the wrong search. The useful question is which boundary you are willing to operate, and what still gets through.

## Option 1: launch it as yourself

This is the default in most client configs. The server can read what you can read. It is acceptable for a tool you wrote this afternoon and unacceptable for a package you installed from a registry.

## Option 2: another Unix user

A dedicated user avoids some accidents. It does not build a network allowlist, and people often grant that user the project by making the directory world-accessible or by sharing a group that also covers secrets. Use it as a supplement, not as the policy.

## Option 3: a container

A container with a read-only root, no extra capabilities, one mount, and no default outbound network is a real sandbox. A container with `-v $HOME:$HOME` and the default bridge network is a packaging format. Read the run command. If you cannot point at the lines that limit files and hosts, you do not have a policy yet.

## Option 4: the OS sandbox, written by hand

Linux [bubblewrap](https://github.com/containers/bubblewrap), macOS Seatbelt, and Windows AppContainer can all hide files and constrain processes. They are the strongest common building blocks, and they are easy to mis-bind. Windows in particular needs several primitives at once (AppContainer, egress filters, job limits, an audit trail). Two DLL-binding bugs in Warden's own Windows backend showed up only when CI ran on a real Windows machine, which is written up in [Windows sandboxing is 10x harder](/blog/windows-sandboxing-10x-harder). Hand-rolled profiles have the same class of risk.

## Option 5: a policy runtime

A policy runtime is option 4 with the profile generated from a file you can review, plus a rule that a broken backend does not launch the server. That is the category Warden is in. It is open source, MIT licensed, and costs nothing. It is a local CLI, not a hosted security product.

Published checks, which you can re-run from the repo:

- 18 MCP servers with pinned policies: 14 pass, 2 conditional, 2 fail. [Matrix](/docs/compatibility).
- Linux proof harness: 8 of 8 steps, verdict ok. [Write-up](/blog/ai-tried-to-escape-my-sandbox).
- Windows: five attack scenarios blocked, including files, network, environment, process spawn, and symlink tricks. Same write-up. Those runs do not prove that no other bypass exists.

If you only need one page on how to apply this, start with [how to sandbox an MCP server](/guides/how-to-sandbox-mcp-server).
