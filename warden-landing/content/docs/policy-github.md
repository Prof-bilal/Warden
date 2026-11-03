# GitHub MCP Server Policy

Sandbox the GitHub MCP server. Both **REST** and **GraphQL** live on `api.github.com`.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read + Write | Cache for API responses, output for generated files |
| Network | `api.github.com` | REST API + GraphQL (same hostname) |
| Env | `GITHUB_TOKEN` | GitHub **personal access token** |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-github"]

filesystem:
  read:
    - "./cache"              # API response cache
  write:
    - "./output"             # Generated diffs, reports

network:
  allow:
    - "api.github.com"       # REST + GraphQL

env:
  allow:
    - "GITHUB_TOKEN"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-github

# Step 2: Set your token
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

# Step 3: Run sandboxed
warden run --policy policy-github.yaml
```

## Tips

- Grant `HOME` if the server needs to read git config or `.netrc`
- **Never** put `GITHUB_TOKEN` in the policy, give only the name; the value comes from the shell
- Both GraphQL and REST are on `api.github.com`, so **one entry is enough**
- Increase `timeout_s` for **long-running PR reviews**

## Troubleshooting

| Error | Fix |
|---|---|
| `permission denied` on network | Check that `api.github.com` is in `network.allow` |
| Token not found | Export `GITHUB_TOKEN` in the shell; the policy contains only the name |
| Rate limiting | Grant write access to the `cache` directory |

## Resources

- [GitHub MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/github)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
