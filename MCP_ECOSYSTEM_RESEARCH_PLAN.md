# Warden: MCP ecosystem research and six-batch plan

Research date: September 30, 2026. Scope: research and planning only; no implementation changes. Recommendations and proposed commands below describe future work, not existing functionality. Research combines official protocol/client documentation, upstream server documentation, security research, and read-only inspection of this checkout. Client documentation establishes integration routes, not tested Warden compatibility. No live upstream compatibility runs or market-share measurements were performed.

## Recommendation

Build Warden as a portable security boundary for MCP connections: OS isolation for processes Warden launches, MCP policy enforcement for traffic routed through Warden, and reversible installation adapters for individual hosts. Make the core independent of model vendor. A new agent should be able to use an existing stdio command or standard HTTP endpoint without waiting for a dedicated integration.

Use precise protection labels: local process sandboxed; MCP traffic filtered; remote upstream not sandboxed by Warden; unprotected/direct connection. Protection applies only to connections routed through Warden. Built-in tools, agent shell access, browser access, in-process integrations, and alternative direct connections remain outside that boundary unless separately integrated.

Suggested positioning: “Least-privilege MCP access across AI agents.” The practical differentiator should be consistent policies, few setup steps, native isolation where supported, and verifiable evidence. Do not claim universal protection from prompt injection or that a sandbox makes every authorized API action safe.

## Findings that change the original proposal

### 1. Universal support is a transport problem and an installation problem

MCP has standard stdio and Streamable HTTP bindings. These give Warden portable connection surfaces; hosts still have different configuration formats, scopes, authentication, and installation mechanisms. [Current transport overview](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports).

The 2026-07-28 revision changes protocol semantics and removes HTTP GET streams and protocol-level sessions. Earlier clients use initialization and session behavior. Warden needs explicit protocol-era detection and a tested supported-version matrix; an HTTP proxy built only around older session mechanics cannot substantiate broad compatibility. [Current HTTP binding](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http), [version compatibility](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning).

For stdio clients: launch a Warden executable that forwards MCP messages and starts the upstream under the sandbox. For HTTP clients: expose a standard authenticated MCP endpoint. For cloud clients: that endpoint must be reachable from their execution environment; a user's localhost listener is insufficient. SDKs can use these same boundaries, except in-process MCP servers must be externalized or use separately reviewed middleware. [OpenAI remote MCP](https://developers.openai.com/api/docs/guides/tools-connectors-mcp), [LangChain MCP](https://docs.langchain.com/oss/python/langchain/mcp).

### 2. Warden already has useful foundations, but some guarantees are separate

Repository evidence:

- `README.md`: OS backends, trace/init, readiness checks, gateway integration, auditing, resource limits, examples, and attack/proof harnesses are documented.
- `warden-starter/warden/internal/gateway/wrap.go`: existing JSON/YAML wrapping preserves environment values in the host config and forwards only allowed names at runtime. Remote entries are passed through unchanged. Reuse this foundation instead of building another unrelated wrapper.
- `warden-starter/warden/internal/compat/compat.go`: compatibility checks use the policy engine only, with no real sandbox or network; end-to-end runs remain manual. The advertised matrix counts must be distinguished from fresh upstream/runtime verification.
- `warden-starter/warden/internal/mcpproxy/mcpproxy.go` and `docs/client-proxy.md`: MCP filtering and remote HTTP/SSE upstream handling exist. The downstream listener is newline-delimited JSON-RPC over TCP. It is not directly a standard client-facing Streamable HTTP endpoint. Local upstreams are spawned with `exec.Command` and environment filtering, without the run sandbox.
- The proxy's local upstream command uses `strings.Fields`; paths or arguments containing spaces require structured argv rather than a command string.
- `docs/security.md`: documents HTTP/HTTPS-only egress proxy interception, platform restrictions, auditing overhead, and resource-limit gaps. A limitation row mentioning broader fallback conflicts with the README's fail-closed contract. Reconcile documentation and enforcement evidence before marketing guarantees.

These observations are a planning audit, not a complete security review. Preserve current invariants: deny by default, no silent unsandboxed fallback, clear failures, and honest auditing capabilities on each backend.

### 3. “Popular default policies” need maintained upstream identities

Several older MCP reference servers are archived. The official reference repository identifies the archived group. The PostgreSQL package in the suggested setup command should not be the flagship onboarding example. [Reference servers](https://github.com/modelcontextprotocol/servers).

Datadog documented a read-only bypass in the deprecated reference PostgreSQL server. A filesystem/network sandbox cannot enforce database privileges after a permitted connection is established. Database-level restricted accounts and patched server versions remain necessary. Datadog's download counts are historical figures from its 2025 article, not present adoption measurements. [Datadog case study](https://securitylabs.datadoghq.com/articles/mcp-vulnerability-case-study-SQL-injection-in-the-postgresql-mcp-server/).

The current egress proxy's HTTP/HTTPS scope also means PostgreSQL raw TCP support must be demonstrated before a default policy is called functional. A hostname grant alone does not prove connectivity or enforcement. Add a narrowly scoped database transport route with endpoint/port restrictions, or retain an explicit unsupported/conditional label.

### 4. Host-native sandboxing already exists

VS Code documents local stdio MCP sandboxing on macOS/Linux, with explicit file/network rules. It also states that sandboxed server tool calls are auto-approved. Warden must test this interaction and avoid interpreting local sandboxing as permission to perform remote destructive operations. [VS Code MCP configuration](https://code.visualstudio.com/docs/agent-customization/mcp-servers).

Docker offers catalog setup, gateway orchestration, credential handling, container isolation, and signed images. Invariant documents tool poisoning checks, tool pinning, runtime policy and sensitive-data controls. ToolHive is another relevant MCP management/security project. These are research comparisons, not independent benchmarks or assurance claims. [Docker toolkit](https://docs.docker.com/ai/mcp-catalog-and-toolkit/toolkit/), [Invariant documentation](https://github.com/invariantlabs-ai/docs/blob/main/docs/mcp-scan/index.md), [ToolHive](https://docs.stacklok.com/toolhive/).

Inference: convenience or a badge alone will be easy to copy. Warden's stronger opportunity is a coherent combination of native isolation, portable policy enforcement, rollback, and public reproducible evidence.

### 5. A badge must describe evidence, not confer safety

Tracing observes one workload. It does not prove a server benign, discover every future access, or certify immunity from vulnerabilities. Warden's current trace runs unsandboxed; creator onboarding must not encourage running unknown packages with real credentials to obtain a badge.

Use two statuses: “Warden policy included” for a policy artifact, and “Warden checks passing” for version-specific verified test evidence. Each links to the tested artifact/version, policy hash, backend, workflow, date, and limitations. “Secured by Warden” can be the program name, but the detail page must say which claims actually hold. Signatures establish evidence provenance when identity and issuer are verified; they do not prove code safety. [Sigstore verification](https://docs.sigstore.dev/cosign/verifying/verify/).

## Cross-agent compatibility strategy

Treat this as the initial research-supported integration map, not an exhaustive list of all agents.

| Client family | Installation/configuration route | Proposed Warden integration |
|---|---|---|
| Claude Desktop | OS-specific desktop configuration; MCP bundles where supported | Local stdio wrapping; bundle distribution with backend readiness checks |
| Claude Code | Native MCP management; distinct local/project/user scopes | Separate adapter from Claude Desktop; preserve project scope and approval behavior |
| Cursor | Project/global `.cursor/mcp.json`; extension API | Reversible config adapter; later extension-based registration |
| Codex | User/project `.codex/config.toml`, MCP management | TOML-aware adapter; stdio wrapper or standard HTTP endpoint |
| VS Code/GitHub Copilot | Workspace/user/profile and agent-host configuration differ | JSONC-aware integration; preserve interpolation, trust, inputs and native sandbox settings |
| Gemini CLI | User/project `.gemini/settings.json`, native management | Preserve tool filters, env, timeouts and trust settings |
| Windsurf/Cascade | Version-specific config locations and management | Detect host version; support current and documented legacy paths rather than assume one path |
| Cline and similar IDE agents | Extension-specific MCP settings/UI | Adapter or generated configuration; retain approval behavior |
| ChatGPT and hosted MCP clients | Remote endpoint/plugin configuration | Reachable HTTPS endpoint with supported authentication |
| OpenAI Agents SDK | stdio and remote MCP connection objects | Launch Warden or point connection at Warden HTTP gateway |
| LangChain/LangGraph and Pydantic AI | Framework MCP adapters | Same transport boundaries; framework/version-specific examples |
| Other MCP-capable hosts | Standard command+args or endpoint configuration | Generic config export or manual registration, followed by verification |

Sources: [Claude Desktop setup](https://modelcontextprotocol.io/docs/develop/connect-local-servers), [MCP bundles](https://github.com/modelcontextprotocol/mcpb), [Claude Code](https://code.claude.com/docs/en/mcp), [Cursor](https://cursor.com/docs/mcp), [Codex](https://learn.chatgpt.com/docs/extend/mcp?surface=cli), [VS Code](https://code.visualstudio.com/docs/agent-customization/mcp-servers), [Gemini CLI](https://google-gemini.github.io/gemini-cli/docs/tools/mcp-server.html), [Cascade](https://docs.devin.ai/desktop/cascade/mcp), [Cline](https://docs.cline.bot/mcp/mcp-overview), [OpenAI Agents SDK](https://openai.github.io/openai-agents-python/mcp/), [LangChain](https://docs.langchain.com/oss/python/langchain/mcp), [Pydantic AI](https://pydantic.dev/docs/ai/mcp/overview/).

Publish two distinct claims: transport interoperability and tested automatic configuration. Track host version, OS/backend, transport/protocol version, upstream artifact, authentication route, workflow and last test date. Unknown clients can use generic registration, but never label them tested until a real workflow passes.

## Strengthening the three original proposals

### Policy packs

A policy pack should include upstream identity, package/image digest or pinned version, maintainer, supported backend/transport/protocol, required parameters, credential names/scopes, filesystem/network/tool grants, resource settings, positive/negative tests, evidence, review date and revocation status. Resolve paths against a defined base; do not let a catalog policy silently grant a user's entire home directory.

Keep install-time dependency fetching separate from runtime access. Pinned `npx`/`uvx` invocations may still fetch and execute dependency/install code. Prepare dependencies in an isolated install phase, verify their identity, then run offline where possible with a narrower runtime policy. A version pin alone is not a security guarantee.

Default to read-only task profiles, but explain actual grants. Filesystem packs require a selected directory; API packs require credentials scoped at the provider; database packs require restricted database accounts. Metadata such as `readOnlyHint` is untrusted unless its source is trusted. [MCP tool trust guidance](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).

Candidate order: filesystem, local Git, memory, GitHub's maintained implementation, documentation/context lookup, and search. Then Notion, Linear, Slack and database workloads after maintained implementation/endpoint verification. Browser automation, Docker-socket access and Kubernetes administration need separate privileged profiles; do not hide their broad authority in easy defaults. This is a launch hypothesis, not a measured popularity ranking. Prioritize using maintainer activity, registry presence, demand interviews, package usage with caveats, risk and support cost.

### Auto-configuration

Prefer wrapping an existing named server over reconstructing a launcher. A proposed command is `warden wrap --client cursor --server filesystem --scope project`. New-server installation is a separate proposed flow with `--name` and `--` before structured command arguments. Use explicit identifiers such as `claude-desktop` and `claude-code`.

The installation transaction should detect the relevant host/version/scope, preserve unknown fields and host interpolation, select/parameterize a pack, show a redacted grant/config diff, check backend readiness, make a restricted-permission backup, apply an atomic write, verify the MCP connection and an allowed/blocked operation, then record the protection state. If verification fails, roll back safely and never launch a direct unsandboxed fallback as part of verification.

Keep `--dry-run`, unattended `--yes`, explicit `--config`, generic export, idempotence and unwrap/restore. A second wrap must not nest Warden wrappers. Rollback must detect intervening edits and restore only the Warden-owned change. Do not change native tool approval settings. GUI environment/PATH differences, spaces, Windows quoting, symlinks and concurrent writes deserve integration tests. Never log env values, tokens or database passwords from command arguments/backups.

### Creator program

Creator flow: isolated exercise with fake credentials and representative test data → candidate policy → human review → positive and denial tests → version-bound evidence → verified CI status → README badge and installation link. A trace-derived policy starts as a candidate, never automatically trusted. Unexpected new grants after an update require review.

Registry metadata is useful for discovery, not a security endorsement: the official registry focuses on namespace authentication/metadata and leaves actual code scanning to the ecosystem. Attach evidence links through supported metadata where appropriate; do not imply official MCP certification. [Registry responsibilities](https://modelcontextprotocol.io/registry/about).

## Six implementation batches

The batches are sequential release gates, not arbitrary amounts of code. Estimates below are planning ranges in engineer-weeks, not deadlines; staffing, platform coverage and the Batch 1 audit will change them.

### Batch 1 — Establish the boundary and evidence baseline (2–3 engineer-weeks)

Goal: know exactly what each mode protects before expanding distribution.

Deliver a capability matrix for sandbox/proxy/gateway modes, platform guarantees and missing primitives; protocol-era support decisions; audited configuration/argv model; documented remote limitations; measured runtime/trace overhead; and concrete upstream smoke tests. Reconcile contradictory docs and distinguish fixture checks from end-to-end results. Review the custom TCP downstream, proxy subprocess isolation, protocol surface bypasses, and raw database transport gap.

Define local managed servers, remote managed servers and remote external servers separately. Only managed execution hosts can enforce Warden process isolation. Policy authority must come from trusted configuration; remote tool content cannot modify policy.

Release gate: a supported local workflow both succeeds and blocks out-of-policy file/network/env access; missing backend fails closed; every advertised mode has honest evidence and known limitations. Do not gate all platforms on one platform's success.

### Batch 2 — Release maintained policy packs (3–4 engineer-weeks)

Goal: complete a useful sandboxed task without writing YAML.

Start with approximately six high-value packs selected after identity/maintenance review. Implement task profiles, required parameter prompts, explicit version/digest binding, separate dependency preparation, least-privilege secret requirements and pack validation. For GitHub use the maintained server and its documented read-only mode as another layer alongside credentials and Warden policy. [GitHub server configuration](https://github.com/github/github-mcp-server/blob/main/docs/server-configuration.md).

Each pack includes a real MCP workflow plus negative enforcement tests on its advertised backend. Database packs remain conditional until raw transport and restricted role tests pass. Add release review and a revocation mechanism.

Release gate: fresh supported environment reaches a successful protected task using each released pack; no implicit whole-home/all-network grant; unavailable enforcement blocks launch; package updates cannot silently expand permissions.

### Batch 3 — Ship reversible multi-client setup (3–5 engineer-weeks)

Goal: eliminate manual config editing for local-first users.

Build adapters for Claude Desktop, Claude Code, Cursor, Codex, VS Code and Gemini CLI, plus a generic explicit-config/export path. Implement backup, atomic patch, redacted preview, readiness checks, idempotent wrap, verification and conflict-aware undo. Preserve settings outside the chosen server, credential references, host interpolation and native trust/approval controls. Use native registration APIs where they preserve the required transaction semantics.

Release gate: each advertised adapter passes detect → preview → apply → connect → allowed/blocked workflow → second wrap → undo tests; malformed config and concurrent writes produce safe failures; original settings/secrets remain intact.

### Batch 4 — Make the gateway interoperable with remote and custom agents (5–8 engineer-weeks)

Goal: let any compatible MCP host use the same enforcement core.

Add standard client-facing stdio and Streamable HTTP interfaces rather than requiring an external TCP bridge. Harden the existing upstream transport/filtering foundations. Support advertised protocol eras independently, including older sessions/server requests where required and July 2026 request metadata/MRTR behavior. Avoid a blanket “legacy SSE supported” claim without explicit old-transport tests.

For local upstreams, enforce sandboxed execution beneath the MCP filter. For remote upstreams, enforce calls/data access but explicitly state that Warden cannot sandbox the provider's machine. Cover tool invocation and resource/prompt access, dynamic discovery and server-originated interactions; reject unsupported capability paths explicitly. Define per-client policy/credential isolation. Add OAuth discovery and audience validation, separate upstream credentials, loopback/origin controls and SSRF protections for endpoints, redirects and metadata discovery. [Authorization](https://modelcontextprotocol.io/specification/2026-07-28/basic/authorization), [security guidance](https://modelcontextprotocol.io/docs/2025-11-25/tutorials/security/security_best_practices).

Add SDK recipes for OpenAI Agents SDK, LangChain/LangGraph and Pydantic AI; secondary host adapters for Cline/Cascade; and remote deployment instructions for hosted clients. A client-facing policy error must be explicit and correlated to the request, not a dropped response that hangs the agent. Cancellation must not leave managed work executing unnoticed.

Release gate: two independent HTTP clients, two stdio clients and at least one SDK complete the same allowed/blocked workflow; OAuth and cross-client isolation tests pass; advertised protocol capabilities survive filtering and streaming; unsupported combinations are labeled before launch.

### Batch 5 — Launch the evidence-backed creator program (3–4 engineer-weeks)

Goal: make creator adoption useful and trustworthy.

Deliver isolated trace/generation workflow, permission review, positive/negative test templates, reusable CI integration, signed version-bound evidence, a verification page and badge assets. Reuse the existing proof/attack harness and GitHub Action where suitable. The badge records upstream digest, policy digest, Warden version, platform/backend, test scope, timestamp and verification identity. Offer “policy included” without a certification claim; require verified tests for “checks passing.” Expired, revoked, stale or mismatched evidence must show that status.

Release gate: a contributor can reproduce verification; a changed artifact cannot reuse the previous verified badge; forged/stale/revoked evidence fails validation; no real secret is required in onboarding/CI; no badge claims complete safety.

### Batch 6 — Distribute, measure and maintain (3–5 engineer-weeks to launch; ongoing upkeep)

Goal: become an easy default that remains reliable as hosts and servers change.

Publish a searchable pack/compatibility catalog and per-host installation instructions. Distribute MCP bundles or plugin packages only where the host supports them; never imply one universal bundle works everywhere. Provide reviewed maintainer contribution templates, upgrade permission diffs and rollback, endpoint/config drift checks, release regression jobs and an issue template that redacts sensitive data.

Launch with a small pilot group, then broaden based on observed setup failures and recurring workflows. Prefer maintainer-contributed policy/CI integrations over vanity promotion. Measure install → configure → first protected workflow → repeat usage, setup time, connection failure/rollback rate, pack freshness, support workload and badge-driven verified activations. Use opt-in anonymous measurement or local reports; never collect tool bodies/credentials by default.

Release gate: all advertised integrations have fresh evidence and rollback guidance; pilot results meet the agreed activation target; release ownership and policy-review cadence are assigned. Optional targets to validate with pilots: median setup under five minutes, at least 90% successful first protected tasks within explicitly tested combinations, and zero silent unsandboxed fallback. These are proposed targets, not present results.

## Five additional features to add

1. **Protection inventory and drift detection.** Extend existing readiness checks into a view of discovered MCP connections, exact protection mode, direct bypasses and changed configs. Read-only discovery must not start unknown servers. Begin in Batch 3, complete in Batch 6.
2. **Argument-level policy and constrained approvals.** Go beyond existing tool-name allowlists: limit filesystem paths, repository owners, channels, database objects and permitted operation parameters. Sensitive one-time approvals bind to exact arguments, actor and expiry; unattended mode denies unresolved approval requests. Rules fail closed on missing fields/unknown tools. Add in Batch 4; provider-side privileges remain essential.
3. **Credential broker with minimal exposure.** Extend environment allowlisting into OS-keychain/secret-store references, per-upstream credentials, short-lived grants and rotation/revocation. Inject secrets only into the selected managed boundary. Arbitrary third-party servers may still need tokens; do not promise token invisibility without protocol/provider support. Start in Batch 2, expand in Batch 4.
4. **Dependency and tool-definition change detection.** Pin package/image identity and reviewed tool schema/description hashes. New tools or schema changes require review; declared read-only annotations cannot expand grants. Combine signatures/advisories with revocation and safe rollback. Heuristic poisoning detectors can complement deterministic controls, but cannot guarantee injection prevention. Start in Batch 2, complete in Batch 5.
5. **Privacy-preserving incident record and emergency stop.** Correlate host, connection, upstream, policy version, decision, reason and timestamp; redact arguments/results by default. Stop managed processes, close/revoke Warden sessions and credentials, and export sanitized evidence. Explicitly report that Warden cannot cancel independent provider-side effects or direct bypasses. Add in Batch 4, improve reporting in Batch 6.

## Practical order and unresolved questions

Ship the first six useful packs and reversible client setup before spending heavily on badge promotion. Standard gateway interoperability is the route to broad MCP coverage; creator evidence multiplies distribution once enforcement is trustworthy. Keep Batch 1's assurance work as a release prerequisite throughout.

Decisions needed before implementation: native Windows elevation/user experience; supported macOS sandbox availability; database raw TCP approach; oldest supported protocol era; initial pack maintenance owners; HTTP gateway deployment/auth model; evidence hosting/revocation model; and acceptable auditing overhead. None requires a coding decision in this research phase.

The research supports the architecture and identifies current integration routes. It does not establish a universal agent count, a current popularity ranking, verified Warden support for every host, or a security certification. Recheck fast-changing host docs and actual pinned artifacts at implementation time.
