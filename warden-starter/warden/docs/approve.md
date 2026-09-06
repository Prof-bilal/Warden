# Interactive Approval Mode

Instead of only hard-failing on the first out-of-policy access, `warden run
--approve` prompts you on the terminal and lets you decide per resource.

```bash
warden run --policy policy.yaml --approve -- /usr/bin/node server.js
warden run --policy policy.yaml --approve --approve-timeout 2m -- /usr/bin/node server.js
```

## What happens per resource type

| Access | Prompt offers | Effect when approved |
|---|---|---|
| Network host | allow once / for this session / and save to policy / deny | **Live**: the proxy allows the host immediately (session) and optionally appends it to `network.allow` in the policy file (save) |
| Filesystem path | save + restart now / save for next run / deny | **Saved**: the grant is appended to the policy file (`filesystem.read` on the path, or `filesystem.write` on the parent dir for mutating access — exactly what `warden init` would generate). Bind mounts are fixed at spawn, so a restart is required; Warden respawns the server automatically on "restart now" |

You are asked **once per resource per run**: session/session-save/deny answers
are remembered, so a chatty server cannot prompt-loop you. "Allow once"
(network only) deliberately prompts again next time.

## Rules that keep it safe

- **Prompts use the controlling terminal (`/dev/tty`), never stdio.**
  The MCP client's stdin/stdout stream stays protocol-clean; stderr only
  carries one-line status notes ("saved grant …", "restarting …").
- **No terminal, no answer, timeout, or save failure ⇒ deny (fail closed).**
  `--approve` with no interactive terminal refuses to start at all, so it
  can never silently downgrade to deny-everything under an MCP client.
- **Filesystem approvals always persist.** A restart reloads policy from
  disk, so a memory-only filesystem grant would be a lie — there is no
  "session-only" filesystem option.
- **Every decision is audit-logged** (`type: approval` in `warden logs`),
  including denials and timeouts.
- **Prompts are selective, not noisy.** A filesystem prompt fires only for a
  failed access that maps to a real grant (absolute, non-runtime path,
  outside current grants) naming a path that **exists on the host** —
  routine probes for optional files never ask. Runtime paths (`/usr`,
  `/proc`, `/tmp`, …) and the server's own executable never ask.
- **Write grants into the runtime base (`/usr`, `/lib64`, …) are refused**:
  no backend could honor them.
- Restarts are capped (10 per invocation) so a misbehaving approval loop
  cannot respawn forever; the policy is reloaded from disk on each restart.

## Backend support

- **Linux**: full support (network live + filesystem via live `strace` watch).
- **macOS Seatbelt, Docker, Windows**: network approval only. Those
  backends have no live filesystem signal (Seatbelt reports EPERM without
  structured events, ETW/container tracing is post-hoc), so filesystem
  stays hard-deny — use `warden trace` + `warden init` to widen the policy.

## Non-goals

- Environment variables are granted at spawn, never "requested" at runtime,
  so they are out of scope: no prompts, ever.
- Approval is a local-developer UX (a human at a terminal), not a
  multi-user gateway feature.
