# Dev.to / Hashnode Blog Post

## Title:
How to Sandbox Your MCP Servers in 5 Minutes

## Tags:
`security` `mcp` `ai` `claude` `nodejs` `tutorial`

---

## Body:

If you're using MCP servers with Claude Desktop, Cursor, or other AI tools, you're running third-party code with **full access** to your machine.

That GitHub MCP server? It can read your SSH keys.
That database MCP server? It can connect to any host.
That filesystem MCP server? It can delete everything.

**This is a security problem.** Here's how to fix it.

### The Problem

MCP servers run as plain Node.js or Python processes. No sandboxing. No permission model. If you install an MCP server from GitHub, it gets the same access as opening a terminal.

```bash
# This MCP server can read YOUR ENTIRE home directory
npx @modelcontextprotocol/server-github
```

### The Solution: Warden

[Warden](https://github.com/Prof-bilal/Warden) is a lightweight sandbox runtime for MCP servers. You write a simple YAML policy, and Warden runs the server with only those permissions.

**Install:**

```bash
npm install -g warden-sandbox-cli
```

**Create a policy:**

```yaml
# policy.yaml
command: ["node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
```

**Run sandboxed:**

```bash
warden run --policy policy.yaml -- node server.js
```

The server only sees:
- `./data` (read-only)
- `./output` (read-write)
- `api.github.com` (network)
- `GITHUB_TOKEN` (environment variable)

Everything else is **invisible**. Not "permission denied" — the paths literally don't exist.

### Auto-Generate Policies

Don't want to write policies manually? Warden can generate them for you:

```bash
# 1. Trace what the server accesses
warden trace -- node server.js

# 2. Generate a policy from the trace
warden init

# 3. Review and customize the generated policy
cat policy.yaml
```

### See What Happened

Every access attempt is logged:

```bash
warden logs

# Output:
# {"type":"file","action":"openat","resource":"/home/user/data/config.json","allowed":true}
# {"type":"network","action":"connect","resource":"api.github.com:443","allowed":true}
# {"type":"file","action":"openat","resource":"/home/user/.ssh/id_rsa","allowed":false,"reason":"not in policy"}
```

### Why Warden?

| Feature | Warden | Docker | Nothing |
|---------|--------|--------|---------|
| No root required | ✅ | ❌ | ✅ |
| OS-native | ✅ | ❌ | N/A |
| Deny by default | ✅ | Config | ❌ |
| Audit trail | ✅ | ❌ | ❌ |
| Auto-generate policies | ✅ | ❌ | ❌ |

### Try It Now

```bash
npm install -g warden-sandbox-cli
warden doctor  # Check if your system is ready
```

---

**GitHub:** https://github.com/Prof-bilal/Warden
**Docs:** https://prof-bilal.github.io/Warden/

If you found this useful, give Warden a star on GitHub! ⭐
