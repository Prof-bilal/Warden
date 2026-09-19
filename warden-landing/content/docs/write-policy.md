# Write a Policy

Policy.yaml likhna bahut simple hai. Sirf 3 cheezein batani hain:
**kya padh sakta hai**, **kahan network kar sakta hai**, aur **kaunse secrets mil sakte hain**.

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

Bas! Jo cheez list mein nahi hai, wo **exist hi nahi karti** sandbox ke andar.

---

## Step 1: `command` - Server kaise start hoga

```yaml
command: ["/usr/bin/node", "./server/dist/index.js"]
```

- Pehla element: executable ka **absolute path**
- Baaki elements: arguments

**Tip:** Bare name (`node`, `npx`) bhi kaam karta hai PATH se resolve hota hai.
Lekin checked-in policy mein absolute path likho.

---

## Step 2: `filesystem` - Kya padh/likh sakta hai

```yaml
filesystem:
  read: ["./data", "./config"]     # Sirf ye directories padh sakta hai
  write: ["./output"]              # Sirf ye directory mein likh sakta hai
```

**Rules:**
- Relative paths policy file ke folder ke against resolve hote hain
- `read` = sirf padh sakta hai (read-only mount)
- `write` = padh + likh dono kar sakta hai
- Jo path list mein nahi hai, wo **invisible** hai (permission denied nahi, literally exists nahi)

**Common patterns:**

| Use Case | read | write |
|---|---|---|
| Cache only | `[]` | `["./cache"]` |
| Read-only data | `["./data"]` | `[]` |
| Read + write | `["./data"]` | `["./output"]` |

---

## Step 3: `network` - Kahan internet kar sakta hai

```yaml
network:
  allow:
    - "api.github.com"
    - "api.slack.com"
```

**Rules:**
- Sirf hostnames ya IP addresses
- Ports part of grant nahi hain (proxy forward karta hai)
- DNS resolution blocked hai allowed hosts ke liye
- `[]` = koi network access nahi

---

## Step 4: `env` - Kaunse secrets mil sakte hain

```yaml
env:
  allow:
    - "GITHUB_TOKEN"
    - "NODE_ENV"
    - "HOME"
```

**Rules:**
- Sirf **names** likho, values policy mein kabhi nahi hoti
- Values parent shell se aati hain runtime pe
- `[]` = empty environment (koi variable nahi)

**Security tip:** `HOME` mat do agar `~/.ssh` bhi grant kar rahe ho.

---

## Step 5: `limits` - Resource caps

```yaml
limits:
  memory_mb: 512       # Max memory (MB)
  timeout_s: 300       # Max time (seconds)
```

- 0 ya omit = no limit
- Linux pe enforced hai (process kill hota hai limit breach pe)

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

## Shortcut: Auto-Generate Policy

Policy guess mat karo. Pehle trace karo:

```bash
# Step 1: Server ko run karo, access record hoga
warden trace -- node server.js

# Step 2: Trace se policy generate karo
warden init
```

`init` sirf successful accesses ko grant karta hai. Generated policy review karo,
aur tight karo jo broad hai.

---

## Common Mistakes

| Mistake | Fix |
|---|---|
| Same path in read AND write | Write grant subsumes read. Sirf `write` mein daalo |
| `HOME` + broad filesystem | `HOME` do + `~/.ssh` grant = SSH keys leak |
| Wildcard hosts (`*`) | Supported nahi. Har host alag list karo |
| Bare command name | Absolute path do ya PATH pe rely karo |
| No limits set | Memory/timeout set karo (especially CI mein) |

---

## Next Steps

- [Server Guides](policy-github.md) - Har MCP server ka detailed guide
- [Compatibility Matrix](compatibility.md) - 18 tested servers with exact policies
- [Schema Reference](schema.md) - Har field ka detailed explanation
