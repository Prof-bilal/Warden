---
title: "Warden vs Docker for MCP servers"
description: Docker and Warden solve different problems. When a container is enough, when a deny-by-default policy is the point, and how Warden uses Docker as a fallback.
section: compare
slug: warden-vs-docker
date: "2026-09-30"
answer: Docker isolates a process only as tightly as the run command you wrote. Warden starts from no filesystem, no hosts, and no environment, and refuses to launch an MCP server if that policy cannot be enforced. Docker is also one backend Warden can use. It is not a competing MCP policy language.
tldr:
  - "A default docker run still has outbound network and whatever volumes you mount."
  - "Warden's default is the opposite. Nothing is granted until the policy says so."
  - "Mounting the Docker socket into any sandbox gives away the host. Warden's compatibility matrix fails that server on purpose."
faqs:
  - question: Should I stop using Docker if I use Warden?
    answer: No. Use Docker when you want an image, a daemon, and a container boundary you configure yourself. Use Warden when you want one MCP policy enforced by an OS sandbox, with Docker as a fallback on hosts where the native backend cannot match the guarantee.
  - question: Is Docker less secure than Warden?
    answer: A carefully written docker run can be strict. An unexamined docker run is not. The difference is the default and the MCP-specific policy, not a magic property of either name.
related:
  - href: /compare/mcp-sandbox-options
    label: Other ways to isolate an MCP server
  - href: /guides/how-to-sandbox-mcp-server
    label: How to sandbox an MCP server
  - href: /blog/how-warden-enforces-policies
    label: Why a container is not the same as a grant list
  - href: /docs/compatibility
    label: Why the Docker MCP server fails the matrix
---

People reach for Docker because it is the isolation tool they already have. That is a good instinct and an incomplete policy.

## What each one is for

Docker packages a process and can hide the host filesystem behind the mounts you pass. You still decide `--network`, the user, the read-only root, and every volume. Forget one of those and the container is a convenient place to run code with broad network access and a copy of your project, or of your home directory.

Warden is a policy runner for local MCP servers. The file lists grants. Absence of a grant is the denial. The same file is compiled into bubblewrap, Seatbelt, or AppContainer, and the process starts only after that compilation succeeds.

| Question | Docker, as people usually run it | Warden |
| --- | --- | --- |
| What is the default filesystem? | The image, plus any volume you remembered | No host path until `filesystem` grants it |
| What is the default network? | Outbound allowed | No host until `network.allow` lists it |
| What happens to your shell environment? | Whatever you pass with `-e` or `--env-file` | Only names in `env.allow` |
| If the isolation setup fails? | Depends on the script you wrote | The server does not start |
| Does it understand MCP? | No. You wrap the command yourself | The CLI is built to sit on the stdio command |
| Can it use a container? | It is the container | Docker is an optional backend, not the policy |

## A fair case for Docker

Choose Docker when the problem is "this server has awkward dependencies and I want a known image." You can make that container strict: read-only root, no network or a user-defined network, one bind mount, dropped capabilities, a non-root user. At that point you have written a policy in `docker run` flags. Keep it in version control and review it like a policy.

Docker does not, by itself, tell you which MCP tool tried to read `~/.ssh`. You add that logging yourself.

## A fair case for Warden

Choose Warden when the problem is "this MCP server should keep working for the client, and should not inherit my account." One YAML file is the review surface. Denials land in a JSONL log. The native backend is the OS sandbox you would otherwise configure by hand. On a machine where that backend cannot meet the guarantee, Warden can fall back to Docker and says that it did, instead of pretending the weaker mode is the native one.

## The Docker socket

One result from the published matrix is easy to misread. The community Docker MCP server is a **fail**, classified as inherent, because it needs `/var/run/docker.sock`. Granting that socket is full control of the host's containers. Warden does not paper over that by mounting the socket inside the sandbox. Details are in the [compatibility matrix](/docs/compatibility) and in [How Warden enforces policies](/blog/how-warden-enforces-policies).
