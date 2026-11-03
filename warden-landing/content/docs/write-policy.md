# Write a Policy

Writing `policy.yaml` is **simple**. You only need to declare **three things**:
**what it can read**, **where it can reach on the network**, and **which secrets it can receive**.

## Basic Structure

```yaml
command: ["node", "server.js"]

filesystem:
  read: ["./data"]
  write: ["./output"]

network:
  allow: ["api.example.com"]

env:
  allow: ["API_KEY"]
```

That's it. Anything **not** in the list **does not exist** inside the sandbox.

---

## Step 1: `command` - How the server starts

```yaml
command: ["/usr/bin/node", "./server/dist/index.js"]
```

- First element: the **absolute path** to the executable
- Remaining elements: arguments

**Tip:** A bare name (`node`, `npx`) also works and resolves through `PATH`.
But in a **checked-in policy**, write the **absolute path**.

---

## Step 2: `filesystem` - What it can read and write

```yaml
filesystem:
  read: ["./data", "./config"]     # Only these directories are readable
  write: ["./output"]              # Only this directory is writable
```

**Rules:**

- Relative paths resolve against the **policy file's directory**
- `read` = **read-only** mount
- `write` = **read and write**
- Any path not in the list is **invisible** (not "permission denied", it literally **does not exist**)

**Common patterns:**

| Use Case | read | write |
|---|---|---|
| Cache only | `[]` | `["./cache"]` |
| Read-only data | `["./data"]` | `[]` |
| Read + write | `["./data"]` | `["./output"]` |

---

## Step 3: `network` - Where it can connect

```yaml
network:
  allow:
    - "api.github.com"
    - "api.slack.com"
```

**Rules:**

- **Hostnames** or **IP addresses** only
- Ports are **not** part of the grant (the proxy forwards them)
- **DNS resolution is blocked** for hosts that are not allowed
- `[]` = **no network access**

---

## Step 4: `env` - Which secrets it can receive

```yaml
env:
  allow:
    - "GITHUB_TOKEN"
    - "NODE_ENV"
    - "HOME"
```

**Rules:**

- Write **names** only, values are **never** stored in the policy
- Values come from the **parent shell** at runtime
- `[]` = **empty environment** (no variables)

**Security tip:** Do **not** grant `HOME` if you are also granting `~/.ssh`.

---

## Step 5: `limits` - Resource caps

```yaml
limits:
  memory_mb: 512       # Max memory (MB)
  timeout_s: 300       # Max time (seconds)
```

- `0` or omitted = **no limit**
- Enforced on **Linux** (the process is **killed** when a limit is breached)

---

## Complete Examples

### GitHub MCP Server

```yaml
command: ["node", "@modelcontextprotocol/server-github"]

filesystem:
  read: ["./cache"]
  write: ["./output"]

network:
  allow: ["api.github.com"]

env:
  allow: ["GITHUB_TOKEN", "HOME"]

limits:
  memory_mb: 256
  timeout_s: 300
```

### Slack MCP Server

```yaml
command: ["node", "@modelcontextprotocol/server-slack"]

filesystem:
  read: []
  write: ["./cache"]

network:
  allow: ["slack.com", "api.slack.com"]

env:
  allow: ["SLACK_BOT_TOKEN", "SLACK_TEAM_ID", "HOME"]

limits:
  memory_mb: 256
  timeout_s: 300
```

### Filesystem MCP Server

```yaml
command: ["node", "@modelcontextprotocol/server-filesystem"]

filesystem:
  read: ["./my-files"]
  write: []

network:
  allow: []

env:
  allow: []

limits:
  memory_mb: 128
  timeout_s: 60
```

---

## Shortcut: Auto-Generate a Policy

**Do not guess** the policy. **Trace first**:

```bash
# Step 1: Run the server and record its access
warden trace -- node server.js

# Step 2: Generate a policy from the trace
warden init
```

`init` grants only **successful** observed accesses. **Review** the generated
policy and **tighten** anything that is too broad.

---

## Common Mistakes

| Mistake | Fix |
|---|---|
| Same path in read AND write | A **write grant subsumes read**. Put it only under `write` |
| `HOME` + broad filesystem | Granting `HOME` **and** `~/.ssh` leaks SSH keys |
| Wildcard hosts (`*`) | **Not supported**. List each host separately |
| Bare command name | Use an **absolute path**, or rely on `PATH` |
| No limits set | Set **memory/timeout** (especially in CI) |

---

## Next Steps

- [Server Guides](policy-github.md), a detailed guide for each MCP server
- [Compatibility Matrix](compatibility.md), 18 tested servers with exact policies
- [Schema Reference](schema.md), a detailed explanation of every field
