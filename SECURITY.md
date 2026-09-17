# Security Policy

Warden is a security tool: its entire purpose is to contain untrusted processes. A bug in Warden that lets a sandboxed process escapeor makes Warden run anything unsandboxedis by definition a security vulnerability.

## Supported versions

| Version | Supported |
|---|---|
| latest release ([Releases](https://github.com/Prof-bilal/Warden/releases)) | ✅ |
| older releases | ❌please upgrade |

Warden is pre-1.0 and moves fast; security fixes land on the latest release only.

## Reporting a vulnerability

**Please do not open a public issue for a security problem.**

Report privately via [GitHub Security Advisories](https://github.com/Prof-bilal/Warden/security/advisories/new) ("Report a vulnerability"). If you prefer email, open the advisory and note thatthe maintainers will follow up.

Include whatever you have:

- Warden version (`warden version`) and platform
- The policy file used (redact any secrets)
- Steps or a script to reproduce the escape/bypass
- Audit log excerpt (`~/.local/state/warden/audit.jsonl`) if relevant

## What counts as a security issue

In scope:

- **Sandbox escapes**a sandboxed process reading, writing, or contacting anything outside its policy grants
- **Fail-open behavior**any code path where Warden executes a command unsandboxed (or with partial enforcement) instead of refusing
- **Policy bypasses**malformed policies, symlink tricks, env leakage, DNS leakage past the allowlist
- **Audit integrity**blocked accesses that silently go unlogged where the docs promise they are logged

Out of scope:

- Bugs where sandboxing works as documented but is too strict (usability)
- The documented limitations in the [threat model](warden-starter/warden/docs/security.md#known-limitations) (e.g. symlinks inside granted paths, HTTP/HTTPS-only proxy interception)though improving them interests us, they are known and disclosed
- Attacks requiring the user to disable the sandbox themselves

## What to expect

- **Acknowledgment** within 72 hours
- **Triage + severity estimate** within 7 days
- **Fix or mitigation plan**escape-class bugs are treated as the highest priority
- **Credit** in the release notes if you want it (say so in the report)

## Safe harbor

We consider good-faith research on Warden's sandbox to be authorized: if you isolate your testing to your own machines and data, avoid degradation of the project's services (CI, releases, docs site), and follow this policy, we will not pursue legal action against you for the research or disclosure.

## Security-relevant documentation

- [Threat model & known limitations](warden-starter/warden/docs/security.md)the 7-category threat model, the honest limitations list, and backend-specific caveats
- [Fail-closed guarantees](ARCHITECTURE.md)what Warden refuses to do when primitives are missing
- [Proof harness](warden-starter/warden/testdata/proof/run-proof.sh)reproducible end-to-end sandbox verification with committed evidence files
