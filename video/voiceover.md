# Warden — Marketing Video Voiceover Script

**Runtime:** ~5:20 · **Tone:** technical, confident, calm, direct · **Rule:** describe what is actually on screen. No hype. No security overclaims — Warden is a boundary, not a magic shield.

---

## Scene 01 — Hook (0:00–0:30)

You've probably run a command like this.

An MCP server starts. It connects. It's ready. Everything looks normal.

But the process behind that server runs as *you*. Your files. Your environment. Your SSH keys. Your credentials. Everything in your home directory is right there.

Your MCP doesn't need access to your entire computer.

So why give it that access?

## Scene 02 — The MCP Problem (0:30–1:15)

MCP — the Model Context Protocol — is why AI assistants can actually *do* things. It connects a client to servers that expose tools: filesystem access, network calls, environment lookups. That's what makes it useful.

Here's the part worth paying attention to. Most MCP servers run as plain processes on your machine, with your permissions. Not in a container. Not behind anything. Just... running.

So the same protocol that gives the model useful reach also gives the process behind it broad reach: project files, environment variables, SSH keys, credentials, your network.

Useful access and unrestricted access are not the same thing. MCP gives you the first. Nothing about the protocol requires the second.

## Scene 03 — Access Scenario (1:15–2:00)

Look at two requests from the same server.

The first one reads a project file. If you asked an assistant to summarize your code, that's exactly the access it needs. Reasonable.

The second one reads an SSH private key. Nothing about that request is unreasonable *for a process with full filesystem access* — which is the problem. The process is technically capable, so nothing stops the request.

Reading a project file can be necessary. Reading a private key is sensitive. The difference isn't the protocol. The difference is where the security boundary sits — and who controls it.

So, who controls that boundary today?

## Scene 04 — Introduce Warden (2:00–2:40)

This is Warden — an open-source sandbox runtime for MCP servers.

The idea is simple. The MCP server runs inside a boundary made of two things: a policy, and a sandbox. Outside that boundary, your sensitive resources stay where they are — invisible to the server.

Warden gives every MCP server a controlled environment with explicit permissions. Nothing more.

## Scene 05 — Policy (2:40–3:10)

The boundary is defined in one YAML file. This is real policy syntax — an actual example from the repository.

Which directories the server may read or write. Which hosts it may connect to. Which environment variables pass through. Even resource limits.

And here's the detail that matters: Warden is deny by default. Unlisted paths aren't "permission denied" — they return "not found". The sandbox can't even be probed.

## Scene 06 — Real Warden Demo (3:10–4:00)

Let's run it. This is the actual CLI.

`warden run` with a policy. Before launching anything, Warden prints exactly what the policy grants: the filesystem paths, the network host, the one environment variable. Then: sandbox active.

From here, your MCP client talks to the server over stdio exactly as if nothing had changed. Sandboxing is invisible to the protocol.

But watch the audit log. A read of a project file: allowed. A connection to the permitted API host: allowed. A read of your SSH key: blocked. A read of .env: blocked.

Every attempt — allowed or blocked — is recorded as structured JSON.

## Scene 07 — Architecture (4:00–4:30)

Under the hood, every request follows the same path: the MCP server asks, Warden's policy engine checks the policy, the sandbox backend enforces it at the OS level, and network egress flows through an allowlist proxy. Every decision lands in the audit log.

Allowed requests reach only the resources you listed. Everything else stops at the boundary.

And Warden fails closed: if the sandbox primitives can't be applied, it refuses to run your server at all. A plain-process fallback is never acceptable.

## Scene 08 — Platforms (4:30–4:50)

Warden uses the sandbox each OS already provides. On Linux, bubblewrap namespaces. On macOS, Seatbelt profiles. On Windows, AppContainer with filtering and Job Objects. And where no native backend fits, a Docker fallback — read-only root, no network.

No virtual machine, no daemon required on the native path.

## Scene 09 — Why It Matters (4:50–5:05)

Without a boundary, an MCP server reaches the host with broad permissions.

With Warden, the same server passes through policy and sandbox — and touches only what you granted.

Give tools the access they need. Not the access they don't.

## Scene 10 — Open Source (5:05–5:22)

Warden is open source, MIT licensed. Install it from npm, trace a server, generate a starter policy, and run it sandboxed.

The repository includes the full policy schema, the CLI reference, the security model and its honest limitations, and a reproducible test harness.

Clone it. Run your MCPs.

Don't blindly trust them.

---

*Warden — the sandbox runtime for MCP servers. github.com/Prof-bilal/Warden*
