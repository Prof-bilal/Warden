# Agent isolation threat model

## Scope

Warden treats the coding agent, MCP servers, repository scripts, package install
scripts, language servers, and all descendants as untrusted code. The host kernel,
native sandbox primitive, Warden launcher, policy parser, and host-side egress proxy
are trusted components.

## Assets

API keys, cloud credentials, SSH keys, Git credentials, private source, deployment
configuration, local user files, browser/session credentials, and environment
variables are protected assets. Fake values only are used in tests.

## Trust boundaries

```text
LLM -> agent harness -> tools/MCP/child processes -> Warden launcher
    -> OS sandbox + environment filter + egress proxy -> host resources
```

The model and agent runtime may request any operation. Warden must make the final
decision independently of their instructions or claimed permissions.

## Threat actors

- prompt-injected or compromised agent;
- malicious MCP server or repository script;
- compromised dependency or package install hook;
- accidental agent behavior;
- malicious helper process or language server.

## Security invariant

For a supported backend, if policy does not grant an agent a resource, neither the
agent nor a descendant may obtain it through an alternate path, environment
inheritance, helper process, or direct network route. This excludes kernel/admin
compromise and data intentionally granted to an allowed endpoint.

## Known limits

Warden cannot revoke a host descriptor opened before the sandbox unless it closes or
filters that descriptor before launch. It cannot prevent exfiltration to an endpoint
that policy explicitly allows. Remote MCP processes require proxy mediation rather
than local process isolation. Unsupported native capabilities must cause a loud
refusal or an explicitly selected Docker fallback.
