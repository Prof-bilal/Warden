# Editing files with hidden credentials

`warden agent` gives a local coding tool a temporary, sanitized workspace on
Linux, macOS, or Windows. Explicitly selected string values become opaque
Warden placeholders.
The agent can edit surrounding code and configuration. After it exits, Warden
validates every protected field, shows a sanitized before/after review, and
asks the human at the controlling terminal to type `apply` before changing
host files.

This is development-checkout functionality. Build the new CLI from source;
an older npm release does not acquire this command automatically.

## First session

Put this policy outside the project being edited. Paths in filesystem grants
and `agent.workspace` resolve against the policy's directory. Redaction file
paths resolve inside `agent.workspace`.

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
    env: [OPENAI_API_KEY, AWS_SECRET_ACCESS_KEY]
    files:
      - path: .env
        format: dotenv
        fields: [OPENAI_API_KEY, DATABASE_URL]
      - path: config.json
        format: json
        fields: [/credentials/api_key]
```

```sh
go build -o /tmp/warden ./cmd/warden
/tmp/warden agent --policy /path/to/policy.yaml -- claude
# Or use an installed Codex executable:
/tmp/warden agent --policy /path/to/policy.yaml -- codex
```

These examples launch ordinary executables; they do not claim live-client
authentication compatibility. The executable, runtime dependencies and provider
connection must be available under the sandbox policy. The initial policy above
is offline. Provider access requires explicit hosts and an authentication design
that does not hand application credentials to agent-generated shell code.
Use a system-installed self-contained executable where possible: user-home
Node launchers may need dependencies unavailable in this initial single-root mode.

The session starts in the copied project, and prints a notice directing the
agent to `.warden/NOTICE.md`. `.warden/manifest.json` lists file selectors and
environment names without values. Warden also appends the notice to projected
`AGENTS.md` and `CLAUDE.md`, keeping existing sanitized instructions. These
instruction overlays and `.warden` are mounted read-only and never applied to
the host. Clients can discover the notice through their usual instruction files;
Warden does not force a model to read or follow it.

## Formats and protected edits

Supported formats: `dotenv`, `json`, `yaml`, `toml`.

Dotenv selectors are variable names. Other selectors use exact JSON Pointer
object/table paths, such as `/credentials/api_key`. A literal `/` inside a key
is written as `~1`; a literal `~` as `~0`. Array selectors are not supported.
Every selected value must be a nonempty string and every selector must exist.
There is no automatic content detection or secret classifier.

Protected fields must retain their exact placeholders. Deleting or renaming
their fields, copying a placeholder elsewhere, or replacing a protected file
rejects the candidate. Change or rotate credentials directly outside the agent
session. Warden never replaces your original credential with a placeholder.

Dotenv preserves unselected lines. Edited JSON, YAML and TOML files are
serialized through their parsers; formatting and comments may change. The
review displays the serialization to be applied with protected values hidden.
Duplicate keys, YAML anchors/aliases/merge keys, unsupported tags and ambiguous
dotenv multiline syntax are rejected. Untouched files retain their host bytes.

## Using a credential through Warden

The agent cannot safely run arbitrary edited code with a real credential:
that code could encode or print it. Warden therefore offers fixed HTTPS GET
profiles, with a Bearer credential held in the supervisor. This first version
does not inject real credentials into `npm test` or agent-controlled programs.

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

Merge these fields into a policy with the single project grant. Export the
credential on the host before starting Warden. Before launching the agent,
Warden asks the human to approve each profile for the session. An unapproved
profile is unavailable. The profile credential is omitted from the child's
environment even if the base environment allowlist names it.

Inside the session:

```sh
/.warden/proxy-bridge agent exec --profile github-user
```

The internal executable path is always made available by the native sandbox. If
`warden` is also available on PATH, `warden agent exec --profile github-user`
is equivalent. Warden accepts only the profile name. The agent cannot supply
a URL, method, request body, arguments or credentials. Profiles require an
exact host in `network.allow`, HTTPS port 443, and no query or fragment.
Redirects and private/reserved IP destinations are rejected; checked DNS
addresses are pinned when dialing. Responses are bounded to 2 MiB and 20 seconds.
Upstream headers are discarded. Exact credential echoes and common escaped
representations are replaced before returning the body.

The chosen API is a trusted recipient of the credential. Echo redaction does
not cover arbitrary encodings or transformations by a malicious API. Scope
the token and approve only destinations and operations you trust. Approval
allows repeated calls to that fixed operation for the session.

## Limits and failure behavior

- Linux uses bubblewrap, macOS uses Seatbelt, and Windows uses AppContainer with
  WFP when available. Docker projection requests refuse. Windows hosted CI can
  compile and unit-test this path, but WFP enforcement still needs a Windows
  host that permits WFP engine sessions.
- Exactly one read or write grant equal to `agent.workspace`; extra grants
  refuse. A read grant supports inspection only.
- Real workspace paths are not mounted. Absolute paths embedded in commands
  or application configuration need to be changed to projection-relative paths.
- Git history and existing `.warden` content are excluded. Git operations
  needing repository metadata are unavailable in this version.
- Symlinks, hardlinks, special files and workspace roots overlapping sandbox
  runtime paths refuse. Individual files are limited to 8 MiB, with a 256 MiB
  workspace content limit. Trim generated dependencies before starting.
- New files in existing host directories are supported. File deletion and
  directory creation/renames are not applied. Empty directories are not synced.
- Host files are checked again before apply; concurrent edits reject the merge.
  Each replacement uses an atomic rename and fsync where the platform provides
  it; Windows uses the native replacement API to preserve file identity and ACLs.
  A multi-file update is not
  a crash-atomic transaction; completed replacements are rolled back when a
  later write fails. Rollback failures are reported explicitly. Do not edit the
  host tree concurrently.
- Nonzero agent exit, unavailable TTY, rejected review, parse errors or
  placeholder tampering leave host files untouched before the apply phase.
- Temporary projections are removed on normal completion. Abrupt termination
  can leave sanitized copies in the OS temporary directory; no originals are
  deliberately written there by Warden.
- Only declared fields and environment names are protected. Copies of the
  same credential in other files require their own selectors. Hard-denied files
  stay excluded; redaction does not silently override `filesystem.deny` or
  `protect_secrets`. Remove a hard-deny for a file only when explicitly choosing
  its sanitized projection instead.
- Environment redaction names are omitted from the child environment; their
  names appear in the manifest. Other explicitly allowed variables retain
  ordinary passthrough semantics. Existing model context and unsandboxed IDE
  tools are outside this process boundary.

## Verification

```sh
go test ./internal/agentworkspace ./internal/policy
```

On Linux CI, the required native boundary test is run with
`WARDEN_REQUIRE_AGENT_E2E=1` and `TestLinuxProjectionBoundary`. macOS and
Windows CI run the portable projection and apply tests natively; the Windows
host job may still skip only the separate WFP/AppContainer escape tests when
the hosted image refuses WFP engine sessions.

The Linux E2E test proves child startup, original-file exclusion, read-only
notices, sanitized edits and restoration of the protected value. Broker
requests and human decisions record names and decisions in Warden's audit log;
file bodies, credential values and response bodies are not audit payloads.
