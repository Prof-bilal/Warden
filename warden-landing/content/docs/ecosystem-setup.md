# Policy packs and reversible MCP setup

These commands are implemented in this development checkout. Build this checkout
first; older published binaries may not include them. The [setup wizard](/setup)
generates local commands. It does not read your computer or accept credentials.

## One command instead of five

If the server you want is a catalog pack (`warden packs list` — filesystem,
git, memory, github, brave, context7, slack, notion) or **any npm MCP
package**, `warden add` chains the whole flow for you: sandboxed runtime
preparation, deny-by-default policy generation, preview + confirmation,
client config registration, and an undo record.

```bash
warden add slack to claude-desktop
warden add notion to claude-desktop
warden add @scope/mcp-server@1.2.3 to claude-code \
    --allow-host api.example.com --allow-env SCOPE_KEY
warden add --list-clients   # read-only view of every detected client config
```

Any npm package not in the catalog gets a deny-by-default starter policy;
grant only what it needs with repeatable `--allow-host` / `--allow-env` /
`--allow-read` / `--allow-write` flags, or bring your own reviewed policy
with `--policy`. Credentials are reported by name only and never written
into configs. `--dry-run` previews without writing; `--yes` skips the
prompt in scripts. Undo with `warden unwrap --client <client> --server
<entry>`. The manual flow below remains for non-npm runtimes, custom data
paths, and gateways.

## Choose a candidate profile

```bash
warden packs list
warden packs show filesystem
warden packs prepare filesystem --output /absolute/new-runtime --backend linux
warden packs generate filesystem --runtime /absolute/prepared-server --path /absolute/selected-data --output /absolute/filesystem.yaml
```

Profiles cover filesystem, local Git, memory, maintained GitHub, Brave Search,
and Context7. All remain **candidate** until actual upstream and denial workflows
pass. Recorded release metadata does not attest your installed runtime/dependencies.
Four npm profiles support `packs prepare`: it runs the trusted Node/npm installer
inside the selected sandbox, grants only registry.npmjs.org egress, disables
install scripts, sets a dedicated installer home/cache and never inherits user
npm credential configuration. Output must be a new directory. Failure retains
incomplete output and never retries outside the sandbox. Review its package-lock
and integrity metadata. Git/Python and GitHub's native binary require separate
preparation. Runtime policies never grant package registries.

Generation writes an owner-only policy and `.pack.json` metadata with a policy
hash. It refuses existing outputs, whole-home/root grants, and memory storage
containing the runtime. Directory symlinks are resolved. Runtime gets read access;
filesystem/Git data is read only; memory storage is writable. Set MEMORY_FILE_PATH
inside that storage directory. GitHub requires its native binary, `--read-only`,
restricted provider credentials, and verified proxy behavior. Search/context
profiles also require proxy verification. Hostname rules alone do not constrain
API mutations. Runtime base mounts and scratch storage are backend-specific.
For npm profiles, generation checks the installed package manifest's name/version
and includes a command using the prepared entry point. This is local manifest
validation, not signature attestation or a dependency content pin.

## Preview an existing server

```bash
warden clients
warden wrap --client cursor --scope project --server filesystem --policy /absolute/filesystem.yaml --dry-run
```

`--server` is an existing configuration name. Only `command` and `args` change;
env values/references, cwd, trust, approvals, timeouts and unknown fields stay intact.
No new server is installed. Default behavior is preview; `--dry-run` performs no
writes and starts no process. Preview omits command arguments and env values.
For a generated npm profile use `--use-policy-command` explicitly to replace
an old npx/online launcher with the reviewed prepared command. Undo restores the
original command/args. Without this option the original launcher is preserved;
it must already use the prepared local runtime to work with these narrow grants.

| Adapter | Configuration |
|---|---|
| claude-desktop | Standard macOS/Windows desktop JSON; other locations via `--config` |
| claude-code | Project `.mcp.json` or user `.claude.json` top-level servers |
| cursor | Project/user `.cursor/mcp.json` |
| codex | Project/user `.codex/config.toml`; explicit `[mcp_servers.NAME]` tables |
| vscode | Project `.vscode/mcp.json` (JSONC); profile path via `--config` |
| gemini | Project/user `.gemini/settings.json` |
| generic | Explicit JSON file with `mcpServers` or `servers` |

Claude Code nested local-scope/managed/plugin sources are not auto-discovered.
Codex inline/dotted launcher layouts are refused for editing; use explicit tables.
Host versions are not detected. These are fixture-tested adapters, not live host
certifications. See official [Claude Code](https://code.claude.com/docs/en/mcp),
[Cursor](https://cursor.com/docs/mcp), [Codex](https://learn.chatgpt.com/docs/extend/mcp?surface=cli),
[VS Code](https://code.visualstudio.com/docs/agent-customization/mcp-servers),
[Gemini](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html),
and [Claude Desktop](https://modelcontextprotocol.io/docs/develop/connect-local-servers).

Comments outside edited launcher values remain intact. Duplicate JSON keys,
malformed configs, remote entries, nested Warden wrappers and config symlinks
are refused. SDKs can register the resulting structured command/args directly.

## Apply, verify, and undo

```bash
warden wrap --client cursor --scope project --server filesystem --policy /absolute/filesystem.yaml --yes
warden inventory --client cursor --scope project
warden unwrap --client cursor --scope project --server filesystem --dry-run
warden unwrap --client cursor --scope project --server filesystem --yes
```

Apply runs an inert Warden probe in the real sandbox before writing. It never
starts the target for readiness or retries directly. Failed readiness leaves
the config unchanged. Use an absolute `--warden-bin` for a different installation.

Owner-only backups and undo records live in Warden's private user configuration
storage (`warden/setup` beneath the OS user config directory), outside project
data. They can contain original secrets. Set an absolute `WARDEN_SETUP_STATE_DIR`
for isolated deployments; setup refuses grants containing or inside this storage.
Keep them private. Writes use same-directory temp
files and an exclusive Warden lock. Observed config/policy edits abort; external
editors do not honor the lock and can race the last compare/rename. Close your
host/settings editor while applying. Backups are retained after undo.

Repeated unchanged wrap is a no-op. Changed policy grants require undo/review.
Generated launchers pin the reviewed policy SHA-256; `warden run` refuses a
changed policy before launch. Remove the wrapper and review any grant update
before wrapping again. Digest-pinned runs do not permit approval mode to mutate
their policy automatically.
Undo restores exact original bytes when unchanged; after unrelated edits it
restores only the launcher. Changed managed launchers produce a safe conflict.
Use `--output /absolute/export.json` to export without mutating the source.

Restart the host and retain normal trust/approval controls. Then:

1. Discover the server and complete an allowed task using test data.
2. Prove the server started, then attempt a read outside selected data/runtime paths.
3. For read-only profiles, attempt a write to the selected fixture and confirm denial.
4. For API profiles, test the permitted API and a denied host. Use fake/test credentials.
5. Inspect `warden logs`; record host/OS/backend/Warden/upstream versions, policy hash,
   date and scope. Audit coverage depends on the backend.

Verification is **manual**. Setup does not negotiate MCP or automatically roll back
a later host connection failure. Use undo if a connection fails. Inventory reports
local stdio entries as direct, wrapped/workflow-unverified, or drifted; it does not
discover inherited/plugin/remote connections. Standard downstream HTTP, hosted
agents and complete protocol filtering remain batch 4 work.
