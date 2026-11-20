# Migrate policies and MCP launchers to Warden 2.0

Warden still accepts schema v1 and reports a migration warning when you
validate or run an explicitly versioned v1 policy. The `warden add` command
still works and emits a deprecation warning. Use `warden run` to launch a
reviewed server or `warden wrap` to update an existing client launcher.

Preview from your project directory to see version changes, policy digests and
client launchers that need attention:

```bash
warden migrate ./warden.yaml
```

With no arguments, Warden checks for `warden.yaml` in your project and inspects
client configs at known project and user paths. For a config stored elsewhere,
specify its client and path:

```bash
warden migrate --client cursor --config ./.cursor/mcp.json ./warden.yaml
```

Add `--write` before the policy paths to apply the previewed changes. Warden
verifies that the effective policy stays the same when rewriting its schema
version. It updates matching SHA-256 pins in inspected launchers and preserves
their undo records. It creates an owner-only `.bak` beside each changed file.
Config backups may contain credentials, so keep them private.

## Findings that require review

- **Digest changed since review.** Compare the current policy with the copy you
  approved before replacing its pin. For a relative policy path, verify the
  working directory used by the client and identify the file explicitly.
- **Empty MCP tool allowlist.** An empty or omitted list denies every tool call.
  List each tool name you want to allow.
- **Legacy stdio proxy.** Review how the upstream is isolated. Warden requires
  explicit `--unsafe-unconfined` for a direct stdio upstream. You can launch
  under an OS sandbox with a reviewed policy instead.
- **Unsupported policy keys.** Compare the field with the [policy schema](schema.md)
  and validate the resulting policy before migrating.

Exit code `0` means no work remains. Code `3` means the preview found pending
changes or manual findings remain. Code `2` signals a usage or write error.
