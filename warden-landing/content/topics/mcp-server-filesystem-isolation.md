---
title: How to stop an MCP server from reading your filesystem
description: Why an MCP filesystem server can see your home directory, and how bind mounts and grants limit it to one folder.
section: use-cases
slug: mcp-server-filesystem-isolation
date: "2026-09-30"
answer: An MCP server can read your filesystem because it runs as you. Stop that by mounting only the directory it should see into its sandbox. A permission-denied error is weaker, because the process still learns the other path exists.
tldr:
  - "The filesystem MCP server is not special. Any local server can open files your user can open."
  - "Grant one directory. Leave the rest unmounted so it is not visible."
  - "On Linux, Warden bind-mounts granted paths into a bubblewrap namespace. The integration check expects an ungranted sibling to be NOT_VISIBLE."
faqs:
  - question: I granted the repo. Why can it still read ~/.ssh?
    answer: The grant is probably wider than the repo. A read on your home directory, or HOME passed through with a parent path that covers .ssh, includes the keys. Grant the repository path only.
  - question: Is chmod 700 on .ssh enough?
    answer: It helps against other users on a shared machine. It does not help against a process running as you. The server is you, unless a sandbox changes that.
related:
  - href: /guides/mcp-server-permissions
    label: What permissions an MCP server should have
  - href: /blog/deny-by-default-invisible-not-denied
    label: Why invisible files are stronger than permission denied
  - href: /use-cases/run-untrusted-mcp-servers
    label: Running an MCP server you do not trust
  - href: /docs/examples
    label: Example policies, including the filesystem server
---

The filesystem server is the clearest demo, and it is not the only server with the problem. A Git server, a search server, or a package you skimmed can call `open` on any path. MCP tool arguments are just strings the process interprets.

## What people usually do

They configure the server with a root of `/` or `$HOME` because the README says "set the allowed directory," and the allowed directory is enforced by the server. A bug, a path traversal, or a second tool in the same process skips that check. The operating system still allows the read.

Server-side allowlists are useful. They are not the boundary. The boundary is what the kernel is willing to resolve.

## What isolation looks like

The sandboxed process has a filesystem made of the paths you granted. Listing a sibling directory does not return "permission denied." It returns nothing, because that directory was not mounted.

On Linux, bubblewrap builds that view with bind mounts in a new mount namespace. macOS uses a Seatbelt profile. Windows uses an AppContainer token. The user-facing rule is the same: if the path is not in the policy, it is not part of the process's world.

A minimal filesystem policy:

```yaml
filesystem:
  read: ["./notes"]
  write: ["./notes/out"]
network:
  allow: []
env:
  allow: []
```

Reads outside `./notes` fail closed. Writes outside `./notes/out` fail closed. There is no network, so a file that does get read cannot be posted to an arbitrary host by this process.

## Evidence, and its limit

Warden's integration test for this behavior expects `ls` on an ungranted sibling to print `NOT_VISIBLE`. The Linux proof harness reported 8 of 8 steps passing, including secret reads and unlisted paths. Both are described in [AI tried to escape my sandbox](/blog/ai-tried-to-escape-my-sandbox) and [Deny-by-default: why permission denied is not enough](/blog/deny-by-default-invisible-not-denied).

That is evidence about those tests. It is not a proof that every symlink trick on every OS version has been enumerated. The Windows write-up includes symlink and UNC attempts in the five scenarios that were blocked, and it says so as a tested set.

## Where Warden fits

Put the filesystem server, or any other local server, under a policy like the one above and launch it with `warden run`. The [example policies](/docs/examples) include a filesystem server you can copy. If the server errors, the log names the path. Add it only if you would be willing to see that path in a chat transcript.
