# Warden Container & Kubernetes Mode

Transform Warden policies into production-ready container security configurations. The same `policy.yaml` you test locally becomes Docker security options and Kubernetes SecurityContext manifests.

## Overview

**Development → Production Pipeline:**
```
Local Development    Container Deployment    Kubernetes Production
       │                      │                       │
   policy.yaml  ────────► docker run cmd  ────────► K8s manifests
       │                      │                       │
   warden run          --read-only            readOnlyRootFilesystem
   --policy            --cap-drop ALL        securityContext
                       --tmpfs /tmp          NetworkPolicy
```

## Quick Start

### 1. Create Container Policy

```yaml
# container-policy.yaml
command: ["node", "/app/server.js"]

filesystem:
  read: ["/app", "/usr/lib", "/lib"]
  write: ["/tmp", "/app/logs"]

network:
  allow: ["api.github.com"]

env:
  allow: ["NODE_ENV", "GITHUB_TOKEN"]

limits:
  memory_mb: 512
```

### 2. Generate Docker Command

```bash
warden k8s docker --policy container-policy.yaml --image myapp:latest

# Outputs:
# docker run --read-only --security-opt no-new-privileges:true \
#   --cap-drop ALL --tmpfs /tmp \
#   -v /app:/app:ro --memory 512m \
#   myapp:latest node /app/server.js
```

### 3. Generate Kubernetes Manifests

```bash
warden k8s render --policy container-policy.yaml --image myapp:latest

# Generates:
# - Deployment with SecurityContext
# - NetworkPolicy for egress control  
# - SeccompProfile for syscall filtering
```

## Policy Translation Reference

### Filesystem → Volume Configuration

| Warden Policy | Docker | Kubernetes |
|---|---|---|
| `filesystem.read: ["/app"]` | `-v /app:/app:ro` | `hostPath` volume + `readOnly: true` |
| `filesystem.write: ["/tmp"]` | `--tmpfs /tmp` | `emptyDir` volume with `Memory` medium |
| `filesystem.write: ["/data"]` | `-v /data:/data` | `hostPath` volume + `readOnly: false` |

### Network → Network Policy

| Warden Policy | Docker | Kubernetes |
|---|---|---|
| `network.allow: []` | `--network none` | `NetworkPolicy` with empty egress rules |
| `network.allow: ["api.github.com"]` | `--network bridge`* | `NetworkPolicy` with specific egress rules* |

*Note: Docker and K8s have FQDN filtering limitations - see [Network Limitations](#network-limitations) below.

### Environment → Container Configuration

| Warden Policy | Docker | Kubernetes |
|---|---|---|
| `env.allow: ["NODE_ENV"]` | `-e NODE_ENV` | `env: [{name: NODE_ENV}]` |
| `env.allow: ["SECRET"]` | `-e SECRET` | `envFrom: secretRef` (recommended) |

### Limits → Resource Controls

| Warden Policy | Docker | Kubernetes |
|---|---|---|
| `limits.memory_mb: 512` | `--memory 512m` | `resources.limits.memory: 512Mi` |
| `limits.timeout_s: 300` | N/A | `activeDeadlineSeconds: 300` |

## Generated Security Controls

### Docker Security Options

```bash
# Generated docker run command includes:
--read-only                    # Read-only root filesystem
--security-opt no-new-privileges:true  # Prevent privilege escalation
--cap-drop ALL                 # Drop all capabilities
--tmpfs /tmp:rw,noexec,nosuid,nodev    # Safe temporary storage
--user 1000:1000              # Non-root user
--memory 512m                  # Memory limit
--network none                 # Network isolation (if no network.allow)
```

### Kubernetes Security Context

```yaml
# Generated SecurityContext includes:
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault

# Pod-level security:
podSecurityContext:
  runAsNonRoot: true
  fsGroup: 1000
```

## Commands Reference

### `warden k8s render`

Generate complete Kubernetes manifests:

```bash
warden k8s render --policy policy.yaml --image myapp:latest [options]

Options:
  --namespace <ns>      K8s namespace (default: default)
  --output <file>       Output file (default: stdout)
  --platform <target>   Target platform (kubernetes, openshift)

Examples:
  warden k8s render --policy app.yaml --image myapp:v1.2.3
  warden k8s render --policy app.yaml --image myapp --namespace production
  warden k8s render --policy app.yaml --image myapp --output k8s-manifests.yaml
```

**Generated Resources:**
- `Deployment` with security contexts and volume mounts
- `NetworkPolicy` for egress traffic control
- `SeccompProfile` for syscall filtering (if security-profiles-operator available)

### `warden k8s docker`

Generate Docker run command:

```bash
warden k8s docker --policy policy.yaml --image myapp:latest [options]

Options:
  --platform <target>   Target platform (docker, podman)

Examples:
  warden k8s docker --policy app.yaml --image myapp:latest
  warden k8s docker --policy app.yaml --image myapp --platform podman
```

### `warden k8s validate`

Check policy compatibility:

```bash
warden k8s validate --policy policy.yaml

# Checks for:
# - FQDN network rules (K8s limitation)
# - Filesystem wildcard patterns
# - Resource limit recommendations
# - MCP configuration conflicts
```

## Example Manifests

### Secure MCP Server Deployment

**Input Policy:**
```yaml
command: ["node", "server.js"]
filesystem:
  read: ["/app"]
  write: ["/tmp"]
network:
  allow: ["api.github.com"]
env:
  allow: ["GITHUB_TOKEN", "NODE_ENV"]
limits:
  memory_mb: 256
```

**Generated Deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mcp-server
  labels:
    warden.security/managed: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mcp-server
  template:
    metadata:
      labels:
        app: mcp-server
    spec:
      securityContext:
        runAsNonRoot: true
        fsGroup: 1000
      automountServiceAccountToken: false
      containers:
      - name: mcp-server
        image: myapp:latest
        command: ["node", "server.js"]
        securityContext:
          runAsNonRoot: true
          readOnlyRootFilesystem: true
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
        resources:
          limits:
            memory: "256Mi"
          requests:
            memory: "128Mi"
        volumeMounts:
        - name: app-code
          mountPath: /app
          readOnly: true
        - name: tmp-storage
          mountPath: /tmp
        env:
        - name: NODE_ENV
          value: "production"
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: github-token
      volumes:
      - name: app-code
        hostPath:
          path: /app
      - name: tmp-storage
        emptyDir:
          medium: Memory
          sizeLimit: 100Mi
```

**Generated NetworkPolicy:**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mcp-server-network-policy
spec:
  podSelector:
    matchLabels:
      app: mcp-server
  policyTypes:
  - Egress
  egress:
  - ports:
    - protocol: TCP
      port: 80
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 53
    # Note: K8s NetworkPolicy requires IP/CIDR ranges
    # FQDN filtering needs additional tooling
```

## Advanced Use Cases

### Multi-Environment Deployment

```bash
# Development
warden k8s render --policy dev-policy.yaml --namespace dev

# Staging  
warden k8s render --policy staging-policy.yaml --namespace staging

# Production
warden k8s render --policy prod-policy.yaml --namespace production
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
- name: Generate K8s Manifests
  run: |
    warden k8s validate --policy k8s-policy.yaml
    warden k8s render --policy k8s-policy.yaml \
      --image ${{ env.IMAGE_TAG }} \
      --output manifests.yaml

- name: Deploy to K8s
  run: kubectl apply -f manifests.yaml
```

### Helm Chart Integration

```bash
# Generate manifests as Helm templates
warden k8s render --policy policy.yaml --image "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
```

## Security Best Practices

### 1. Principle of Least Privilege

```yaml
# Minimal filesystem access
filesystem:
  read: ["/app"]           # Only application code
  write: ["/tmp"]          # Only temporary storage

# Minimal network access  
network:
  allow: ["api.example.com"]  # Only required APIs

# Minimal environment
env:
  allow: ["APP_CONFIG"]    # Only required variables
```

### 2. Resource Management

```yaml
limits:
  memory_mb: 256          # Prevent memory exhaustion
  timeout_s: 300          # Prevent runaway processes
```

### 3. Secret Management

```yaml
# In policy - only declare variable names
env:
  allow: ["DB_PASSWORD"]

# In K8s - use Secret resources
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
data:
  db-password: <base64-encoded-value>
```

### 4. Network Segmentation

```yaml
# Restrict to essential services only
network:
  allow: 
    - "api.internal.company.com"
    - "metrics.company.com"
    # Avoid wildcards or overly broad access
```

## Limitations & Workarounds

### Network Limitations

**Issue**: Kubernetes NetworkPolicy works with IP/CIDR ranges, not hostnames.

**Workaround Options:**
1. **Cilium**: Use CiliumNetworkPolicy for FQDN-based rules
2. **Istio**: Use ServiceEntry and VirtualService for external services
3. **Manual Resolution**: Resolve hostnames to IP ranges and update policies

```yaml
# Standard K8s (limited)
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
spec:
  egress:
  - to:
    - namespaceSelector: {}  # Only works with IPs/CIDRs

# Cilium (FQDN support)
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
spec:
  egress:
  - toFQDNs:
    - matchName: "api.github.com"
```

### Filesystem Limitations

**Issue**: Kubernetes volumes require explicit paths, no wildcard support.

**Workaround**: Use initContainers to set up required directory structure.

### Environment Variable Patterns

**Issue**: Warden `env.allow` supports patterns, K8s requires explicit names.

**Workaround**: Policy validation warns about patterns; expand manually.

## Monitoring & Compliance

### Security Scanning

```bash
# Validate generated manifests
kubectl --dry-run=server apply -f manifests.yaml

# Scan with security tools
kubesec scan manifests.yaml
kube-score score manifests.yaml
```

### Runtime Security

```bash
# Monitor with Falco rules
# Pod started with privileged: true → Alert
# Process accessed /etc/shadow → Alert
# Network connection to unexpected host → Alert
```

### Compliance Verification

```bash
# Check against security benchmarks
kube-bench run --targets node,policies,managedservices
```

## Migration Strategies

### From Docker Compose

1. **Extract policies** from Docker Compose security settings
2. **Validate translation** using `warden k8s validate`
3. **Generate manifests** with `warden k8s render`
4. **Test deployment** in staging environment

### From Kubernetes YAML

1. **Reverse engineer** existing SecurityContext into Warden policy
2. **Validate consistency** with original security posture
3. **Adopt policy-driven** approach for future deployments

### From VM-based Deployment

1. **Analyze system access** using `warden trace` on VM
2. **Generate initial policy** with `warden init`
3. **Refine for container** environment using `warden k8s validate`
4. **Deploy with confidence** using generated manifests

## Integration Examples

### ArgoCD GitOps

```yaml
# argocd-application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: secure-mcp-app
spec:
  source:
    plugin:
      name: warden-k8s
      env:
      - name: WARDEN_POLICY
        value: policy.yaml
      - name: WARDEN_IMAGE  
        value: myapp:v1.2.3
```

### Terraform Integration

```hcl
resource "kubernetes_manifest" "secure_deployment" {
  manifest = yamldecode(
    shell_script {
      command = "warden k8s render --policy ${var.policy_file} --image ${var.image}"
    }.stdout
  )
}
```

### Jenkins Pipeline

```groovy
pipeline {
  stages {
    stage('Generate Manifests') {
      steps {
        sh 'warden k8s validate --policy k8s-policy.yaml'
        sh 'warden k8s render --policy k8s-policy.yaml --image ${IMAGE_TAG} --output manifests.yaml'
        archiveArtifacts 'manifests.yaml'
      }
    }
    stage('Deploy') {
      steps {
        sh 'kubectl apply -f manifests.yaml'
      }
    }
  }
}
```

## Performance Considerations

### Resource Overhead

- **SecurityContext**: Minimal overhead (~1% CPU)
- **NetworkPolicy**: Low overhead with modern CNI
- **SeccompProfile**: Very low overhead (~0.1% CPU)
- **ReadOnlyRootFilesystem**: No performance impact

### Startup Time

- **EmptyDir volumes**: Fast (memory-backed)
- **HostPath volumes**: Fast (direct mount)
- **Seccomp loading**: Minimal delay (<100ms)

### Network Performance

- **NetworkPolicy**: Minimal latency impact
- **FQDN resolution**: May add DNS lookup time
- **Connection filtering**: Negligible overhead

## Troubleshooting

### Common Issues

**Pod won't start: "permission denied"**
```bash
# Check SecurityContext
kubectl describe pod <pod-name>

# Common fix: ensure runAsNonRoot user exists in image
USER 1000:1000  # In Dockerfile
```

**Network connections blocked**
```bash
# Check NetworkPolicy
kubectl describe networkpolicy <policy-name>

# Debug with temporary policy
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy  
metadata:
  name: debug-allow-all
spec:
  podSelector: {}
  egress: [{}]  # Allow all (temporary)
EOF
```

**Volume mount failures**
```bash
# Check volume definitions
kubectl get pod <pod-name> -o yaml | grep -A 10 volumes

# Verify host paths exist
kubectl exec <pod-name> -- ls -la /host/path
```

### Debug Commands

```bash
# Check generated SecurityContext
kubectl get pod <pod-name> -o jsonpath='{.spec.securityContext}'

# Verify capabilities
kubectl exec <pod-name> -- grep Cap /proc/self/status

# Check filesystem permissions
kubectl exec <pod-name> -- touch /test-write  # Should fail in read-only

# Network connectivity test
kubectl exec <pod-name> -- nc -zv api.github.com 443
```

## Related Documentation

- [Warden Policy Reference](./policy-schema.md)
- [MCP Client Proxy](./client-proxy.md)
- [CI/CD Integration](../github/actions/warden-action/docs/ci-integration.md)
- [Security Best Practices](./security.md)