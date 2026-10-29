# Protect an AI coding agent

`warden agent` gives a local coding tool a temporary, sanitized workspace.
The agent can inspect and edit project files, while selected credentials stay
hidden. Warden shows the proposed changes and asks you to approve them before
anything is written back to the real project.

This is useful when an agent needs to fix code in a project that also contains
API keys, deployment credentials, or other sensitive configuration.

## How it works

1. Warden copies the selected project into a temporary workspace.
2. Selected environment values and file fields become opaque placeholders.
3. The agent works inside that sanitized copy.
4. Warden checks that protected placeholders were not changed or moved.
5. You review the sanitized changes and type `apply` to update the host files.

If the agent exits unsuccessfully, the review is rejected, or validation fails,
the host workspace is left unchanged.

The projection contains `.warden/NOTICE.md` and `.warden/manifest.json`. These
tell the agent that values are hidden and list protected names without exposing
their contents. Warden does not force an agent to read or follow the notice.

## Create a policy

Keep the policy outside the project being edited. Paths in the filesystem grant
and `agent.workspace` resolve relative to the policy file. Redaction paths
resolve inside the workspace.

```yaml
filesystem:
  write: [./project]
network:
  allow: []
env:
  allow: [PATH, TERM, LANG]

agent:
  workspace: ./project
  redaction:
    apply: review
    env:
      - OPENAI_API_KEY
      - AWS_SECRET_ACCESS_KEY
    files:
      - path: .env
        format: dotenv
        fields: [OPENAI_API_KEY, DATABASE_URL]
      - path: config.json
        format: json
        fields: [/credentials/api_key]
```

The agent mode requires exactly one filesystem grant, and it must equal
`agent.workspace`. Use a read grant for inspection-only sessions or a write
grant when the agent needs to propose edits.

## Run the agent

Build Warden from the current source tree, then pass the agent command after
`--`:

```sh
go build -o /tmp/warden ./cmd/warden
/tmp/warden agent --policy /path/to/policy.yaml -- claude
```

Other local coding tools work the same way:

```sh
/tmp/warden agent --policy /path/to/policy.yaml -- codex
/tmp/warden agent --policy /path/to/policy.yaml -- opencode run "Inspect the project"
```

The executable, its runtime dependencies, and its provider connection must be
available under the policy. Network access is deny-by-default, so add only the
provider hosts the session needs.

## Protected formats

Warden supports these redaction formats:

- `dotenv`, using variable names such as `DATABASE_URL`;
- `json`, using exact JSON Pointer fields such as `/credentials/api_key`;
- `yaml`, using exact object or table paths;
- `toml`, using exact object or table paths.

Selectors must exist and protected values must be non-empty strings. Arrays are
not supported as selectors. There is no automatic secret scanner: declare each
file field and environment name that needs protection.

Unselected content remains visible. Copies of the same secret in another file
need their own selector. Hard-denied files remain unavailable and are not
silently replaced by a sanitized copy.

## Review and apply

After the agent exits, Warden displays a sanitized before-and-after review.
Protected values remain hidden in the review. Type `apply` only after checking
the paths and edits:

```text
Apply the reviewed changes, preserving protected values?
Type apply to continue.
```

Warden checks the original files again before applying. Concurrent host edits,
changed projections, placeholder tampering, parse errors, and unsafe file
operations reject the apply. Protected values are restored from the original
host files; Warden never replaces the original credential with a placeholder.

## Fixed credential requests

The agent must not execute arbitrary edited code with a real credential. For a
small number of trusted HTTPS GET operations, a policy can define a fixed
`secret_exec` profile. The URL, method, host, and credential are controlled by
the policy and supervisor:

```yaml
network:
  allow: [api.github.com]

agent:
  workspace: ./project
  redaction:
    apply: review
    env: [GITHUB_TOKEN]
  secret_exec:
    profiles:
      github-user:
        url: https://api.github.com/user
        env: GITHUB_TOKEN
```

The human approves a profile before the session. Inside the workspace, the
agent can request only the named profile:

```sh
warden agent exec --profile github-user
```

The agent cannot provide a different URL, request body, or credential. The
destination must be an exact HTTPS host allowed by the network policy. Treat
the chosen API as a trusted recipient and use a narrowly scoped token.

## Platform behavior and limits

- Linux uses bubblewrap.
- macOS uses Seatbelt.
- Windows uses AppContainer with WFP when available.
- Docker projection requests are refused for this agent workflow.
- Symlinks, hardlinks, special files, and unsafe workspace roots are refused.
- Individual files are limited to 8 MiB and the workspace to 256 MiB.
- New files in existing directories are supported.
- File deletion and directory creation or rename are not applied.
- Git metadata and existing `.warden` content are excluded.
- A multi-file update is not crash-atomic, although failed later writes are
  rolled back when possible.
- Abrupt termination can leave sanitized temporary copies; original values are
  not deliberately written there.

The agent workspace is a process and file boundary. It does not make an
untrusted model trustworthy, and it does not protect files already exposed in
the model context or through an unsandboxed IDE integration.

## Verify the feature

Run the focused tests from the Warden source directory:

```sh
go test ./internal/agentworkspace ./internal/policy
```

The cross-platform tests cover sanitized projection, protected edits, review,
apply, rollback, and unsafe file handling. Linux CI also runs the required
native boundary test when `WARDEN_REQUIRE_AGENT_E2E=1` is enabled.

Use fake credentials while testing. Never place a real production key in a
fixture, screenshot, policy file, or agent prompt.
