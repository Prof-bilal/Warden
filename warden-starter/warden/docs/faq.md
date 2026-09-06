# FAQ / Troubleshooting

## `warden run` refuses to start

### "no sandbox backend available … refusing to run unsandboxed"

No usable backend on this host. On Linux install `bwrap`
([Install](install.md#per-os-prerequisites)); on macOS make sure
`sandbox-exec` exists; anywhere else, start the Docker daemon. Warden never
falls back to an unsandboxed run — this error is the product working as
designed.

### "command … is not an absolute path"

The executable must resolve to an absolute path so the backend can
bind-mount its directory. Bare names (`npx`, `uvx`, `node`) are resolved via
your `PATH` automatically — if you see this error, the name isn't on `PATH`
in the shell launching Warden. Check with `command -v npx`, or put the
absolute path in the policy's `command:`.

### "executable …: no such file or directory"

The absolute path doesn't exist *on this host*. Common cause: a policy
written on another machine (or a fixture copied verbatim) whose `command:`
points at someone else's install. Point it at your entry point.

## My server starts, then something is denied

### How do I see what was blocked?

```bash
warden logs --tail 20
```

Every event names the type (`file`/`network`), action, resource, and
allowed/blocked reason. Blocked file paths and hostnames map directly onto
the grant you need to add.

### The policy looks right but access still fails

- **Same path in `read` and `write`?** `Load` coalesces the overlap to a
  single write grant — harmless, but check `warden logs` for the real
  denial, which is usually a *parent* directory that was never granted.
- **SQLite writes failing?** Grant `write` on the database *directory*, not
  the db file — WAL and journal sidecars live next to it.
- **Child processes failing?** The sandbox covers the whole process tree,
  including grandchildren. The denial still appears in the log with the
  exact resource.

### Can I approve access interactively instead of editing YAML?

Yes: `warden run --policy policy.yaml --approve -- <cmd>`. Network grants
apply live; filesystem grants are saved to the policy file and the server
restarts (bind mounts are fixed at spawn). Approval needs a terminal and
fails closed without one.

## Security footguns

### I passed `HOME` and now the server can read `~/.ssh`?

Expected — and your policy bug, not Warden's. `HOME` passthrough plus a
`filesystem.read` grant covering `~/.ssh` (or a parent of it) exposes your
keys. Grant only the exact credential paths the server needs
(`./config/.pgpass`, not `~`), and re-read the
[Security Review](security.md#5-credential-exposure) before shipping a
policy that passes `HOME`.

### `warden trace` shows my secrets — is that logged?

`trace` runs unsandboxed by design, so observed env values can appear in the
trace log. `init` only writes variable *names* into the policy — values stay
in your shell — but treat raw trace logs as sensitive and delete them after
generating the policy.

## Things Warden can't do (by design or yet)

### Wildcard hosts (`*.example.com`)

Not supported — `network.allow` takes bare hostnames or IP literals only,
and adding a wildcard is rejected at validation. Browser-like servers that
visit arbitrary domains (Playwright) therefore have no honest policy. This
is a tracked [schema gap](compatibility.md#failures-classified), locked in
by regression test so it can't change silently.

### Daemon sockets (`/var/run/docker.sock`)

There is no unix-socket grant type, and granting a daemon socket would hand
over host control anyway. Servers that need one (docker-mcp) are
[inherently incompatible](compatibility.md#failures-classified).

### "Will this work with my server?"

Check the [Compatibility Matrix](compatibility.md) first. If your server
isn't listed, trace it and [file a compatibility report](beta.md#filing-a-compatibility-report) —
that's exactly the signal the beta program collects.
