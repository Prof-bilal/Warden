# Batches 1–3: first working implementation

Date: September 30, 2026. Website and CLI developed together. This checkpoint
starts all three batches; it does **not** mark their research-plan release gates complete.

Release follow-up: [0.2.0-beta.1 preparation](RELEASE_PREVIEW_STATUS.md) records
packed npm installation and actual Codex Linux app-server checks. Other host
workflows and candidate pack gates still need independent evidence.

## Batch 1: boundary and evidence

Added the mode/capability baseline and website `/protection` page. Corrected
stale unsandboxed-fallback, resource-limit and trace guidance in both security
docs. Distinguish upstream metadata, policy/config fixtures, backend execution
and real client workflows. No fresh runtime/trace overhead number is claimed.

Native Linux sandbox tests ran with positive startup controls, allowed reads,
blocked unlisted reads/writes, env isolation and resource limits. A real pinned
filesystem server also passed the stdio workflow below. Other-platform and
live-client assurance remain separate work.

## Batch 2: candidate policy profiles

One catalog feeds the Go binary and website. Six profiles identify filesystem,
Git, memory, maintained GitHub, Brave Search and Context7 upstream versions.
`packs list/show/generate` create narrow YAML plus owner-private metadata with
a policy hash. Generation requires explicit existing data/runtime directories;
refuses whole-home/root grants, replacement files and overlapping writable
memory/runtime directories. Credentials are allowed by name only.

Website `/policies` supports search/category filtering, profile details, grant
review, upstream links and setup handoff. Every profile remains **candidate**.

`packs prepare` supports four npm profiles with a sandboxed installer, disabled
scripts, dedicated home/cache and registry-only egress. Generation validates the
installed package name/version and supplies its prepared entry point. Git/Python
and native GitHub preparation remain separate. This does not cryptographically
attest installed versions, pin dependency content or implement revocation updates.
GitHub/search profiles still need permitted API and proxy-behavior tests.

## Batch 3: reversible multi-client setup

Added adapters for Claude Desktop, Claude Code, Cursor, Codex, VS Code and Gemini
CLI; generic explicit JSON covers other command-launching MCP hosts. Added
`clients`, `wrap`, `unwrap` and read-only `inventory` commands. Website `/setup`
and `/setup/[client]` generate quoted macOS/Linux or PowerShell instructions.
`/compatibility` labels adapter fixtures separately from live host verification.

Setup defaults to a redacted preview. Apply probes the selected sandbox with an
inert Warden process, preserves every field except the selected launcher,
stores a protected original backup/undo record outside project data, and replaces
config atomically. Setup refuses grants containing or inside backup storage.
Windows uses a restricted owner DACL for setup output; Unix uses 0600/0700.
Generated launchers include an absolute Warden path and reviewed policy digest.
An explicit `--use-policy-command` opts into the prepared launcher, avoiding a
runtime npx download while retaining original argv for undo.
Changed policies fail before target launch. Repeated wrap is idempotent. Undo
preserves unrelated edits and refuses launcher conflicts. Inventory detects
launcher/policy drift and leaves configured entries workflow-unverified.

JSON/JSONC/TOML tests cover settings, credential references, comments, spaces,
malformed/duplicate keys, remote refusal, symlinks, concurrent edits and undo.
Previews do not start configured servers or expose args/env values.

Limits: no host-version detection, live GUI-host verification or automatic MCP
verification/rollback. Claude Code nested local-scope sources and Codex inline
launcher layouts are not editable. VS Code user profiles require an explicit
path. External editors can race compare/rename because they do not honor Warden's
lock. Windows replacement/ACL behavior requires actual Windows tests. Standard
remote HTTP and hosted-agent support remain batch 4.

## Validation at this checkpoint

- Go build, vet and the full Go test suite passed after the final policy-digest
  and private-storage checks.
- Native Linux sandbox checks actually executed rather than skipped.
- Website production build and its TypeScript checks passed (82 generated pages).
- Windows amd64 and macOS arm64 cross-compilation passed; this is compile
  coverage, not native platform assurance.
- Sandboxed dependency preparation passed for the pinned filesystem npm release,
  with scripts disabled and an isolated installer home/cache/config.
- Real `@modelcontextprotocol/server-filesystem@2026.8.31`, native Linux, simulated
  stdio host using Cursor's configuration shape: initialize, tools/list, allowed
  read, denied write within the server's permitted directory, denied outside
  path, preview, apply, second wrap and exact undo all passed. This does not
  certify the Cursor application itself. Reproduce with
  `testdata/ecosystem/smoke-filesystem.mjs` after preparing the pinned package.
- Browser inspection was denied by the browser security policy. Visual layout
  and interactive browser behavior remain unverified in this session.

No real user MCP configuration was modified. No deployment, release or commit
was made. Existing unrelated `.commandcode/` and `video/` content was preserved.
