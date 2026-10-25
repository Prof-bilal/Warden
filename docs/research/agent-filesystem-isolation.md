# Agent filesystem isolation research

## Executive conclusion

Warden already has the right enforcement shape for an agent-independent boundary: a
native process sandbox, a deny-by-default environment, and a separate egress proxy.
The current filesystem policy is an allowlist of mounted paths, which is stronger
than prompt instructions but cannot express “allow this directory except these
descendants”. Agent runs therefore need a projected workspace when sensitive denies
fall beneath an allowed tree.

The recommended design is a hybrid: bubblewrap/Seatbelt/AppContainer (or Docker as
a declared fallback) for visibility, Landlock on Linux for a second kernel check,
strict descriptor and environment handling, and the existing network proxy.

## Mechanism comparison

| Mechanism | Boundary | Read/write | Descendants | Important limitation |
| --- | --- | --- | --- | --- |
| Bubblewrap | mount/user/PID/network namespaces | mount-level | yes | path mounts do not express arbitrary child denies; inherited descriptors must be closed |
| Landlock | kernel filesystem rights | yes | yes | rules are allow-oriented; pre-opened special handles and some `/proc` objects need separate controls |
| seccomp | syscall filtering | indirect | inherited | cannot safely authorize filesystem paths by itself |
| Seatbelt | macOS kernel policy | profile-dependent | inherited | command-line interface is deprecated and profile semantics are platform-specific |
| AppContainer + DACL | token and object ACLs | yes | process token | reparse points, inherited ACLs, and path canonicalization need dedicated handling |
| Docker | container namespace and mounts | mount-level | yes | daemon/image/runtime become part of the trust boundary |
| userspace proxy | per-operation mediation | potentially | depends on implementation | high compatibility and performance cost; easy to bypass through unmediated APIs |

Bubblewrap creates an initially empty filesystem namespace and exposes selected
bind mounts. Landlock attaches access rights to kernel objects and carries those
rights through descendants and descriptor passing. These properties make the two
complementary rather than interchangeable.

## Security-sensitive findings

- A broad read bind exposes every descendant, including hidden files and files
  added after launch.
- A path-only deny list cannot prevent a hard-link alias to an allowed inode.
- A process must not inherit arbitrary host descriptors, sockets, directory handles,
  or credential-bearing standard streams.
- A new PID namespace reduces `/proc` visibility, but `/proc` is not itself a
  complete filesystem policy.
- Git object databases must be treated as repository-sensitive data; denying only
  the working-tree `.env` is insufficient if history contains its contents.
- Environment filtering must happen before every child is launched; agent prompts
  and shell wrappers are not a boundary.

## Decision criteria

The first proof must demonstrate that shell, Python, Node, subprocesses, symlinks,
traversal, absolute paths, Git, `/proc`, and environment enumeration all receive
the same kernel-enforced result. A secret scanner may improve UX, but deterministic
policy and OS enforcement remain authoritative.
