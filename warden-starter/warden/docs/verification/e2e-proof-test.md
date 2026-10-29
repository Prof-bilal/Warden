# Warden End-to-End Proof Test

Short runbook for testing Warden later on a real machine. The first complete
run should be performed on Linux. Friends can use the same checklist on macOS
or Windows and return the generated report.

## Safety

- Use only disposable test accounts, repositories, databases, and Kubernetes
  namespaces.
- Never use production credentials or real personal files.
- The attack tests use fake secrets, decoy documents, loopback collectors, and
  reversible simulations. Do not run real malware.
- Save the generated evidence, but redact tokens, usernames, home paths, and
  private hostnames before sharing it.

## 1. Install from the landing page

Open the published Warden landing page and copy the displayed install command.
The expected npm command is:

```bash
npm install -g warden-sandbox-cli
warden --version
warden doctor
```

Record:

- Landing-page URL and command used.
- Warden version and package version.
- Operating system, kernel, architecture, and hostname.
- Backend reported by `warden doctor`.
- Any missing prerequisites.

On Linux, confirm the native prerequisites:

```bash
command -v bwrap
command -v strace
docker info
```

## 2. Run the built-in proof

From the Warden module directory:

```bash
cd warden-starter/warden
go build -o warden ./cmd/warden
bash testdata/proof/run-proof.sh ./warden
```

Expected result: the proof reports all steps as passed and writes evidence
under `evidence/` (or under the directory selected by `WARDEN_PROOF_OUT`).

## 3. Run safe attack simulations

```bash
cd warden-starter/warden
WARDEN_BACKEND=docker bash testdata/attacks/run-attacks.sh ./warden
```

Use `WARDEN_BACKEND=linux` for the native Linux run when bubblewrap and user
namespaces are available.

Expected result: every control attack lands without Warden, and every matching
sandboxed attack is contained. A `VOID` result means the control did not prove
the fixture could perform the attempted action.

Review the generated attack evidence, especially:

- `summary.json`
- `evidence.md`
- `collector.log`
- vault hash comparisons
- sandbox audit logs

## 4. Test MCP behavior

Start with the harmless local MCP fixture and verify:

1. MCP initialize succeeds.
2. `tools/list` succeeds.
3. An allowed file read succeeds.
4. An allowed write succeeds.
5. A read outside the policy fails.
6. A write outside the policy fails.
7. A blocked network request fails.
8. The decision appears in the audit log.

Then test disposable real integrations where available:

- Filesystem MCP server.
- GitHub MCP server with a temporary repository and token.
- PostgreSQL MCP server backed by a disposable Docker database.
- Kubernetes MCP server inside a disposable kind/minikube namespace.
- One disposable network/search integration.

Do not mark an integration as passing unless both an allowed call and a denied
call are observed.

## 5. Test the MCP proxy and gateway

```bash
bash warden-starter/warden/scripts/test-mcp-proxy.sh \
  warden-starter/warden/warden
```

Verify that allowed tools are forwarded, denied tools return JSON-RPC errors,
invalid input is rejected, and decisions are audited.

For the gateway, generate a policy and wrapped client configuration, then
launch one local MCP server through the wrapper and repeat the allowed/denied
tool checks.

## 6. Test Kubernetes output

The renderer emits portable Kubernetes resources: a `Deployment` and a
`NetworkPolicy`. Seccomp is configured through the standard Kubernetes
`RuntimeDefault` security context; no Security Profiles Operator CRD is
required.

Run the CLI smoke checks:

```bash
cd /path/to/Warden
export PATH="$PWD/warden-starter/warden:$PATH"
bash warden-starter/warden/scripts/test-k8s-mode.sh
```

For a live API check, use a disposable kind or minikube cluster and a real
container image that exists in the cluster:

```bash
kind create cluster --name warden-e2e
kubectl create namespace warden-e2e

warden k8s render \
  --policy warden-starter/warden/examples/container-policy.yaml \
  --image warden-e2e:local \
  --namespace warden-e2e \
  --output /tmp/warden-e2e-manifest.yaml

kubectl apply --context kind-warden-e2e \
  --namespace warden-e2e \
  -f /tmp/warden-e2e-manifest.yaml
kubectl get deployment,networkpolicy --namespace warden-e2e
```

Confirm that:

- `NetworkPolicy.spec.podSelector.matchLabels` contains the application label.
- NetworkPolicy ports are numeric (`53`, `80`, `443`), not quoted strings.
- No `kind: SeccompProfile` document is emitted.
- Pod and container security contexts use `RuntimeDefault` seccomp.
- The workload is non-root, has a read-only root filesystem, drops
  capabilities, and has resource limits.

If the image is only a placeholder, `ErrImagePull` is expected; this does not
invalidate manifest/API validation. Use a locally loaded image or a published
test image to verify pod readiness and runtime behavior. Delete the disposable
resources afterward:

```bash
kubectl delete namespace warden-e2e
kind delete cluster --name warden-e2e
```

## 7. Test CI/CD

Run the Warden GitHub Action workflow with:

- One valid policy and successful command.
- One allowed network operation.
- One blocked network operation.
- One blocked filesystem operation.
- One invalid policy that must fail.
- One malformed policy that must fail.

Save the workflow URL, commit SHA, runner OS, backend, exit status, and
uploaded evidence artifact.

## 8. Cross-platform friend test

Each tester should run:

1. Landing-page installation.
2. `warden --version` and `warden doctor`.
3. The proof harness.
4. The safe attack harness.
5. The MCP fixture test.
6. Kubernetes checks if kind/minikube is available.

Platform notes:

- Linux: native bubblewrap proof is preferred; Docker is the fallback.
- macOS: run the native Seatbelt backend on real hardware.
- Windows: native AppContainer/WFP testing may require administrator access and
  suitable hardware; hosted CI results do not replace this test.

## 9. Report template

Return a redacted `report.md` containing:

```text
Tester:
Date:
OS / version:
Architecture:
Warden version:
Backend:

Installation: PASS / FAIL
Doctor: PASS / FAIL
Proof harness: PASS / FAIL
Attack simulations: PASS / FAIL / VOID
MCP fixture: PASS / FAIL
Real MCP integrations:
Proxy/gateway: PASS / FAIL
Kubernetes: PASS / FAIL / SKIP
CI/CD: PASS / FAIL / SKIP

Failures or limitations:
Evidence location or artifact URL:
```

The final project report should map each product claim to its test, environment,
evidence, result, and limitation. A green unit test or a skipped backend is not
enough to claim that platform is end-to-end verified.

See [E2E proof results](e2e-proof-results.md) for the latest Linux, Docker,
MCP, CI, and Kubernetes results.
