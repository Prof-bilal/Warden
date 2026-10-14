# Warden End-to-End Proof Results

Run date: 2026-10-14
Host: Linux x86_64, kernel 7.2.5-3-omarchy
Warden source binary: `dev`
Published npm binary: `v0.1.16`

## Passed

- Published installation: `npm install -g warden-sandbox-cli` completed.
- Published `warden doctor`: READY with bubblewrap, proxy, strace, policy
  engine, filesystem isolation, environment filtering, and fail-closed checks.
- Native Linux proof: 8/8 checks passed.
- Docker proof: 8/8 checks passed.
- Native Linux attack simulations: 7/7 contained.
- Docker attack simulations: 7/7 contained.
- MCP proxy contract: 6/6 passed.
- Full Go test suite: all packages passed with host socket permissions.
- Pinned real filesystem MCP server: passed initialize, discovery, allowed read,
  denied write, denied outside read, wrapper apply, idempotency, and undo.
- MCP growth smoke: passed raw stdio, authenticated HTTP, official SDK stdio,
  official SDK HTTP, signed evidence verification, artifact-drift refusal, and
  emergency stop.
- CI policy checks: valid policies passed, generated policies validated, and
  invalid policy rejection passed.
- npm wrapper runtime test and package dry-run passed.
- Landing site production build passed: 104 static pages generated.
- Kubernetes policy validation, Docker command generation, and YAML rendering
  completed. The rendered output contains Deployment, NetworkPolicy, and
  SeccompProfile documents with non-root, read-only root, dropped capabilities,
  RuntimeDefault seccomp, and memory limits.

## Kubernetes failures found

The first live kind-cluster run found three renderer issues, all now fixed:

1. NetworkPolicy selectors now render as `spec.podSelector.matchLabels`.
2. The optional Security Profiles Operator `SeccompProfile` CRD is no longer
   emitted; the Deployment uses standard Kubernetes `RuntimeDefault` seccomp.
3. NetworkPolicy ports now render as numeric values rather than quoted strings.

The corrected renderer produced valid Deployment and NetworkPolicy documents.
The live kind API accepted both resources and the expected security settings.
The pod remained `ErrImagePull` only because `warden-e2e:local` was a placeholder
image, not because of manifest validation. The disposable cluster and namespace
were deleted after the retest.

## Remaining external dependencies

- GitHub, PostgreSQL, Kubernetes-MCP, and search MCP accounts were not
  available as environment credentials, so no real external account calls were
  attempted.
- macOS Seatbelt and Windows AppContainer/WFP require real separate machines.
- GitHub-hosted CI requires an authenticated `gh` installation or a workflow
  dispatch from the repository.

## Evidence

- Installed proof: `/tmp/warden-e2e/proof-installed/linux/20261014-141500`
- Native attack evidence:
  `/tmp/warden-e2e/attacks-native/linux/20261014-141345/attacks`
- Docker proof: `/tmp/warden-e2e/proof-docker/linux/20261014-141641`
- Docker attack evidence:
  `/tmp/warden-e2e/attacks-docker/linux/20261014-141642/attacks`
- Full Go test log: `/tmp/warden-e2e/go-test-full.log`
- Kubernetes render artifacts: `/tmp/warden-e2e/k8s-render`
- Pinned MCP runtime: `/tmp/warden-e2e/mcp-runtime`

## Important version note

The published npm binary (`v0.1.16`) installed successfully and passed the
core proof, but it does not expose newer source commands such as `packs`. The
source-built binary contains those commands. Release/package parity should be
verified before claiming that the published install represents the entire
current checkout feature surface.
