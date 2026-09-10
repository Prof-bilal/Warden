# Warden CI Policy Integration

This directory contains policy integration features for the Warden GitHub Action, including policy generation from traces and secrets hygiene validation.

## Policy Generation Workflow

The CI action supports generating policies from traced executions:

1. **Trace Mode**: Run commands with `warden trace` to record access patterns
2. **Policy Generation**: Use `warden init` to create policy from trace
3. **Validation**: Ensure generated policy follows CI best practices

## Secrets Hygiene

The action enforces strict secrets hygiene:

- Only explicitly allowed environment variables are passed to sandboxed processes
- Policy validation checks for common secret leakage patterns
- Audit logs track any attempts to access blocked environment variables

## Usage in Workflows

### Generate Policy from Trace

```yaml
- name: Generate policy from trace
  run: |
    # Trace the command first
    warden trace -- your-command --with args
    
    # Generate policy from trace
    warden init --output ci-policy.yaml
    
    # Review and adjust the generated policy
    cat ci-policy.yaml
```

### Validate Policy Security

```yaml
- name: Validate policy security
  run: |
    # Check for potential secret exposures
    python3 .github/actions/warden-action/scripts/validate-policy.py ci-policy.yaml
```

## Best Practices

1. **Minimal Privileges**: Only grant the minimum permissions needed
2. **Explicit Environment**: Never use wildcards in `env.allow`
3. **Network Restrictions**: Limit network access to required endpoints only
4. **Audit Review**: Regularly review audit logs for policy violations

## Files

- `scripts/validate-policy.py` - Policy security validation
- `scripts/generate-ci-policy.sh` - CI-specific policy generation
- `templates/` - Common policy templates for CI use cases