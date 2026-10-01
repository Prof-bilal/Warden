# Batches 4–6: development implementation

September 30, 2026. Website and CLI implemented together. This checkpoint delivers
working features; it does not certify every research-plan release gate or every AI agent.

Release follow-up: [0.2.0-beta.1 preparation](RELEASE_PREVIEW_STATUS.md) records
verified npm packaging/downloads, version negotiation and actual Codex Linux
app-server workflows. The platform and GUI limits below still apply.

## Batch 4: standard gateway

- `connect`: newline-delimited stdio with MCP rules above digest-pinned sandboxed run.
- `serve`: authenticated Streamable HTTP POST/DELETE at `/mcp`, numeric loopback
  binding, strict authority/Origin controls, per-principal isolated sessions,
  bounded size/time/session count, idle expiry and cancellation cleanup.
- Local upstreams always launch Warden run; no direct fallback. Remote upstreams
  get filtered requests, independent sessions and separate credentials. HTTPS,
  public-IP dialing, redirect refusal and explicit loopback test opt-in prevent
  ordinary endpoint/DNS-rebinding SSRF paths.
- Exact tool argument constraints, tool-definition pins refreshed before calls,
  filtered tool/resource/prompt discovery and correlated denials. Unknown method
  names are normalized in the payload-free gateway journal.
- Existing-issuer OAuth protected-resource metadata and trusted-local-JWKS RS256
  checks for signature, issuer, audience, expiry/not-before, subject and mcp scope.
- Private `stop` control closes managed connections; no provider-side undo promise.
  Private journals, controls and signing keys are rejected inside resolved grants,
  executable directories and readable runtime paths, including ancestor symlinks.
- Cline CLI/extension and explicit legacy Cascade adapters, plus SDK/deployment guides.
- Website `/integrations`: local rules download and structured connection builder;
  compatibility/protection/setup flows updated.

Supported scope is the **2025-11-25 request/response subset**. 2026-07-28/MRTR,
tasks, subscriptions, pagination, server sampling/elicitation/roots and independent
notification streams are refused. No full protocol conformance is claimed. Dynamic
OAuth discovery/registration, consent, remote JWKS rotation/revocation and upstream
OAuth exchange remain release work. Different policies/providers require separate
gateway instances. Windows process cancellation and macOS/Windows gateway behavior
require native tests. Python SDK recipes and GUI hosts remain live-workflow-unverified.

## Batch 5: creator evidence

- `creator init/keygen/check/verify`: disposable kit, owner-private issuer key,
  sandboxed allowed/denied checks, per-check liveness controls and Ed25519 evidence.
- Binds artifact tree (including internal symlinks), policy, rules, test manifest,
  command, Warden binary/version, native platform/backend, protocol/transport,
  issuer, scope and expiry. Artifact/config changes during checks stop signing.
- Independent trusted public key and fresh trusted local revocation snapshot
  required; invalid/expired/revoked/mismatched inputs cannot produce a passing badge.
- Separate policy-included and checks-passing SVGs, CI template, creator guide,
  `/creators` and local-only `/verify` signature/expiry inspector.

Signatures establish issuer provenance, not complete safety. System interpreter/
base-runtime identity is still platform trust. The browser inspector cannot check
installed artifacts or revocation and labels that limitation. Automatic issuer
federation, trusted hosted evidence/revocation and production creator enrollment
remain release work. Sample badge assets alone confer no verified status. The CI
fixture key is ephemeral and never presented as maintainer identity.

## Batch 6: distribution and maintenance

- `policy-diff`: conservative grant changes, command drift, expanded limits and
  legacy MCP authorization changes marked for review without printing argv secrets.
- Explicit local enum events and `report summary`: durations, setup failure fraction
  and funnel event counts; no network telemetry or conversation collection.
- `/maintenance`: policy/rollback instructions and in-browser local report summary.
- Updated navigation, discoverability, sitemap and dual website/Go docs.
- Policy/adapter contribution template and real-sandbox Linux CI regression.

Automatic remote endpoint monitoring, full config/host discovery, update/revocation
distribution, reviewed per-host MCP bundles/plugins, named maintenance owners and
a live-user pilot remain release gates. Local event counts are not unique users
or measured production activation. Existing inventory covers configured entries,
not every possible inherited plugin or direct bypass.

## Validation

The actual CLI passed allowed reads, filtered-tool denial, OS write/outside-read
denial, and discovery filtering through a handwritten stdio host and HTTP client
plus the official TypeScript MCP SDK over both transports. Signed creator checks,
trusted verification, changed-artifact refusal and authenticated stop passed on
native Linux with disposable fixtures. No model API call or real credential was used.

Gateway tests cover argument/definition drift, case-alias and duplicate-key attacks,
audit failure, OAuth claims, principal isolation, origins, SSRF/redirect refusal and
scoped SSE upstream responses. Attestation tests cover forged/untrusted, expired,
revoked, stale and changed-input evidence; artifact external symlinks are refused.
Maintenance tests cover permission expansions and malformed/private local reports.

- Final `go build ./...`, `go vet ./...`, and `go test ./...` passed in the required order.
- Race checks passed for the gateway, attestation and maintenance packages. Ten
  consecutive gateway race runs covered stalled-write cancellation and successful
  notification cleanup without terminating a healthy process.
- All 21 native Linux sandbox tests executed and passed; none skipped.
- Windows amd64 and macOS arm64 cross-builds passed. These do not establish native
  runtime behavior on those platforms.
- The final real CLI harness passed all six workflow gates, including raw clients
  and the official TypeScript MCP SDK over stdio and HTTP, creator verification,
  changed-artifact rejection and stop cleanup. The pinned SDK runtime was prepared
  using Warden's sandboxed installer.
- `npm run build` passed TypeScript and generated 91 pages. Turbopack now uses the
  repository root to resolve the website/CLI shared policy catalog.
- The strict MkDocs build and `git diff --check` passed.

Browser inspection remains denied by the browser security policy from this session;
visual layout and interactive browser behavior are unverified. The browser denial
was not bypassed. No real MCP host configuration was edited, and nothing was deployed,
published or committed. Earlier batch changes and unrelated workspace files remain.
