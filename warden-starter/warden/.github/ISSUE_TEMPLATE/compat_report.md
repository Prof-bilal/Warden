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
