# External Beta Program (M8)

The compatibility matrix proves Warden works for servers we chose and
policies we wrote. This program collects the signal we cannot generate
ourselves: **can people who didn't design the policy schema use it?**

## Goal

5–10 MCP server maintainers/users run their own servers under Warden and
report friction: where the schema was unclear, what had to be traced twice,
which access pattern had no honest grant, and where they gave up.

## How to join

1. Install Warden (see [Install](install.md)) on Linux with
   `bwrap` (or macOS with `sandbox-exec`; Docker fallback works anywhere).
2. Pick your server and trace it once:
   ```bash
   warden trace -- <your-server-command>
   warden init
   ```
3. Review the generated `policy.yaml`, tighten it, and run sandboxed:
   ```bash
   warden run --policy ./policy.yaml
   ```
4. File a compatibility report (below) — one per server.

No NDA, no private channel: reports are public GitHub issues so the whole
community sees the friction log.

## Filing a compatibility report

Use the **Compatibility report** issue template, reproduced in full below so you can copy it here.
Every report must include:

- server name + upstream repo/commit,
- host OS + backend (`--backend` value or `auto` resolution),
- verdict: works / works-with-hacks / blocked,
- the exact policy used (attach `policy.yaml`),
- for every failure, your best-guess class:
  - **Warden bug** — it should work but doesn't,
  - **schema gap** — no honest grant expresses what the server needs,
  - **inherent** — the server needs something sandboxing forbids by design
    (arbitrary filesystem/URL access, daemon sockets),
- the friction narrative: what was confusing, what you tried, where the
  docs/schema failed you.


### Report template

File this as a GitHub issue on the repo (reports are public so the whole community sees the friction log). Source: `.github/ISSUE_TEMPLATE/compat_report.md`.

```markdown
---
name: Compatibility report
about: Run your MCP server under Warden and report friction (M8 beta)
title: "[compat] <server-name>"
labels: compatibility
---

## Server

- Name / upstream repo + commit:
- Host OS + backend (`auto` resolution or explicit `--backend`):

## Verdict

<!-- works / works-with-hacks / blocked -->

## Policy

<!-- Attach the exact policy.yaml you ran with. Values are never stored in
     the policy — only env NAMES — so it is safe to paste. -->

```yaml
# paste policy.yaml here
```

## Access the server needs

<!-- What filesystem paths, network hosts, and env vars did `warden trace`
     + `warden init` reveal? What did you have to add by hand? -->

## Failures (one per bullet)

<!-- For each: what broke, and your best-guess class —
     warden-bug / schema-gap / inherent (see docs/compatibility.md). -->

-
-

## Friction narrative

<!-- What was confusing? What did you try? Where did the docs or schema
     fail you? This is the most valuable section — be specific. -->
```

## What maintainers do with reports

- Reproduce against the report's policy; add passing servers to
  `testdata/compat/` + `matrix.yaml` + `docs/compatibility.md`.
- Classify confirmed failures into the three classes above and update the
  [compatibility page](./compatibility.md#failures-classified).
- Fix most-used-servers-first: Warden bugs before schema gaps, schema gaps
  before polish. Each fix ships with a regression test, like the M8
  `Normalize` / `ResolveExecutable` fixes did.
- Close the loop: comment the outcome on the reporter's issue.

## Exit criteria

The beta has served its purpose when 5+ external reports are in, every
reported failure is classified, and the highest-impact gaps found are fixed
or explicitly scheduled. The matrix page then reflects reality, not just
fixtures written alongside the backends.
