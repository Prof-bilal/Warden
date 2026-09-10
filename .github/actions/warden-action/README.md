# Warden GitHub Action

A GitHub Action that runs commands inside the Warden sandbox with security policy enforcement. Perfect for running AI agents, MCP servers, or any untrusted code in CI/CD pipelines with controlled permissions.

## Features

- **Fail-closed security**: Commands never run unsandboxed, even if setup fails
- **Policy enforcement**: Filesystem, network, and environment access controlled by YAML policy
- **Audit logging**: All access attempts logged, including blocked operations
- **CI/CD optimized**: Designed for Linux GitHub runners with minimal overhead
- **Secret protection**: Environment variables filtered through policy, preventing accidental exposure

## Quick Start

```yaml
name: Secure AI Review
on: [pull_request]

jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run AI code review
        uses: ./.github/actions/warden-action
        with:
          policy: ./ci-policy.yaml
          run: claude -p "Review this PR for security issues" --max-turns 3
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `policy` | Path to Warden policy.yaml file | Yes | - |
| `run` | Command to run inside the sandbox | Yes | - |
| `backend` | Sandbox backend (auto, linux, docker) | No | `auto` |
| `approve` | Enable interactive approval mode | No | `false` |
| `working-directory` | Working directory for the command | No | `.` |

## Outputs

| Output | Description |
|--------|-------------|
| `exit-code` | Exit code of the sandboxed command |

## Security Model

Warden Action follows a **fail-closed** security model:

1. ✅ **Policy required**: No command runs without an explicit policy
2. ✅ **Validation first**: Policy and environment validated before execution  
3. ✅ **Blocked by default**: Only explicitly granted access is allowed
4. ✅ **Audit everything**: All access attempts logged
5. ✅ **No fallbacks**: Failed sandbox setup prevents execution