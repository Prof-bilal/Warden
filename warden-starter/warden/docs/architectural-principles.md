# Architectural principles

Status: normative. This document explains *why* Warden is shaped the way it is
and what must remain true as new isolation backends are added. It is deeper than
`ARCHITECTURE.md` (which describes the current implementation) and
`isolation-backend-contract.md` (which describes the contributor checklist).

Concrete backend selections, sequencing, and product decisions belong in an
implementation plan. This document deliberately does not choose technologies; it
defines the constraints any choice must satisfy.

---

## 1. The invariant

> One Warden policy system. Multiple isolation backends. Consistent security
> guarantees across platforms.

"Consistent" does not mean identical mechanisms. It means a policy author can
reason about the same grant vocabulary, the same deny-by-default rule, the same
failure behavior, and the same audit contract on every platform. Where a backend
cannot provide a guarantee, it must say so and refuse the policy rather than
quietly weaken it.

The system has one job:

> Given an untrusted workload and a declarative policy, enforce the policy with
> the strongest boundary the host can provide, and prove what was enforced.

Everything else is subordinate to that.

**Today vs. target.** This document defines the target every backend must
satisfy. Current behavior, including known gaps, is documented in `security.md`
and `ARCHITECTURE.md`. A documented current limitation is not precedent for a
new backend; where the two differ, this document says so explicitly.

---

## 2. System model and trust boundaries

```text
+----------------------------- untrusted ------------------------------+
|  agent / MCP server / repo scripts / package hooks / children        |
+------------------------------+---------------------------------------+
                               | syscalls, stdio, network
+------------------------------v---------------------------------------+
|                     isolation boundary (backend)                     |
|  filesystem view | network namespace/device | env | process tree     |
+------------------------------+---------------------------------------+
                               | mediated decisions
+------------------------------v---------------------------------------+
|                    Warden runtime (trusted)                          |
|  policy engine | backend lifecycle | egress proxy | audit | evidence |
+------------------------------+---------------------------------------+
                               |
+------------------------------v---------------------------------------+
|                       host kernel / hypervisor                       |
+----------------------------------------------------------------------+
```

Trusted computing base: the host kernel or hypervisor, the backend primitive,
the Warden launcher, the policy parser, the host-side egress proxy, and the
audit/evidence writers. Everything inside the boundary is untrusted, including
the workload's own configuration and any helper it starts.

Two consequences drive many decisions:

1. **The policy decision must happen outside the boundary.** A workload cannot
   be trusted to enforce, report, or respect its own limits.
2. **The smaller the trusted base, the better.** Prefer OS-native primitives
   over layers of userspace mediation. A larger trusted base is more code to
   audit and more places for a fail-open bug.

---

## 3. Threat model

### Assets

Credentials (cloud, API, SSH, Git), private source, deployment configuration,
local user files, browser and session state, environment variables, and the
integrity of the host. Test fixtures use fake values only.

### Actors

- a prompt-injected or compromised agent;
- a malicious or compromised MCP server;
- a compromised dependency, installer, or package hook;
- an accidental agent mistake;
- a malicious helper process, language server, or child;
- a local attacker who can influence host configuration but not the kernel.

### Assumptions

- The host kernel, hypervisor, and the platform sandbox primitive are not
  already compromised.
- Warden runs as the invoking user. Linux and macOS native backends are
  unprivileged; Windows AppContainer requires WFP and an ETW audit session,
  which today need an elevated (admin) process, and Warden fails closed without
  them. A backend must declare its privilege requirements and refuse when they
  are absent.
- The policy file and the Warden binary are trusted inputs. If they are
  attacker-controlled, the boundary is already lost.
- The agent runtime may request anything and may lie about permissions. Warden
  decides independently.

### In scope

- Obtaining an ungranted file, host, environment variable, secret, or process
  capability through the workload or any descendant.
- Escaping the boundary through symlinks, hardlinks, path aliases, device nodes,
  setuid binaries, ptrace, namespaces, shared folders, or the control channel.
- Reaching the network without the host proxy's decision.
- Continuing to run after a required primitive fails.
- Leaking secrets through argv, logs, environment, images, or audit events.

### Out of scope

- Kernel or hypervisor zero-days.
- A host administrator who can read Warden's memory or modify its binary.
- Data intentionally sent to an allowed endpoint.
- Side channels that a shared kernel or shared hardware inherently exposes,
  except where the backend claims otherwise.

State these limits plainly in user-facing docs. Overclaiming is a security bug.

---

## 4. Principle 1: one policy, one source of truth

The YAML policy is the contract between the user and every backend. Backends are
translators, not policy authors.

Rules:

- **One schema.** New capability settings go into the shared schema, not into a
  backend-specific file or flag namespace.
- **One validation path.** Parse, resolve, normalize, validate. Backends may add
  plan-time feasibility checks, but they may not reinterpret grants.
- **One normalization.** Overlapping read/write grants coalesce deterministically;
  deny always wins. Backends consume the normalized result.
- **Fail closed on schema.** Unknown keys are rejected (`KnownFields(true)`).
  This is intentional: a typo must not silently drop a restriction.
- **Versioned evolution.** Version 2 adds `version`, `temporary_grants`, and
  `guard`; unversioned and version 1 policies still load with a migration
  warning, and `warden migrate` previews the change. A backend must refuse a
  policy feature it cannot enforce (for example, active temporary grants
  currently require the Linux or Windows native sandbox) instead of ignoring it.
- **Supervisor-owned fields stay supervisor-owned.** `ReadOnlyPaths`,
  `AgentSocket`, and `AgentCwd` are not policy input. A backend must never accept
  them from YAML.
- **Agent preparation is part of enforcement.** An agent policy requires the
  supervisor that builds the sanitized projection. A backend must refuse to run
  one directly.

Compatibility note: strict schema parsing means an older binary rejects a newer
policy. That is the correct failure direction. Schema evolution needs an explicit
versioning decision, not a loosened parser.

---

## 5. Principle 2: a modular isolation runtime

Isolation backends differ in mechanism, not in contract. The runtime must expose
a stable seam so the policy engine and CLI do not grow platform branches.

The seam has four responsibilities:

1. **Probe.** Report what this host can enforce, conservatively.
2. **Plan.** Translate a validated policy into an inspectable, deterministic
   plan.
3. **Run.** Create the boundary, launch the workload, and mediate the run.
4. **Tear down.** Reap the tree and release host resources, idempotently.

Design consequences:

- Selection (`auto`, explicit backend) is policy-agnostic and fails closed. It
  may prefer a native primitive, but it may not "try the next one" after a
  security-relevant failure.
- A backend declares capabilities. A policy that requests an unavailable
  capability is refused before launch, with the capability named.
- Plans are testable without a host. Pure plan generation is what makes a
  multi-platform matrix affordable in CI.
- Backend-specific code stays behind build tags and stubs. Shared packages must
  cross-compile for every supported platform.
- The command/stdio contract is identical across backends. The MCP client must
  not be able to tell which backend ran.

Isolation classes are a useful vocabulary, not a hierarchy of virtue:

| Class | Boundary | Typical strength | Typical weakness |
| --- | --- | --- | --- |
| Process | namespaces, LSMs, tokens, job objects | fast, small trusted base | shared kernel |
| Container | container runtime + namespaces | image isolation, portability | shared kernel, daemon trust |
| Hardware-virtualized | hypervisor + guest kernel | separate kernel, device model | boot cost, image supply chain, share semantics |

A class is not a guarantee. The guarantee is the capability report for a
specific host, policy, and backend.

---

## 6. Principle 3: deny by default, transitively

The default is absence, not permission. This applies to every resource class and
to every descendant.

- **Filesystem:** ungranted paths are invisible, not merely unreadable. Deny
  rules win inside grants. Symlinks, hardlinks, `..`, and case aliases must not
  escape.
- **Network:** no route, no DNS, no inbound. The host proxy is the only decision
  point.
- **Environment:** names only, values from the parent, deny wins, empty allowlist
  means empty environment.
- **Secrets:** never in argv, guest command lines, images, logs, or policy files.
  Protection is deterministic, not classifier-based.
- **Process:** every descendant is contained. Executables come from granted
  paths. Escape primitives are blocked.
- **Limits:** requested limits must be enforced on the tree, or the run refused.
  Today limit accounting covers the direct child tree, and grandchildren spawned
  via `fork()` + `exec()` outside it may escape limits (`security.md`); new
  backends must close this gap.

**Network, today vs. the requirement.** The current proxy mediates HTTP/HTTPS
only. Raw TCP and UDP are not covered by the hostname allowlist; the native
backends block them by removing the route, not by filtering. A new backend must
preserve that property, and must refuse network grants if it cannot prevent a
raw route.

Transitivity is the hard part. A boundary that holds for the first process but
not for `sh -c`, a language runtime's child, or a helper daemon is not a boundary.
Tests must exercise grandchildren, not just the launcher.

---

## 7. Principle 4: fail securely

Security failures are not availability failures. The safe response to "I cannot
enforce this" is to stop.

Failure taxonomy:

| Failure | Response |
| --- | --- |
| Primitive missing | Refuse to run (`RefuseToRun`) |
| Probe inconclusive | Treat as unavailable |
| Policy unsatisfiable by backend | Refuse before spawn, name the capability |
| Enforcement step fails during start | Tear down, return error, no partial run |
| Backend crashes | Kill the tree, reap, return a distinct error |
| Control channel lost | Stop the workload |
| Audit sink fails | Surface it; fail when evidence is part of the guarantee |
| Cleanup fails | Report; do not reuse artifacts |

Explicit fallback is allowed only where the architecture defines it: the
documented `auto` ordering, or an explicit `--backend docker`. A runtime failure
must never be converted into a fallback. "The VM would not boot, so we ran it
natively" is a vulnerability, not resilience.

Partial enforcement is the subtle case. If a backend can enforce filesystem and
environment but not network, it may not run a policy that grants network access.
It may run a policy with an empty network section only if the result is a
genuinely closed boundary, and it must say so in the capability report.

---

## 8. Principle 5: backward compatibility and guarantee classes

Existing behavior is part of the product:

- Linux keeps bubblewrap-based isolation.
- macOS keeps Seatbelt-based isolation.
- Windows keeps AppContainer-based isolation.
- Docker remains the documented cross-platform fallback.

New backends are additive. They must not change `auto` ordering by accident,
rename existing backends, or alter exit/stdio semantics. A user who upgrades
Warden should not discover that their policy now runs under a different boundary
without an explicit choice.

Compatibility also means honest guarantees:

- Publish a capability matrix per platform and backend.
- Include the backend name and artifact identity in audit and evidence.
- Document what a backend cannot observe (for example, no syscall-level file
  audit on some backends).
- Refuse policies that ask for guarantees the selected backend cannot provide.

Published behavior today is in `security.md` and its Known Limitations table.
This document's requirements are the target for new backends; a current
limitation listed there is not precedent for a weaker new backend.

---

## 9. Workspace sharing and data flow

Shared folders and projections are where a strong boundary most often leaks.

Principles:

- **Minimum share.** Share only policy-granted paths.
- **Path identity.** The current contract exposes a granted path at the same
  absolute path inside and outside the boundary. A virtual root is acceptable
  only if the mapping is explicit, documented, and tested; do not silently
  rewrite the paths policy authors wrote.
- **One-way where possible.** Prefer read-only shares for inputs. Read-only must
  be enforced by the host, not only by the guest mount.
- **No traversal through the share.** Guest-created symlinks, hardlinks, device
  nodes, and `..` must not reach host paths outside the grant.
- **No implicit shares.** Never share `$HOME`, the repository root, `/tmp`,
  clipboard, SSH agents, Docker sockets, or package caches by default.
- **Explicit write-back.** Define what the host sees, what the guest sees, and
  what happens on a mid-write crash. Do not claim atomicity the implementation
  does not provide.
- **Overlay honesty.** A sanitized projection must not fall back to the original
  file if redaction or copying fails. Fail the preparation instead.
- **Redaction is not a boundary.** Projection reduces exposure; the sandbox is
  what enforces access. Do not rely on projection alone.

---

## 10. Networking and egress

Network policy is mediated, not merely filtered.

```text
workload -> local bridge -> host proxy -> policy check -> DNS -> upstream
```

Requirements:

- The workload has no direct route and no direct DNS. Attempts fail rather than
  bypass the proxy.
- The host proxy checks `network.allow` before any DNS lookup or upstream
  connection. This ordering matters: a DNS query can itself leak intent.
- Every allow and deny is audited.
- Inbound connectivity is denied by default and has no schema surface today. Do
  not add one incidentally.
- Link-local and cloud metadata addresses stay blocked even under broad
  allowlists.
- A backend may replace the transport (for example, a guest network device), but
  it must preserve the property that the decision happens on the host.
- Loopback inside a guest or container is not a trust boundary. Do not expose
  host services on it.

**Current limitation.** The proxy mediates HTTP/HTTPS only. Raw TCP and UDP
cannot be filtered by it; the native backends prevent them by having no route.
A backend that exposes a raw route bypasses `network.allow`, so it must refuse
network grants unless it can prevent one.

If a guest cannot be forced through the host decision point, the backend does
not satisfy the network guarantee and must refuse network grants.

---

## 11. VM lifecycle and host-to-guest communication

Hardware-virtualized backends add a lifecycle that process backends do not have.
The principles are the same: bounded, authenticated, observable, and fail closed.

- **Boot is part of the boundary.** A workload does not start until the guest is
  ready and the control channel is authenticated. A boot timeout is a failure,
  not a reason to continue.
- **The control channel is not a shell.** It carries bounded control messages
  for the run. It must not expose arbitrary command execution, host filesystem
  reads, or host environment reads.
- **Authenticate both directions.** Per-run credentials, a private channel, and
  framing validation are the minimum.
- **Bound everything.** Message size, concurrency, queue depth, and time. A hung
  guest must not hang the host indefinitely.
- **Keep stdio transparent.** The MCP client's stdio must not be multiplexed
  with control traffic.
- **Stop on channel loss.** Never continue a workload whose mediation path is
  gone.
- **Crash-safe cleanup.** Reap processes, mounts, sockets, disks, and firewall
  rules on normal exit, signal, panic, and crash. A later run or `doctor` should
  detect leftovers.
- **No snapshots as a security feature.** A snapshot rollback is a convenience;
  it is not a substitute for a clean boundary per run.

---

## 12. Resource governance and performance

Limits are security-relevant: a workload that can exhaust the host is a
denial-of-service vector.

- Enforce memory and wall-clock limits on the whole tree; report the limit that
  fired as structured data.
- **Current gap.** Limit accounting covers the direct child tree; grandchildren
  spawned via `fork()` + `exec()` outside the tracked tree may escape limits
  (`security.md`). New backends must close this or refuse the limit.
- Add CPU, process-count, and disk limits to the shared schema before any
  backend claims them. Until then, documentation and `doctor` must say they are
  unavailable.
- A requested limit that cannot be enforced causes refusal, not a warning.
- Measure overhead per backend: cold start, steady-state CPU, memory footprint,
  and teardown time. Publish the numbers and set a budget; performance is part
  of usability for a local tool.
- Do not hide boot cost behind a warm pool unless the pool itself is modeled as
  untrusted and reset per run. Reusing a dirty guest weakens the boundary.

Performance targets are a product decision, but the measurement method is not:
same workload, same policy, per backend, reproducible, checked in CI where the
hardware allows it.

---

## 13. Audit, evidence, and observability

Audit is part of the security contract, not a debugging extra.

- Log every allow and deny with timestamp, action, resource, decision, and
  reason.
- Include run identity, backend name, policy identity, and (where available)
  process identity. `audit.Event` now carries command, tool, PID, hash, and
  kind fields, but still lacks explicit run and backend identity; new backends
  should close the gap rather than widen it.
- Never log file contents, environment values, tokens, or request bodies.
- Be explicit about observability per backend. A backend without syscall tracing
  cannot claim per-file access logs. Say what it can prove.
- Evidence and attestation must bind the backend, the policy, the binary, and
  any boot artifacts (kernel, root filesystem, image digest). A signature over
  an incomplete identity is misleading.
- Verification must require independent trust and a fresh revocation snapshot.
  Do not treat a valid signature as proof of safety.

---

## 14. Supply chain and platform requirements

New isolation mechanisms bring new trusted artifacts. Treat them like code.

- **Images and kernels are trusted inputs.** Pin digests, verify signatures
  where the ecosystem supports it, and record the identity in evidence. Never
  use a floating `latest`.
- **Reproducibility.** Prefer reproducible builds for guest images and document
  how to rebuild them. If reproducibility is impossible, document the gap.
- **Update path.** Define how images are updated, how old images are retired,
  and how a compromised image is revoked.
- **Host prerequisites.** Document required CPU features, kernel modules, device
  access, entitlements, code signing, nested-virtualization support, and any
  elevated/admin requirement. Windows WFP and ETW need an elevated process
  today; `warden doctor` must check prerequisites and refuse when missing.
- **CI reality.** Some primitives are unavailable on hosted runners. Mark those
  tests as explicit skips with the required hardware, and run them in a
  scheduled or self-hosted job. A silent skip is a coverage lie.
- **Licensing.** Review the license of every new runtime dependency and image
  component before adoption.

---

## 15. Adding a backend: decision criteria

A new backend is justified when it provides a guarantee the existing set cannot,
on a platform users actually run. Before accepting one, answer:

1. **What new guarantee does it provide?** Separate kernel? Stronger filesystem
   semantics? A platform Warden does not currently cover?
2. **What does it weaken?** Boot cost, audit fidelity, share semantics, supply
   chain surface, platform requirements.
3. **Can it consume the existing policy without schema changes?** If not, the
   schema change must be justified for all backends.
4. **Can it fail closed?** What happens when the hypervisor, device, image, or
   control channel is unavailable?
5. **Can it be tested?** Conformance suite, escape tests, fail-closed tests, and
   a CI path that actually runs them.
6. **What is the trusted computing base?** Count the new components and say why
   each must be trusted.
7. **What is the performance cost?** Cold start and steady state, measured.
8. **What privilege does it require?** Root, admin, entitlements, or device
   access? Does it fail closed when the privilege is absent, and does `doctor`
   explain the requirement?

If the answers are "weaker audit, new supply chain, no CI, and a schema fork,"
the correct decision is usually not to add the backend.

---

## 16. Non-goals

- Agent identity and "who asked" provenance. Warden bounds what a workload can
  do, not who is acting through the client.
- Multi-tenant or server-side deployment. Warden targets a developer running
  local tools.
- Replacing the operating system's security model. Warden composes the primitives
  the host provides.
- A single mechanism for every platform. The contract is shared; the mechanism
  is not.

---

## 17. Open questions

These are deliberate, not oversights. Resolve them with an explicit decision
before the code depends on an answer.

1. **Schema versioning.** 2.0 introduces `version: 2` and `warden migrate`.
   Decide whether migration is the only supported path, and how long version 1
   remains accepted.
2. **Guarantee reporting.** Should `warden doctor` print a per-policy effective
   guarantee report, or only per-backend capabilities?
3. **Audit parity.** How do we present weaker audit on stronger isolation
   without implying the boundary is weaker overall?
4. **Path identity for VMs.** Is same-absolute-path a hard requirement, or may a
   backend use a declared virtual root?
5. **Limit semantics.** Are requested limits hard guarantees, or may a backend
   document "best effort" for a specific limit? The current principle says hard.
6. **Image lifecycle.** Who signs guest images, how often are they rebuilt, and
   what is the revocation story?
7. **Warm pools.** Can a pre-booted guest be reused safely, and what must be
   reset between runs?

---

## Related documents

- [Architecture](https://github.com/Prof-bilal/Warden/blob/main/ARCHITECTURE.md) - current components, policy schema,
  backend behavior.
- [Isolation backend contract](isolation-backend-contract.md) - contributor
  checklist.
- [Policy schema](schema.md) - field-level reference, including version 2.
- [Security review](security.md) - user-facing security model and limits.
- [Testing guide](testing.md) - local and platform test workflow.
- [Agent workspace](agent-workspace.md) - projection and credential handling.
