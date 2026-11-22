# Isolation backend contract

Status: normative contributor guidance. Applies to every backend, present and
future.

Warden's promise is one policy model, multiple isolation backends, one set of
security guarantees. A backend is not a separate product with its own rules: it
is an enforcement adapter for the shared policy engine. This document is the
contract a backend must satisfy. Reviewers use it as a checklist; contributors
use it as a specification.

If a rule here conflicts with `ARCHITECTURE.md`, stop and open an issue. The
conflict is a documentation bug, not permission to choose the weaker behavior.

---

## 1. Non-negotiables

These are not style preferences. A change that violates one of them is rejected.

1. **The policy is the only source of truth.** A backend reads a parsed
   `policy.Policy`. It does not read YAML, merge a second configuration file, or
   keep backend-specific policy keys.
2. **Deny by default, transitively.** Filesystem, network, environment,
   secrets, and process execution are denied unless granted. Restrictions apply
   to every descendant, not just the first process.
3. **Fail closed.** If a required primitive cannot be initialized or a requested
   restriction cannot be enforced, refuse the run. Never execute the target
   command without the boundary the policy asked for.
4. **No silent degradation.** Do not substitute a weaker mechanism, widen a
   grant, or drop a limit because the preferred primitive is missing. Refuse, or
   fail the run with a structured error.
5. **Every decision is auditable.** Allowed and denied access attempts produce
   structured events. Never log secret values or file contents.
6. **Backward compatibility.** Existing `linux`, `seatbelt`, `windows`, and
   `docker` behavior keeps working. New backends are additional options, not
   replacements, and `--backend auto` ordering only changes with an explicit
   architecture decision.

---

## 2. Where the code lives

| Concern | Location | Rule |
| --- | --- | --- |
| Schema, parsing, validation, normalization | `internal/policy` | Shared by every backend. No backend-specific parsing. |
| Backend selection | `internal/sandbox` (`Resolve`, `detect`) | Policy-agnostic. Must refuse when nothing is available. |
| Native backends | `internal/sandbox/{linux,darwin,windows}` | Platform-gated with build tags and stubs. |
| Cross-platform fallback | `internal/sandbox/docker` | Explicit fallback only, never a silent default on a supported platform. |
| Egress mediation | `internal/proxy` | Host-side allow decision, one implementation for all backends. |
| Audit | `internal/audit` | One event schema; backends supply observations. |
| Evidence | `internal/attestation` | Must identify the backend and its artifacts. |

The current tree dispatches with a string switch in
`internal/sandbox/backend.go`. That switch is an implementation detail, not the
contract. New backends must satisfy the lifecycle below and should be
introduced behind a common interface so selection stays independent of policy
semantics.

---

## 3. Runtime lifecycle contract

A backend must implement the following phases. Names are illustrative; the
semantics are mandatory.

```go
// Probe reports what this host can enforce. It must not mutate host state and
// must treat an inconclusive check as unavailable.
Probe(ctx context.Context) (Capabilities, error)

// Plan translates a validated policy into an immutable, inspectable plan.
// Planning is pure: no spawn, no mounts, no network. Plan errors are
// policy-unsatisfiable errors, not runtime errors.
Plan(p policy.Policy, cmd []string) (Plan, error)

// Start creates the boundary and launches the command inside it.
// There is no "start unsandboxed and fix up later" mode.
Start(ctx context.Context, plan Plan) (Instance, error)

// Wait returns the exit status and, if applicable, which limit was hit.
Wait(ctx context.Context, inst Instance) (Result, error)

// Stop terminates the process tree and tears down the boundary. Idempotent.
Stop(ctx context.Context, inst Instance) error

// Cleanup releases host resources after Stop. Idempotent.
Cleanup(ctx context.Context, inst Instance) error
```

Required behavior by phase:

- **Probe** answers "can this host enforce this class of policy?" It must be
  cheap, side-effect free, and conservative. A probe that cannot decide returns
  unavailable.
- **Plan** is deterministic. The same policy and command produce the same plan,
  so plans can be diffed, tested, and attested.
- **Start** must not fall back. If any step fails, tear down what was created
  and return an error.
- **Wait** must report limit breaches distinctly from ordinary non-zero exits,
  and must not lose the exit status when Stop is racing.
- **Stop/Cleanup** must be safe to call twice, must reap descendants, and must
  remove mounts, sockets, temporary directories, VM disks, and firewall rules.

New backends should expose the plan in a form that `warden doctor` and tests can
inspect without launching a workload.

---

## 4. Capability reporting

The cross-platform design already requires backends to report capabilities
before launch. Make that explicit in code:

```go
type Enforcement int

const (
    Enforced Enforcement = iota
    Partial
    Unavailable
)

type Capabilities struct {
    Filesystem Enforcement
    Network    Enforcement
    Env        Enforcement
    Process    Enforcement
    Limits     Enforcement
    Audit      Enforcement
    // Optional detail: which limit kinds, which audit event classes, and
    // which path semantics are supported.
}
```

Two rules follow:

1. **A policy that requests an `Unavailable` capability is refused before
   launch.** Do not run with the capability silently removed.
2. **`Partial` must be declared and documented.** For example, a backend that
   enforces memory and time but not CPU is `Partial` for limits. The run may
   proceed only if the policy does not request the unsupported part; otherwise
   it refuses.

Capability results must appear in `warden doctor` so users can see the effective
boundary before they run anything.

---

## 5. Consuming the policy

- Load through `policy.Load` (or `LoadVerified` when a digest is bound). The
  resulting policy has resolved absolute paths, normalized grants, and
  validated fields.
- Do not touch the YAML directly. Do not re-resolve relative paths.
- Do not add backend-only YAML keys. If a setting is needed, add it to the shared
  schema and require every backend to implement or refuse it.
- Never accept supervisor-owned fields from policy input: `ReadOnlyPaths`,
  `AgentSocket`, and `AgentCwd` are marked `yaml:"-"` for a reason.
- Reject an agent policy (`p.Agent != nil`) unless the process was launched by
  the `warden agent` supervisor. Preparation is part of the boundary.
- Keep `KnownFields(true)`. Unknown keys are a fail-closed feature; do not relax
  it for convenience.
- Treat policy version and features as part of validation. Version 2 fields
  (`version`, `temporary_grants`, `guard`) are shared schema, not backend
  extensions. `warden migrate` owns upgrades; a backend must not silently drop a
  v2 feature it cannot enforce.
- Active temporary grants are capability-gated today by an ad-hoc check in
  `sandbox.Run` (Linux and Windows native only). New backends must refuse a
  policy with active grants they cannot enforce, and that check should move
  into the capability model.
- Validation order is fixed: parse, resolve paths, normalize, validate. A
  backend may add plan-time checks, but it must not weaken these.

---

## 6. Enforcement requirements by resource

### Filesystem

- A granted path is visible at the same absolute path on the host and inside the
  boundary. This path identity is part of the contract.
- Everything not granted is absent, not merely unreadable. Do not mount `$HOME`,
  the current directory, `/tmp`, package caches, or the repository by default.
- `filesystem.deny` wins over `read` and `write`, including inside a granted
  tree. Deny must be resolved after paths are made absolute, and must survive
  symlinks, hardlinks, `..`, and case-insensitive aliases.
- Symlink traversal must not escape a grant. Follow the same no-follow rules the
  existing backends use; do not add a follow-everything shortcut.
- A hardlink to a denied inode is a deny. Do not treat link identity as a
  separate namespace.
- `protect_secrets` is deterministic filename protection. Do not replace it with
  a content classifier.
- Read-only grants must be enforced by the boundary, not by file permissions
  that the workload can change.

### Network

- Default is no route: no direct DNS, no direct connect, no inbound listener
  reachable from the host.
- The only egress path is the host-side proxy decision. A backend may implement
  a different transport, but the decision point must stay on the host and must
  consult `network.allow` before DNS or connect.
- `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` pointing at a loopback bridge is the
  current pattern; a new transport must preserve the same property: the
  sandboxed process cannot reach an address the host proxy did not allow.
- Block link-local and metadata addresses even if a broad allowlist would
  otherwise match. Treat loopback inside the guest as untrusted.
- Do not add host networking, NAT, port forwarding, or inbound forwards without
  a schema change and an explicit deny-by-default rule.

### Environment and secrets

- `env.allow` lists names only. Values come from the parent environment and are
  never stored in the policy.
- `env.deny` wins over `env.allow`. An empty allowlist means an empty
  environment; do not inject a default `PATH` or shell environment.
- Never place secrets in argv, the guest kernel command line, process titles,
  logs, or image layers.
- A credential broker, if present, is supervisor-owned and pre-approved. A
  backend must not let the workload define or extend it.
- Environment crossing a host-to-guest boundary must be filtered on the host
  before it crosses.

### Process execution

- Every descendant stays inside the boundary. Verify this for `fork`, `exec`,
  `clone`, shells, language runtimes, and helper processes.
- Executables must come from granted filesystem paths. Do not add an implicit
  "runtime is always readable" grant beyond what the existing backends document.
- Block escape primitives: setuid/setgid transitions, ptrace of unrelated
  processes, access to host `/proc`, namespace creation that widens the
  boundary, and device nodes.
- Enforce a process-count limit where the platform supports it. Today this is a
  gap; new backends must not make it worse.

### Limits

- `limits.memory_mb` and `limits.timeout_s` apply to the process tree, not only
  the root process.
- A limit that the backend cannot enforce must refuse the run, not be ignored.
- Report which limit fired as structured data, not just a non-zero exit.
- CPU limits are not in the current schema. Until they are, do not claim CPU
  enforcement in docs or `doctor`.

---

## 7. Fail-closed matrix

| Situation | Required behavior |
| --- | --- |
| Backend binary or host primitive missing | `sandboxerr.RefuseToRun`; no fallback to plain execution |
| Probe inconclusive | Treat as unavailable |
| Policy requests an unsupported capability | Refuse before spawn, name the capability |
| Enforcement step fails mid-start | Tear down, return error, no partial run |
| Audit sink unavailable | Surface the failure; do not silently drop security events |
| Guest/host control channel fails | Stop the workload; do not continue unmediated |
| Backend crashes | Kill the tree, reap resources, return a distinct error |
| Cleanup fails | Report it; do not reuse the artifacts for another run |
| Policy is invalid or unreadable | Refuse; never run with a partial policy |

An explicit Docker fallback is allowed only where the architecture already
documents it (`--backend auto` ordering or `--backend docker`). "The VM failed,
so run it natively" is never allowed.

---

## 8. Workspace sharing rules

Shared folders and workspace projections are the highest-risk part of a VM-class
backend because they move data across the boundary.

- Share the minimum: only paths granted by `filesystem.read`/`write`.
- Preserve the visible absolute path where practical. If a virtual root is used,
  the mapping must be declared, tested, and documented; do not silently rewrite
  paths that policy authors wrote.
- A shared folder must not become a symlink or device-node smuggling channel.
  Resolve guest-created links on the host with no-follow semantics.
- Read-only shares must be read-only at the host, not just mounted read-only in
  the guest.
- Define write-back behavior explicitly: host writes visible to the guest,
  guest writes visible to the host, and what happens if the guest crashes
  mid-write. Do not promise atomicity the implementation does not provide.
- Do not share clipboard, pasteboard, drag-and-drop, home directories, SSH
  agents, or Docker sockets by default.
- A projection or overlay must never expose the original file as a fallback when
  redaction fails.

---

## 9. Host-to-guest communication (VM-class backends)

If the backend has a guest agent or control channel:

- Authenticate both ends. A random per-run token, a private channel, and a
  bounded message protocol are the minimum.
- The channel is control, not a shell. Do not expose arbitrary command
  execution, host filesystem reads, or host environment reads.
- Bound message size, concurrency, and time. A hung guest must not hang the host
  forever.
- Treat guest input as untrusted. Validate framing and lengths before acting.
- Terminal stdio for the workload must remain transparent to the MCP client.
  Do not mix protocol traffic with control traffic.
- On channel loss, stop the workload. Do not continue with an unmediated guest.

---

## 10. Lifecycle, signals, and cleanup

- Forward termination signals to the workload and its tree; escalate after a
  bounded grace period.
- Reap orphaned descendants and host-side helpers on normal exit, signal,
  panic, and crash.
- Make cleanup crash-safe: a later `doctor` or run should be able to detect and
  remove stale mounts, sockets, disks, and firewall rules.
- Preserve exit codes and stdio transparency. A wrapper that changes observable
  behavior for the MCP client is a compatibility bug.
- Record the backend name, plan identity, and artifact identity in the run so
  audit and attestation can correlate them.

---

## 11. Audit requirements

- Log every allow and deny decision with timestamp, type, action, resource,
  decision, and reason.
- Add run identity, backend name, and policy identity. The current `audit.Event`
  lacks these; new backends should not deepen the gap.
- Never log file contents, environment values, tokens, or request bodies.
- State what the backend can observe. A backend without syscall tracing cannot
  claim per-file access logs. Document the difference instead of implying
  parity.
- Audit failure is a security event. If events cannot be written, surface it and
  fail the run when the missing evidence is part of the requested guarantee.

---

## 12. Testing requirements

A backend is not complete until the shared conformance suite passes. Prefer one
suite with per-backend adapters over copy-pasted tests.

Minimum coverage:

| Area | Cases |
| --- | --- |
| Filesystem | granted read, granted write, undeclared path absent, deny inside grant, symlink escape, hardlink, `..`, absolute alias, case alias (Windows/macOS) |
| Network | allowed host, blocked host, direct IP, direct DNS, proxy bypass, inbound connection, metadata address |
| Environment | allowlist only, deny wins, empty allowlist, secret absent from child and grandchild |
| Process | child inherits, grandchild inherits, exec outside grant fails, process-count bound, signal cleanup |
| Limits | memory breach, timeout breach, tree-wide accounting, limit error surfaced |
| Lifecycle | normal exit, non-zero exit, SIGKILL of launcher, guest/host crash, orphan reaping, idempotent cleanup |
| Fail-closed | primitive missing, probe inconclusive, unsupported policy capability, audit sink failure |
| Performance | cold start, steady-state overhead, memory footprint; publish numbers and enforce a budget |

Also required:

- Unit tests for pure plan generation on every OS (no host dependency).
- Integration tests gated by build tags and skipped explicitly when the
  primitive is absent; skipped tests must be visible in CI output.
- Escape tests that attempt the bypasses listed above, not just happy paths.
- Cross-builds (`GOOS=darwin`, `GOOS=windows`) for shared code.
- Documentation of any test that cannot run in CI, with the hardware or
  entitlement it needs.

---

## 13. Documentation and compatibility

Update all of these in the same change:

- `../ARCHITECTURE.md` (backend behavior, selection order, guarantees)
- `architecture.md` and `schema.md` (capability matrix and policy fields)
- `security.md` (what the boundary does and does not prove)
- repository `README.md` and the user-facing backend list
- `examples/` if the policy surface changes
- `mkdocs.yml` nav for new pages

Compatibility rules:

- Keep existing backend names and aliases working.
- Keep `--backend auto` preference order unless an architecture decision changes
  it.
- New backends are opt-in (`--backend <name>` or an explicit auto tier), not a
  silent replacement.
- Do not change CLI output or exit semantics without updating the docs that
  promise them.

---

## 14. Pull request checklist

- [ ] Policy is consumed through `internal/policy`; no backend-specific config.
- [ ] Deny-by-default holds for filesystem, network, env, secrets, process, limits.
- [ ] Every failure path refuses or tears down; no plain-process fallback.
- [ ] Capabilities are reported and unsatisfiable policies are refused pre-spawn.
- [ ] Descendants are contained and cleanup is idempotent and crash-safe.
- [ ] Audit events cover allow and deny without leaking values or contents.
- [ ] Conformance, escape, fail-closed, limit, and lifecycle tests pass.
- [ ] Unit tests run on all platforms; integration skips are explicit.
- [ ] Cross-builds pass; docs, examples, and the capability matrix are updated.
- [ ] Performance numbers are measured and within the agreed budget.

---

## 15. Current gaps in this repository

These are known and should be fixed before a new backend depends on them.

1. `internal/sandbox/backend.go` dispatches by string. There is no shared
   backend interface or capability probe.
2. `Capabilities` does not exist; the requirement in
   `cross-platform-agent-isolation.md` is documentation only.
3. `policy.Limits` has memory and timeout only. There is no CPU, process-count,
   or disk limit in the schema.
4. Docker hard-codes `--pids-limit 256`; the value is not policy-driven.
5. `audit.Event` gained `cmd`, `tool`, `pid`, `hash`, and `kind`, but still has
   no explicit run, backend, or policy identity.
6. Docker has no live file-access audit comparable to Linux `strace`; this is
   documented but easy to miss in guarantees.
7. `internal/policy/policy.go` still carries a stale "enforcement is M3" comment
   on `Limits`.
8. Windows and macOS capability probing is narrower than the capability model
   described above, so policy-unsatisfiable failures are not uniformly detected
   before launch.
9. Temporary-grant support is an ad-hoc backend check (`checkGrantLifetime` in
   `internal/sandbox/backend.go`), not a declared capability.
10. Version 2 added `temporary_grants` and `guard`; `guard` is a package-review
    feature, not a sandbox feature, so backends must not try to enforce it.
