# PostgreSQL MCP Server Policy

Sandbox the PostgreSQL MCP server. Database connections need **special handling**.

## What it needs

| Resource | Access | Details |
|---|---|---|
| Filesystem | Read | `.pgpass` and `.postgresql` credentials |
| Network | DB hosts | PostgreSQL server hostnames |
| Env | `PG*` variables | Connection details (host, port, password) |

## Complete Policy

```yaml
command: ["node", "@modelcontextprotocol/server-postgres"]

filesystem:
  read:
    - "./config/.pgpass"             # Password file
    - "./config/.postgresql"         # SSL/connection config
  write: []                          # It does not write DB server files

network:
  allow:
    - "localhost"                    # Local DB
    - "db.internal.example.com"      # Remote DB (put your own hostname)

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

- **Never** put `PGPASSWORD` in the policy, it comes from the shell
- Give the **exact path** to the `.pgpass` file (relative to the policy file)
- If you have multiple DB hosts, add **all** of them to `network.allow`
- Use `timeout_s: 600` for migrations

## Security

- **Never** put `PGPASSWORD` in the policy file
- Check the `.pgpass` file permissions (**600**)
- Use `require` for the SSL mode in production

## Troubleshooting

| Error | Fix |
|---|---|
| `connection refused` | Check `PGHOST` + `PGPORT` and add the host to `network.allow` |
| `password authentication failed` | Is `PGPASSWORD` set in the shell? |
| SSL errors | Set `PGSSLMODE` |

## Resources

- [PostgreSQL MCP Server](https://github.com/modelcontextprotocol/servers-archived/tree/main/src/postgres)
- [Schema Reference](schema.md)
- [Write a Policy](write-policy.md)
