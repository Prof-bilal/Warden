# ADR: Warden agent sandbox

## Problem and requirements

Coding agents execute arbitrary programs and must not rely on prompts to protect
files. The boundary must deny filesystem, environment, process, and network access
by default, preserve useful coding workflows, inherit across descendants, and fail
closed when the host cannot provide the requested guarantee.

## Options

### A — visibility sandbox

Expose only explicit mounts in a new filesystem namespace. This is fast, compatible,
and already matches Warden's Linux design. It cannot express arbitrary exceptions
inside a broad mounted directory.

### B — kernel access control

Use Landlock, Seatbelt, AppContainer/DACLs, or equivalent kernel policy. This gives
strong descendant enforcement and read/write distinctions, but semantics vary by
platform and Landlock is not a general child-path deny language.

### C — userspace mediation

Intercept every filesystem operation. This has high overhead and compatibility risk,
and any unmediated syscall, descriptor, helper, or kernel interface can become a
bypass. It is not the primary boundary.

### D — hybrid

Combine restricted visibility, native kernel policy, environment filtering, process
containment, proxy egress, auditing, and a projected workspace for fine-grained
denies. This gives the smallest practical trusted computing base while preserving
compatibility.

## Decision

Choose D. Agent runs use a workspace projection whenever deny rules or protected
defaults target descendants of an allowed tree. Native backends remain the primary
enforcement mechanism; Docker is an explicit fallback only when its capability
probe succeeds.

## Policy direction

Preserve `filesystem.read` and `filesystem.write`, add `filesystem.deny`, and make
deny take precedence. Add `filesystem.mode: workspace` and deterministic
`filesystem.protect_secrets`. Never use an AI classifier as the authority.

## Approvals and audit

Ordinary grants may be approved once, for a session, or persistently when bound to
the canonical resource, operation, project, policy hash, and run. Protected-secret
denials are not permanently approved by the ordinary flow. Audit events contain
actor/run/process identity, operation, canonical resource, decision, reason, and
policy identity, never file contents or secret values.

## Rejected alternatives

Prompt instructions, README rules, unrestricted parent-directory mounts, and a
userspace filesystem proxy are rejected as the sole boundary. A global “allow agent
filesystem access” approval is rejected.

## Known limitations

Kernel compromise, host administrators, explicitly granted resources, and
exfiltration to explicitly allowed destinations remain outside this boundary.
