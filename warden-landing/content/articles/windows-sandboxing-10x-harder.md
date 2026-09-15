---
title: "Windows Sandboxing Is 10x Harder Than Linux"
description: "Bubblewrap just works. AppContainer + WFP + ETW + Job Objects do not — two DLL binding bugs, caught only by real Windows CI, and what they teach about cross-platform security tools."
date: "2026-09-15"
author: prof-bilal
category: engineering
tags: ["windows", "sandboxing", "appcontainer", "ci"]
image: /diagrams/windows-backend.webp
draft: false
---

On Linux, sandboxing an MCP server took days. Bubblewrap does nearly everything: unprivileged user namespaces, bind mounts for granted paths, a network namespace with no external route. One tool, one model.

On Windows, the same guarantee needs four OS primitives working together: AppContainer tokens for filesystem capabilities, WFP filters for egress, ETW for the audit trail, and Job Objects for memory and timeout limits. Each one is a potential failure point. Two of them failed — and both were caught only because CI ran on a real Windows machine.

## Bug 1: ETW bound to the wrong DLL

The ETW consumer procs (`OpenTraceW`, `ProcessTrace`, `CloseTrace`) were bound to `kernel32.dll`. On modern Windows those functions live in `advapi32.dll`. The test run panicked on the first real CI job. The fix was a three-line move — but cross-compiling from Linux would never have caught it, because the binding only resolves at load time on Windows.

## Bug 2: WFP bound to an absent DLL

`FwpmEngineOpen` was bound to `fwpuclnt.dll` — which wasn't even present on the GitHub Windows runner image. The fix probes `fwpuclnt.dll` first, then falls back to `iphlpapi.dll` (same symbols), and fails closed if neither loads. Again: invisible until real hardware ran the code.

## The pattern behind both bugs

Both bugs share a shape: the code was *plausibly correct* on the developer's machine and *actually broken* on the target. The only thing that caught them was a CI job executing escape and ETW tests on a real runner — the Windows job is now the authoritative cross-machine verification for the project.

That generalizes beyond Warden: if your cross-platform tool doesn't test on real OS runners, you don't actually support that OS. You support the fantasy of that OS.

## What the Windows backend looks like now

- **AppContainer token (LowBox):** denies filesystem, network, and environment access by default. The token is the hard boundary — no syscall crosses it.
- **WFP egress filters:** only the loopback proxy bridge is permitted outbound. DNS is denied by the token; TCP is denied by the filters.
- **ETW audit trail:** a private real-time trace session captures kernel file I/O scoped to the sandbox tree.
- **Job Objects:** wall-clock timeout, memory cap, kill-on-close — a runaway process dies with its entire tree.

Each layer requires its primitives to initialize, or the run is refused. That refusal is the feature: when the AppContainer backend can't initialize without admin rights, Warden exits instead of running your server unprotected.

## The lesson for tool builders

Test portability bit too: fixtures assuming POSIX paths and Linux `$PATH` failed on Windows for reasons that had nothing to do with the sandbox. OS-aware fixtures, real runners, and fail-closed defaults — in that order — are what make a cross-platform security claim believable.

The full story, with exact error messages and fixes, lives in `REMAINING_WORK.md` in the repo. If you're building cross-platform sandboxing, start there before you bind your first DLL.
