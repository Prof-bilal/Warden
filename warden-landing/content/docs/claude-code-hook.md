# Screen Claude Code tool requests

Create a dedicated Warden policy for the project and install the
PreToolUse hook from the project directory:

```bash
warden hook install --policy ./warden.yaml
```

The project default is `.claude/settings.local.json`. Use `--scope user` to
edit `~/.claude/settings.json` or `--config <file>` to select a settings file.
The hook compares supported shell commands against literal policy words and
checks supported file tools against the policy’s paths and deny rules.
An allowed request still goes through Claude Code’s normal permissions.

The hook is advisory. It only sees the tool requests that Claude Code sends
to PreToolUse. It runs no OS sandbox and cannot enforce policy inside a
separately launched process. Use [warden run](cli.md) when you
need OS-level containment.

```bash
warden hook uninstall
```

Use the same scope or settings path you used for installation. Warden restores
the original settings when they still match the installed copy. If it reports
drift, review the `PreToolUse` groups and remove the entry that calls
`warden hook check`.
