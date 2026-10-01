# MCP boundary and evidence baseline

Source audit: September 30, 2026. This is implementation evidence, not certification.

| Mode | Boundary | Source / limitation |
|---|---|---|
| `run` | Selected OS/Docker process sandbox; transparent stdio | `internal/sandbox`; no MCP tool filtering in run |
| Client/gateway wrapping | Local launcher uses sandboxed run | `internal/clientconfig`, `internal/gateway`; config wrapping is not workflow verification |
| `proxy`, local upstream | Custom TCP JSON-RPC filter and env allowlist | `internal/mcpproxy`; does not sandbox upstream filesystem/network |
| `proxy`, remote upstream | Routed messages, upstream HTTP/SSE | Cannot isolate remote machine; no standard downstream HTTP endpoint |
| `trace` | Unsandboxed observation of one workload | Use trusted code, disposable data/fake credentials |
| Packs / preview / inventory | Policy/config inspection | No target launch or automatic MCP verification |
| `connect` / `serve`, local | MCP rules plus digest-pinned `run` sandbox | Bounded 2025-11-25 stdio/HTTP subset; unsupported capabilities refused |
| `serve`, remote | Authenticated client sessions and filtered upstream HTTP | Public-IP validation, no redirect, separate credentials; provider process not sandboxed |
| `creator check` / `verify` | Signed workflow evidence and artifact binding | Independent issuer trust, native platform and fresh trusted revocation snapshot required |

Backend selection refuses missing enforcement instead of executing unsandboxed.
Linux constructs mandatory user/network namespaces and never retries the backend
command directly after failure. Native Linux requires strace. Docker is an
isolated fallback. Runtime base mounts/scratch storage are additional to data grants;
audit detail differs by platform. Native egress interception covers HTTP/HTTPS;
raw PostgreSQL TCP requires separate transport work. Provider privileges constrain
allowed API mutations. Local policy configuration is trusted authority.
Managed launchers pin the reviewed policy digest. A changed policy fails before
launch instead of silently taking effect. Local runtime/dependency identities
are still not bound by this digest; profiles stay candidates.

The proxy's stdio subprocess uses its own execution path, not sandbox.Run. Standard
downstream Streamable HTTP, sandboxing under the MCP filter, structured proxy argv,
full capability filtering, protocol-era negotiation and auth isolation remain batch 4.
No 2026 protocol conformance claim follows from upstream HTTP handling. The new
gateway does not alter the legacy proxy's behavior; its separate bounded HTTP
endpoint now has real raw/official-SDK transport evidence and OAuth fixture tests.

Apply probes the real backend with an inert Warden process. Fixture tests cover
six named adapters and generic JSON, preservation/idempotence/undo/conflicts and
policy generation. Profiles remain candidates, and wrapped entries remain
workflow-unverified. Neither a fixture nor readiness probe is a client/server task.

Four npm profiles now have sandboxed, registry-only preparation with install
scripts disabled and isolated credential configuration. A real pinned filesystem
server passed initialization, discovery, an allowed read, denied write/outside
read, repeated setup and exact undo using a simulated stdio host on native Linux.
This does not certify a GUI client or promote all profiles to verified.

Release work still required: per-platform pinned upstream allowed/blocked tests,
measured runtime/trace overhead with methodology, audit-gap review, dependency
content binding and non-npm preparation, executable identity binding, update/revocation lifecycle,
live host/version tests, MCP connectivity verification with rollback, Windows
transaction tests, version detection and external-editor write-race hardening.

Keep metadata review, fixture validation, backend exercise and workflow verification
as separate evidence levels. See the implementation status in the repository root.
