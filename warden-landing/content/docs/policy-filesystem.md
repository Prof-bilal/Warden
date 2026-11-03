# Filesystem MCP Server Policy

The Filesystem MCP server only performs **local file I/O**. It has the **simplest** policy.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read | Only the directories it should serve |
| Network | None | **No internet access** |
| Env | None | **No variables** |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-filesystem"]

filesystem:
  read:
    - "./my-files"           # Only this directory is served
  write: []                  # Write nothing (read-only)

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

- Grant **only** the directories you actually need
- Use `write: []` if you only need to **read**
- If it needs write access, give it a **separate** directory (the output dir)
- This is the **most secure** server, **no network, no env**

## Variations

### Read + Write access

```yaml
filesystem:
  read:
    - "./data"
  write:
    - "./output"             # Output files go here
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
| `permission denied` | Check that the directory is listed in `filesystem.read` |
| File not found | The path is **relative** to the policy file's folder |
| Write failed | Add the directory to `filesystem.write` |

## Resources

- [Filesystem MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/filesystem)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
