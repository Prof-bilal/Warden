---
title: How to sandbox an MCP server
description: How to put a local MCP server in an OS sandbox, what the policy must grant, and what to do when the sandbox cannot start.
section: guides
slug: how-to-sandbox-mcp-server
date: "2026-09-30"
answer: Sandbox an MCP server by starting it inside an OS mechanism that hides ungranted files, blocks unlisted hosts, and filters environment variables. Build that sandbox first. If it cannot be built, do not start the server.
tldr:
  - "Decide the directory, hosts, and env names the server needs before you launch it."
  - "Enforce those grants with bubblewrap, Seatbelt, AppContainer, or a tightly configured container."
  - "A sandbox that fails open is not a sandbox. Abort the launch."
faqs:
  - question: Can I sandbox an MCP server with file permissions alone?
    answer: 'Mode bits return "permission denied," which tells the process the path exists. A bind-mount or equivalent that omits the path means the directory is not there to probe. Prefer that for untrusted servers.'
  - question: Do I have to write a different policy per operating system?
    answer: Not if the tool compiles one policy into the native mechanism. The grants should mean the same thing on Linux, macOS, and Windows even though the kernel primitives differ.
related:
  - href: /guides/mcp-server-security
    label: How to securely run an MCP server
  - href: /compare/mcp-sandbox-options
    label: Ways to isolate an MCP server
  - href: /compare/warden-vs-docker
    label: Warden and Docker, side by side
  - href: /use-cases/mcp-server-filesystem-isolation
    label: Isolating an MCP server from the rest of the disk
---

Sandboxing here means the server process cannot name resources you did not grant. It does not mean a prompt that says "be careful," and it does not mean chmod on a single file while the rest of your home directory stays mounted.

## What you are wrapping

The client still speaks MCP. You change how the server process is created. A wrapper sits in front:

1. Read a policy that lists the command plus filesystem, network, and environment grants.
2. Build the OS sandbox from that list.
3. Spawn the server inside it, with stdio passed back to the client.
4. Record allows and denials.

The client configuration points at the wrapper instead of at `npx` directly. The protocol messages do not change.

## Grants to write down first

Use the smallest set the server's own README claims:

- **Read** paths it must see, usually one project folder.
- **Write** paths it must modify. A write grant is not a read grant on your home directory.
- **Hosts** it calls, such as `api.github.com`. Leave the list empty if the server is local-only.
- **Env names** it reads, such as `GITHUB_TOKEN`. Values stay in your shell. The policy stores names.
- **Limits** for memory and runtime so a loop cannot take the machine.

Then run the server once and read the denial log. Add a grant only when a denial names a resource you accept. That is the opposite of starting from "allow everything" and hoping you remember what to remove.

## OS mechanisms that can do this

| Host | Mechanism people actually use | What you must get right |
| --- | --- | --- |
| Linux | [bubblewrap](https://github.com/containers/bubblewrap) namespaces and bind mounts | Only granted paths exist in the mount namespace. Network has no default route. |
| macOS | Seatbelt (`sandbox-exec`) profiles | Operations absent from the profile are denied by the kernel. |
| Windows | AppContainer, plus egress filters | The token and the filters both have to initialize. |
| Any | A container runtime | The run spec, not the word "Docker," is the policy. Default networking and a home-directory mount undo the point. |

Hand-writing those profiles is reasonable if you already operate them. It is also how flags get dropped on the one machine where the binary is missing.

## Where Warden fits

Warden is a local CLI that performs the four wrapper steps. `warden run --policy policy.yaml -- <command>` compiles the same YAML into bubblewrap, Seatbelt, or AppContainer, and can use Docker when the native backend cannot provide the guarantee. Startup is fail-closed: no backend, no process. The enforcement path is described in [How Warden enforces policies](/blog/how-warden-enforces-policies), and the three backends are compared in [bubblewrap, Seatbelt, and AppContainer](/blog/sandboxing-backends-bubblewrap-seatbelt-appcontainer).

A first policy and the install command are in the [quickstart](/docs/quickstart).
