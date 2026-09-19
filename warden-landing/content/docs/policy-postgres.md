# PostgreSQL MCP Server Policy

PostgreSQL MCP server ko sandbox karo. Database connections ke liye special handling chahiye.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read | `.pgpass` aur `.postgresql` credentials |
| Network | DB hosts | PostgreSQL server hostnames |
| Env | `PG*` variables | Connection details (host, port, password) |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-postgres"]

filesystem:
  read:
    - "./config/.pgpass"             # Password file
    - "./config/.postgresql"         # SSL/connection config
  write: []                          # DB server files nahi likhta

network:
  allow:
    - "localhost"                    # Local DB
    - "db.internal.example.com"      # Remote DB (apna hostname daalo)

env:
  allow:
    - "PGHOST"
    - "PGPORT"
    - "PGUSER"
    - "PGPASSWORD"
    - "PGSSLMODE"
    - "HOME"

limits:
  memory_mb: 512
  timeout_s: 600
```

## How to run

```bash
# Step 1: Install the server
npm install -g @modelcontextprotocol/server-postgres

# Step 2: Set connection variables
export PGHOST="localhost"
export PGPORT="5432"
export PGUSER="myuser"
export PGPASSWORD="mypassword"
export PGSSLMODE="prefer"

# Step 3: Run sandboxed
warden run --policy policy-postgres.yaml
```

## Tips

- `PGPASSWORD` policy mein mat likho - shell se aayegi
- `.pgpass` file ka exact path do (relative to policy file)
- Multiple DB hosts ho toh sab `network.allow` mein daalo
- `timeout_s: 600` rakho migrations ke liye

## Security

- `PGPASSWORD` kabhi policy file mein mat likho
- `.pgpass` file permissions check karo (600)
- SSL mode production mein `require` rakho

## Troubleshooting

| Error | Fix |
|---|---|
| `connection refused` | `PGHOST` + `PGPORT` check karo, `network.allow` mein add karo |
| `password authentication failed` | `PGPASSWORD` shell mein set hai? |
| SSL errors | `PGSSLMODE` set karo |

## Resources

- [PostgreSQL MCP Server](https://github.com/modelcontextprotocol/servers/tree/main/src/postgres)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
