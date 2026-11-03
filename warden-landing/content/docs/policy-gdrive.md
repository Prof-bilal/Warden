# Google Drive MCP Server Policy

Sandbox the Google Drive MCP server. It needs access to **multiple Google APIs**.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read + Write | OAuth credentials file + cache |
| Network | 3 Google API hosts | `www.googleapis.com`, `oauth2.googleapis.com`, `drive.google.com` |
| Env | Google credentials | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-gdrive"]

filesystem:
  read:
    - "./config/credentials.json"     # OAuth client credentials
  write:
    - "./cache"                       # API response cache

network:
  allow:
    - "www.googleapis.com"            # Google APIs
    - "oauth2.googleapis.com"         # OAuth flow
    - "drive.google.com"              # Drive web UI

env:
  allow:
    - "GOOGLE_CLIENT_ID"
    - "GOOGLE_CLIENT_SECRET"
    - "HOME"

limits:
  memory_mb: 256
  timeout_s: 300
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-gdrive

# Step 2: Set credentials
export GOOGLE_CLIENT_ID="your-client-id"
export GOOGLE_CLIENT_SECRET="your-client-secret"

# Step 3: Run sandboxed
warden run --policy policy-gdrive.yaml
```

## Tips

- Give the **exact path** to `credentials.json`
- **3 Google API hosts** are needed, add all of them to `network.allow`
- The OAuth flow happens on `oauth2.googleapis.com`
- The `cache` directory is needed for API responses

## Security

- Grant `credentials.json` **read-only** access (only `read`)
- **Never** put `GOOGLE_CLIENT_SECRET` in the policy
- The `cache` directory may store **sensitive data**

## Troubleshooting

| Error | Fix |
|---|---|
| `OAuth error` | Is `oauth2.googleapis.com` in `network.allow`? |
| `credentials not found` | Check the path to `credentials.json` |
| `API error` | Is `www.googleapis.com` in `network.allow`? |

## Resources

- [Google Drive MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/gdrive)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
