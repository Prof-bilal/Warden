# Filesystem MCP Server Policy

Filesystem MCP server sirf local file I/O karta hai. Sabse simple policy.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read | Sirf wo directories jo serve karni hain |
| Network | None | Koi internet access nahi |
| Env | None | Koi variables nahi |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-filesystem"]

filesystem:
  read:
    - "./my-files"           # Sirf ye directory serve hogi
  write: []                  # Kuch bhi mat likho (read-only)

network:
  allow: []                  # No network

env:
  allow: []                  # No env vars

limits:
  memory_mb: 128
  timeout_s: 60
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-filesystem

# Step 2: Run sandboxed
warden run --policy policy-filesystem.yaml
```

## Tips

- Sirf woh directories grant karo jo sach mein chahiye
- `write: []` rakho agar sirf padhna hai
- Write grant dena ho toh alag directory do (output dir)
- Sabse secure server hai - no network, no env

## Variations

### Read + Write access

```yaml
filesystem:
  read:
    - "./data"
  write:
    - "./output"             # Output files yahan jayengi
```

### Multiple directories

```yaml
filesystem:
  read:
    - "./documents"
    - "./images"
    - "./config"
  write: []
```

## Troubleshooting

| Error | Fix |
|---|---|
| `permission denied` | Check `filesystem.read` mein directory hai |
| File not found | Path relative hai policy file ke folder ke against |
| Write failed | `filesystem.write` mein directory add karo |

## Resources

- [Filesystem MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
