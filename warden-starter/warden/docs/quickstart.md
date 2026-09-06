# Quickstart

Sandbox your first MCP server in five minutes. We'll use the `filesystem`
server fixture — local file I/O only, so the policy stays tiny.

## 1. Start from a fixture policy

Every tested server has an exact policy under `testdata/compat/`. Copy the
closest one:

```bash
cp testdata/compat/filesystem/policy.yaml ./policy.yaml
cat policy.yaml
```

```yaml
command: ["/usr/bin/node", "./server/dist/index.js"]
filesystem:
  read:
    - "./data"
  write: []
network:
  allow: []
env:
  allow: []
```

Relative paths resolve against the policy file's own directory. Point
`command` at your installed server entry point (absolute path), and set the
`read` grant to the directory you want the server to serve.

## 2. Run it sandboxed

```bash
warden run --policy ./policy.yaml
```

Your MCP client talks to the sandboxed server over stdio exactly as if it
were unsandboxed. Try reading a file outside the grant — it fails, and the
denial is logged rather than silent:

```bash
warden logs --tail 10
```

## 3. Don't guess — trace instead

For a server with no fixture, observe what it actually touches first:

```bash
warden trace -- /usr/bin/node ./server/dist/index.js
```

Exercise the server from your client, stop it, then generate a conservative
starter policy from the trace:

```bash
warden init
```

`init` only grants successful observed accesses and never converts a blocked
event into a grant. Review `policy.yaml`, tighten anything too broad
(`trace` runs unsandboxed by design), then `warden run` with it.

## 4. Iterate with the audit log

```bash
warden logs --tail 20   # what was allowed / blocked and why
warden logs --follow    # watch a running server live
```

Blocked attempts name the exact path or host, so widening a policy is
mechanical: add the grant, re-run. If a server needs something the schema
can't express (wildcard hosts, daemon sockets), check the
[Compatibility Matrix](compatibility.md#failures-classified) — it may be a
known gap rather than your mistake.

## Next steps

- [Schema Reference](schema.md) — every field, validation rules, enforcement notes
- [CLI Reference](cli.md) — `trace`, `init`, `logs`, `--approve`, gateway commands
- [FAQ](faq.md) — when something fails, start here
