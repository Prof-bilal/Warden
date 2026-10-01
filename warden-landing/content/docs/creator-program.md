# Creator evidence and badges

The "Secured by Warden" program offers two precise claims: **policy included**
and **checks passing**. Neither proves complete safety or prompt-injection immunity.
The bundled badges are examples; a badge image alone carries no verification.
Link the README badge to version-specific artifacts and reproducible evidence.

## Generate a candidate kit

```bash
warden creator init --output ./warden-kit
warden creator keygen --output /private/issuer.key
```

The kit contains rules, allowed/denied check templates and a policy-included badge.
Prepare a pinned runtime separately, select disposable data, generate/review its
process policy, and edit the fixture paths. This workflow exercises the server
inside the sandbox; it does not invoke unsandboxed trace. If you separately trace,
use trusted code and disposable fake credentials. Trace output remains a candidate.

## Exercise and sign

```bash
warden creator check --policy policy.yaml --rules rules.json --artifact /prepared/runtime --manifest checks.json --key /private/issuer.key --output evidence.json --backend linux -- /absolute/node /prepared/runtime/server.js
```

The manifest requires at least one allowed and one denied tools/resources/prompts
request. Allowed checks may require a known fixture result substring; application
errors do not pass. Denied checks must hit a Warden policy denial, not an unavailable
process or arbitrary upstream error. A ping control after every check proves the
server remains alive. Failures produce no signed report. The artifact tree must
remain unchanged throughout checks. Keep the signing key outside the artifact and
all upstream filesystem grants, the executable's directory and readable runtime
paths. Existing ancestor symlinks are resolved when checking that boundary. On
macOS, keep private state outside host temporary directories too. Use a trusted
Warden build and a dedicated issuer.

Ed25519 evidence binds artifact files/internal symlinks, policy, rules, test manifest,
launcher, Warden binary/version, platform/backend, stdio transport, protocol, scope,
issuer fingerprint, test results and expiry. External artifact symlinks are refused.
System interpreter/base-runtime identity still belongs to platform trust; this is
not hermetic supply-chain attestation. Evidence expires after seven days.

## Verify from independent trust

```bash
warden creator verify --policy policy.yaml --rules rules.json --artifact /prepared/runtime --manifest checks.json --public-key /trusted/issuer.key.pub --revocations /trusted/revocations.json --evidence evidence.json --badge checks-passing.svg -- /absolute/node /prepared/runtime/server.js
```

Verification recomputes every binding, checks the trusted signature, platform,
expiry and passing allowed/denied results. Changed code, policy, rules, launcher,
manifest or Warden binary cannot reuse evidence. It refuses unknown issuers,
expired/revoked/mismatched evidence and missing or expired revocation snapshots.
It writes a checks-passing badge only after successful verification.

Distribute the public key through an independently trusted channel. A public key
uploaded with a report establishes no trusted identity on its own. The verifier
trusts an explicit local revocation snapshot; provenance/distribution of that
snapshot is your responsibility. Example format (replace the expiry):

```json
{"version":1,"expires_at":"REPLACE_WITH_FUTURE_UTC_TIMESTAMP","revoked_evidence":[],"revoked_artifacts":[]}
```

Evidence revocation IDs are SHA-256 of the decoded signed payload; artifact IDs
are the recorded artifact tree hash. Offline verification cannot promise current
revocation beyond the trusted snapshot. Managed remote revocation distribution,
Sigstore/GitHub identity federation and public evidence hosting remain release work.

The [evidence inspector](/verify) checks a signature/expiry locally with the trusted
public key you supply. It cannot inspect your runtime or revocation source, and
never labels that limited check full verification. Nothing is uploaded.

## README and CI

```markdown
[![Warden: policy included](warden-kit/policy-included.svg)](warden-kit/README.md)
[![Warden: checks passing](checks-passing.svg)](evidence/README.md)
```

The evidence README should name the artifact/version, policy and rules digests,
Warden/backend/platform, tested workflow, issuer trust channel, dates, revocation
source and verification command. Retain ordinary tool approval controls.

The repository's Linux CI runs the disposable gateway/creator regression harness
after proving the sandbox works. `testdata/ecosystem/smoke-growth.mjs` exercises
two stdio clients and two HTTP clients, including the official TypeScript MCP SDK.
Its ephemeral fixture issuer does not represent maintainer identity. A reusable
creator workflow template lives in `templates/creator-check.yml`; review and adapt
its runtime preparation and fake fixtures. Never expose a production signing key
to untrusted pull-request code. Enroll release issuer trust separately after review.
