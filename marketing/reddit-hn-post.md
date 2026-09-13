# Reddit / Hacker News Post

## Title Options (pick one):

**Option A (Show HN):**
> Show HN: Warden – sandbox runtime for MCP servers

**Option B (Reddit r/mcp):**
> I built a sandbox for MCP servers so they can't access your whole system

**Option C (Reddit r/selfhosted):**
> Warden: sandbox your MCP servers with fine-grained filesystem/network/env control

---

## Post Body:

MCP servers (Claude Desktop, Cursor, etc.) run as plain processes with **full access** to your machine. That GitHub MCP server you just installed? It can read every file on your disk and connect to any host.

I built **Warden** to fix this. It's a lightweight sandbox runtime that runs MCP servers with only the permissions you explicitly grant.

### How it works

```bash
# Install
npm install -g warden-sandbox-cli

# Write a policy (what the server can access)
cat > policy.yaml << 'EOF'
command: ["node", "server.js"]
filesystem:
  read: ["./data"]
  write: ["./output"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN"]
EOF

# Run sandboxed
warden run --policy policy.yaml -- node server.js
```

The server only sees `./data` (read-only), `./output` (read-write), can only connect to `api.github.com`, and only gets the `GITHUB_TOKEN` env var. Everything else is invisible.

### What makes it different

- **Deny by default** — not "permission denied", the paths literally don't exist
- **OS-native** — uses bubblewrap (Linux), Seatbelt (macOS), AppContainer (Windows)
- **No Docker required** — native sandboxing first, Docker only as fallback
- **Audit trail** — logs every blocked access attempt
- **Auto-generate policies** — `warden trace` watches what a server accesses, `warden init` creates the policy

### Quick demo

```
$ warden run --policy policy.yaml -- node server.js
✓ Sandbox active

  Filesystem
    ✓ ./data (read)
    ✓ ./output (write)
    ✗ everything else (deny by default)

  Network
    ✓ api.github.com
    ✗ everything else
```

### Links

- GitHub: https://github.com/Prof-bilal/Warden
- npm: `npm install -g warden-sandbox-cli`
- Docs: https://prof-bilal.github.io/Warden/

Would love feedback on the approach. Is this something you'd use?
