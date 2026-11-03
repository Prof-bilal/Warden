# Slack MCP Server Policy

Sandbox the Slack MCP server. It needs access to both the **Slack API** and the **web UI**.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Write only | Cache for OAuth tokens, rate-limit state |
| Network | `slack.com` + `api.slack.com` | Web UI + API endpoints |
| Env | `SLACK_BOT_TOKEN`, `SLACK_TEAM_ID` | Bot authentication |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-slack"]

filesystem:
  read: []
  write:
    - "./cache"              # OAuth tokens, rate-limit state

network:
  allow:
    - "slack.com"            # Web UI
    - "api.slack.com"        # API

env:
  allow:
    - "SLACK_BOT_TOKEN"
    - "SLACK_TEAM_ID"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-slack

# Step 2: Set your tokens
export SLACK_BOT_TOKEN="xoxb-xxxxxxxxxxxx"
export SLACK_TEAM_ID="T0123456789"

# Step 3: Run sandboxed
warden run --policy policy-slack.yaml
```

## Tips

- Grant **only** `cache` under `write`, a **write grant also includes read**
- Do **not** add a separate `read: []` if `write` points to the same path
- `HOME` is needed by Node.js libraries
- `SLACK_TEAM_ID` looks optional but is **required**

## Troubleshooting

| Error | Fix |
|---|---|
| `token not found` | Export `SLACK_BOT_TOKEN` in the shell |
| Network blocked | List **both** `slack.com` and `api.slack.com` |
| OAuth errors | Grant write access to the `cache` directory |

## Resources

- [Slack MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/slack)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
