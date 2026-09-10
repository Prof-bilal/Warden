# Warden CI/CD Examples

This directory contains complete examples of using Warden in CI/CD pipelines.

## Directory Structure

```
examples/
├── ai-code-review/          # AI-powered code review workflow
├── secure-build/            # Secure build pipeline
├── security-scanning/       # Security tools with sandboxing
├── multi-language/          # Examples for different languages
└── integration-tests/       # Full integration test examples
```

## Quick Examples

### 1. AI Code Review (Claude)

Complete setup for AI-powered PR reviews with security sandboxing:

- **Workflow**: `.github/workflows/ai-review.yml`
- **Policy**: `.github/policies/ai-review.yaml` 
- **Features**: Secret protection, network isolation, audit logging

### 2. Secure Node.js Build

Standard Node.js build with Warden security:

- **Workflow**: `.github/workflows/secure-build.yml`
- **Policy**: `.github/policies/node-build.yaml`
- **Features**: Package registry access, build isolation, resource limits

### 3. Security Scanning

Run security tools (gitleaks, semgrep) safely:

- **Workflow**: `.github/workflows/security-scan.yml`  
- **Policy**: `.github/policies/security-scan.yaml`
- **Features**: Full repo access, controlled network, safe reporting

## Usage

1. **Copy examples** to your repository
2. **Customize policies** for your specific needs  
3. **Adjust workflows** to match your build process
4. **Test thoroughly** before production use

## Policy Templates

Pre-built policies for common scenarios:

- `minimal.yaml` - Basic template
- `ai-review.yaml` - AI agent workflows
- `build.yaml` - Build processes
- `security-scan.yaml` - Security tools
- `deploy.yaml` - Deployment workflows

## Best Practices

See [CI Integration Guide](../docs/ci-integration.md) for detailed best practices and troubleshooting.