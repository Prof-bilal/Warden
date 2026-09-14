---
title: "Warden backends: bubblewrap, Seatbelt, AppContainer"
description: "How Warden maps one deny-by-default policy onto three different OS sandbox mechanisms — Linux bubblewrap, macOS Seatbelt, and Windows AppContainer with WFP and Job Objects."
date: "2026-09-14"
author: prof-bilal
category: engineering
tags: ["linux", "windows"]
image: /diagrams/linux-backend.webp
draft: false
sample: true
---

Every operating system already ships a kernel-level sandbox mechanism. Warden doesn't invent one — it compiles the same policy file into whichever mechanism the host provides. This article explains the three backends and what each one actually enforces.

## Linux: bubblewrap

On Linux, Warden uses [bubblewrap](https://github.com/containers/bubblewrap) (bwrap) to build the server's view of the filesystem from scratch:

![Diagram of the Linux bubblewrap backend building a mount namespace](/diagrams/linux-backend.webp)

- Only paths granted in the policy are bind-mounted into the new mount namespace; everything else simply does not exist to the process
- Network egress is mediated by a proxy so allowlisted hosts are enforced outside the server's own process, where it can't be bypassed from inside
- CPU and memory limits are applied through kernel resource controls

Because the grants are enforced by the kernel through namespaces, a compromised server process cannot escape them by misbehaving in userspace.

## macOS: Seatbelt

On macOS, Warden compiles the policy into a Seatbelt (sandbox-exec) profile. Seatbelt operates on the same principle — operations not explicitly allowed by the profile are denied by the kernel — so the deny-by-default model maps directly. Where the Seatbelt backend cannot yet guarantee parity with Linux for a given grant, Warden falls back to a Docker-based sandbox and says so, rather than pretending the native backend is stricter than it is.

## Windows: AppContainer + WFP + Job Objects

Windows support uses three cooperating layers:

![Diagram of the Windows backend layers: AppContainer, WFP egress filters, and Job Objects](/diagrams/windows-backend.webp)

- **AppContainer** — the server runs with a LowBox token that constrains filesystem and object access to explicitly granted capabilities
- **WFP egress filters** — Windows Filtering Platform rules enforce the network allowlist at the kernel level
- **Job Objects** — CPU and memory limits, with the whole job terminated on breach
- **ETW auditing** — enforcement events are captured through Event Tracing for Windows into the JSONL audit stream

## One policy, equivalent guarantees

The point of the backend architecture is that a policy file means the same thing everywhere: the same filesystem grants, the same host allowlist, the same env filtering, the same fail-closed behavior on startup failure. The full [feature reference](/features) documents each backend's current verification status, and the [testing page](/testing) shows the cross-platform test matrix.

## Conclusion

You don't need to learn three sandbox APIs to run MCP servers safely on three platforms. Write the policy once; Warden translates it into the primitives your OS already provides. For the enforcement path in detail, see [how Warden enforces policies](/blog/how-warden-enforces-policies).
