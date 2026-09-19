# GitHub MCP Server Policy

GitHub MCP server ko sandbox karo. REST + GraphQL dono `api.github.com` pe hain.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read + Write | Cache for API responses, output for generated files |
| Network | `api.github.com` | REST API + GraphQL (same hostname) |
| Env | `GITHUB_TOKEN` | GitHub personal access token |

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

- `HOME` do agar server ko git config ya `.netrc` padhna ho
- `GITHUB_TOKEN` policy mein mat likho - sirf name do, value shell se aayegi
- GraphQL aur REST dono `api.github.com` pe hain, ek entry sufficient hai
- Long-running PR reviews ke liye `timeout_s` badha do

## Troubleshooting

| Error | Fix |
|---|---|
| `permission denied` on network | Check `network.allow` mein `api.github.com` hai |
| Token not found | `GITHUB_TOKEN` export karo shell mein, policy mein sirf name hai |
| Rate limiting | `cache` directory ka write grant do |

## Resources

- [GitHub MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/github)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
