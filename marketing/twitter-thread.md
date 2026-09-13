# Twitter/X Thread

## Tweet 1 (Hook)
MCP servers run with full access to your machine.

Every file. Every network host. Every env var.

That random GitHub MCP server you just installed? It can read your SSH keys.

I built something to fix this 🧵

## Tweet 2 (Problem)
The problem:

MCP servers (Claude Desktop, Cursor, etc.) run as plain Node/Python processes. No sandboxing. No permission model.

If you install an MCP server from GitHub, it gets the same access as your terminal.

That's... not great.

## Tweet 3 (Solution)
Meet Warden 👇

A lightweight sandbox runtime for MCP servers.

You write a simple YAML policy:
- Which files it can read/write
- Which hosts it can connect to
- Which env vars it receives

Everything else is invisible. Not "permission denied" — literally doesn't exist.

## Tweet 4 (How it works)
How it works:

```bash
npm install -g warden-sandbox-cli

warden run --policy policy.yaml -- node server.js
```

The server runs inside OS-native sandboxing:
- Linux: bubblewrap
- macOS: Seatbelt
- Windows: AppContainer

No Docker required.

## Tweet 5 (Auto-generate)
Don't want to write policies by hand?

```bash
warden trace -- node server.js
warden init
```

Warden watches what the server accesses, then generates a policy automatically.

Run it once to see what it needs. Then lock it down.

## Tweet 6 (Audit)
Every access attempt is logged.

```bash
warden logs
```

You see exactly what the server tried to do — including blocked attempts.

No more guessing what an MCP server is doing behind your back.

## Tweet 7 (CTA)
Open source, MIT licensed, works on Linux/macOS/Windows.

```bash
npm install -g warden-sandbox-cli
```

GitHub: https://github.com/Prof-bilal/Warden

If you use MCP servers, this should be in your toolkit.

Star it if you find it useful ⭐
