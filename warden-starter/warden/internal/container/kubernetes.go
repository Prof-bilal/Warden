// Package container provides Kubernetes manifest generation from Warden policies.
// This extends the policy translation to generate complete K8s YAML manifests.
package container

import (
	"fmt"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
	"gopkg.in/yaml.v3"
)

// KubernetesManifests contains all generated K8s resources
type KubernetesManifests struct {
	Deployment    *DeploymentManifest    `yaml:"deployment,omitempty"`
	NetworkPolicy *NetworkPolicyManifest `yaml:"network_policy,omitempty"`
	ConfigMap     *ConfigMapManifest     `yaml:"config_map,omitempty"`
	SeccompProfile *SeccompProfileManifest `yaml:"seccomp_profile,omitempty"`
}

// DeploymentManifest represents a Kubernetes Deployment
type DeploymentManifest struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   ObjectMeta        `yaml:"metadata"`
	Spec       DeploymentSpec    `yaml:"spec"`
}

// NetworkPolicyManifest represents a Kubernetes NetworkPolicy
type NetworkPolicyManifest struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   ObjectMeta           `yaml:"metadata"`
	Spec       NetworkPolicySpec    `yaml:"spec"`
}

// ConfigMapManifest represents a Kubernetes ConfigMap
type ConfigMapManifest struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   ObjectMeta        `yaml:"metadata"`
	Data       map[string]string `yaml:"data,omitempty"`
	BinaryData map[string][]byte `yaml:"binaryData,omitempty"`
}

// SeccompProfileManifest represents a SecurityProfile CRD
type SeccompProfileManifest struct {
	APIVersion string        `yaml:"apiVersion"`
	Kind       string        `yaml:"kind"`
	Metadata   ObjectMeta    `yaml:"metadata"`
	Spec       SeccompProfile `yaml:"spec"`
}

// ObjectMeta represents Kubernetes object metadata
type ObjectMeta struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// DeploymentSpec represents the spec of a Deployment
type DeploymentSpec struct {
	Replicas int32            `yaml:"replicas,omitempty"`
	Selector LabelSelector    `yaml:"selector"`
	Template PodTemplateSpec  `yaml:"template"`
}

// LabelSelector represents a label selector
type LabelSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}

// PodTemplateSpec represents a pod template
type PodTemplateSpec struct {
	Metadata ObjectMeta `yaml:"metadata"`
	Spec     PodSpec    `yaml:"spec"`
}

// PodSpec represents the specification of a pod
type PodSpec struct {
	SecurityContext      *PodSecurityContext       `yaml:"securityContext,omitempty"`
	Containers           []Container               `yaml:"containers"`
	Volumes              []VolumeSpec              `yaml:"volumes,omitempty"`
	RestartPolicy        string                    `yaml:"restartPolicy,omitempty"`
	ServiceAccountName   string                    `yaml:"serviceAccountName,omitempty"`
	AutomountServiceAccountToken *bool             `yaml:"automountServiceAccountToken,omitempty"`
}

// Container represents a container in a pod
type Container struct {
	Name            string                     `yaml:"name"`
	Image           string                     `yaml:"image"`
	Command         []string                   `yaml:"command,omitempty"`
	Args            []string                   `yaml:"args,omitempty"`
	WorkingDir      string                     `yaml:"workingDir,omitempty"`
	Env             []EnvVar                   `yaml:"env,omitempty"`
	Resources       ResourceRequirements       `yaml:"resources,omitempty"`
	VolumeMounts    []VolumeMountSpec          `yaml:"volumeMounts,omitempty"`
	SecurityContext *ContainerSecurityContext  `yaml:"securityContext,omitempty"`
	ImagePullPolicy string                     `yaml:"imagePullPolicy,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value,omitempty"`
}

// GenerateKubernetesManifests creates complete K8s manifests from a Warden policy
func GenerateKubernetesManifests(p policy.Policy, options TranslateOptions) (*KubernetesManifests, error) {
	manifests := &KubernetesManifests{}

	// Generate base configuration
	config, err := generateKubernetesConfig(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate k8s config: %w", err)
	}

	// Generate Deployment
	deployment, err := generateDeployment(p, config, options)
	if err != nil {
		return nil, fmt.Errorf("generate deployment: %w", err)
	}
	manifests.Deployment = deployment

	// Generate NetworkPolicy
	networkPolicy, err := generateNetworkPolicyManifest(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate network policy: %w", err)
	}
	manifests.NetworkPolicy = networkPolicy

	// Generate seccomp profile if needed
	seccompProfile, err := generateSeccompProfileManifest(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate seccomp profile: %w", err)
	}
	manifests.SeccompProfile = seccompProfile

	return manifests, nil
}

// generateDeployment creates a Deployment manifest
func generateDeployment(p policy.Policy, config *KubernetesConfig, options TranslateOptions) (*DeploymentManifest, error) {
	name := sanitizeName(options.Image)
	if name == "" {
		name = "warden-app"
	}

	labels := map[string]string{
		"app":                name,
		"warden.security/managed": "true",
	}

	// Build environment variables
	var envVars []EnvVar
	for _, envName := range p.Env.Allow {
		envVars = append(envVars, EnvVar{
			Name:  envName,
			Value: "", // Value comes from the actual environment
		})
	}

	// Container definition. A policy may omit Command (it can be supplied
	// at runtime); split it into K8s command/args only when present.
	var command []string
	var args []string
	if len(p.Command) > 0 {
		command = p.Command[:1]
		args = p.Command[1:]
	}
	container := Container{
		Name:            name,
		Image:           options.Image,
		Command:         command,
		Args:            args,
		WorkingDir:      options.WorkingDir,
		Env:             envVars,
		Resources:       config.Resources,
		VolumeMounts:    config.VolumeMounts,
		SecurityContext: &config.ContainerSecurityContext,
		ImagePullPolicy: "IfNotPresent",
	}

	// Pod specification
	automountToken := false
	podSpec := PodSpec{
		SecurityContext: &config.SecurityContext,
		Containers:      []Container{container},
		Volumes:         config.Volumes,
		RestartPolicy:   "Always",
		AutomountServiceAccountToken: &automountToken,
	}

	// Deployment specification
	replicas := int32(1)
	deploymentSpec := DeploymentSpec{
		Replicas: replicas,
		Selector: LabelSelector{
			MatchLabels: labels,
		},
		Template: PodTemplateSpec{
			Metadata: ObjectMeta{
				Labels: labels,
			},
			Spec: podSpec,
		},
	}

	deployment := &DeploymentManifest{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: ObjectMeta{
			Name:      name,
			Namespace: options.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"warden.security/policy-version": "1.0",
				"warden.security/generated-by":   "warden-container-translator",
			},
		},
		Spec: deploymentSpec,
	}

	return deployment, nil
}

// generateNetworkPolicyManifest creates a NetworkPolicy manifest
func generateNetworkPolicyManifest(p policy.Policy, options TranslateOptions) (*NetworkPolicyManifest, error) {
	name := sanitizeName(options.Image)
	if name == "" {
		name = "warden-app"
	}

	labels := map[string]string{
		"app": name,
	}

	var egressRules []NetworkPolicyRule

	if len(p.Network.Allow) == 0 {
		// No network access allowed - empty egress rules
		egressRules = []NetworkPolicyRule{}
	} else {
		// Allow specific hosts
		// Note: This is a limitation - K8s NetworkPolicy works with IPs/CIDRs, not hostnames
		// In practice, you'd need to resolve hostnames or use additional tools like Cilium
		ports := []NetworkPolicyPort{
			{Protocol: "TCP", Port: "80"},
			{Protocol: "TCP", Port: "443"},
			{Protocol: "TCP", Port: "53"}, // DNS
		}

		egressRules = []NetworkPolicyRule{
			{
				Ports: ports,
				// To field would need specific IP ranges or namespace selectors
				// This is a significant limitation of standard K8s NetworkPolicy
			},
		}
	}

	policy := &NetworkPolicyManifest{
		APIVersion: "networking.k8s.io/v1",
		Kind:       "NetworkPolicy",
		Metadata: ObjectMeta{
			Name:      name + "-network-policy",
			Namespace: options.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"warden.security/policy-note": "FQDN filtering requires additional tooling (Cilium, Istio, etc.)",
			},
		},
		Spec: NetworkPolicySpec{
			PodSelector: labels,
			PolicyTypes: []string{"Egress"},
			Egress:      egressRules,
		},
	}

	return policy, nil
}

// generateSeccompProfileManifest creates a SecurityProfile manifest
func generateSeccompProfileManifest(p policy.Policy, options TranslateOptions) (*SeccompProfileManifest, error) {
	name := sanitizeName(options.Image)
	if name == "" {
		name = "warden-app"
	}

	seccompProfile, err := generateSeccompProfile(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate seccomp profile: %w", err)
	}

	manifest := &SeccompProfileManifest{
		APIVersion: "security-profiles-operator.x-k8s.io/v1beta1",
		Kind:       "SeccompProfile",
		Metadata: ObjectMeta{
			Name:      name + "-seccomp",
			Namespace: options.Namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: *seccompProfile,
	}

	return manifest, nil
}

// RenderKubernetesYAML converts manifests to YAML output
func RenderKubernetesYAML(manifests *KubernetesManifests) (string, error) {
	var output strings.Builder

	if manifests.Deployment != nil {
		deploymentYAML, err := yaml.Marshal(manifests.Deployment)
		if err != nil {
			return "", fmt.Errorf("marshal deployment: %w", err)
		}
		output.WriteString("# Deployment\n")
		output.WriteString("---\n")
		output.Write(deploymentYAML)
		output.WriteString("\n")
	}

	if manifests.NetworkPolicy != nil {
		networkPolicyYAML, err := yaml.Marshal(manifests.NetworkPolicy)
		if err != nil {
			return "", fmt.Errorf("marshal network policy: %w", err)
		}
		output.WriteString("# NetworkPolicy\n")
		output.WriteString("---\n")
		output.Write(networkPolicyYAML)
		output.WriteString("\n")
	}

	if manifests.SeccompProfile != nil {
		seccompYAML, err := yaml.Marshal(manifests.SeccompProfile)
		if err != nil {
			return "", fmt.Errorf("marshal seccomp profile: %w", err)
		}
		output.WriteString("# SeccompProfile\n")
		output.WriteString("---\n")
		output.Write(seccompYAML)
		output.WriteString("\n")
	}

	return output.String(), nil
}

// sanitizeName creates a valid Kubernetes resource name from an input string
func sanitizeName(input string) string {
	// Remove image registry and tag info
	if strings.Contains(input, "/") {
		parts := strings.Split(input, "/")
		input = parts[len(parts)-1]
	}
	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		input = parts[0]
	}

	// Replace invalid characters
	input = strings.ReplaceAll(input, "_", "-")
	input = strings.ToLower(input)

	// Ensure it starts and ends with alphanumeric
	input = strings.TrimPrefix(input, "-")
	input = strings.TrimSuffix(input, "-")

	// Limit length
	if len(input) > 63 {
		input = input[:63]
	}

	if input == "" {
		input = "app"
	}

	return input
}

// ValidateKubernetesPolicy checks if a policy can be safely translated to K8s
func ValidateKubernetesPolicy(p policy.Policy) []string {
	var warnings []string

	// Check for FQDN network rules
	if len(p.Network.Allow) > 0 {
		warnings = append(warnings, "Network policy uses hostnames - K8s NetworkPolicy requires IP/CIDR ranges. Consider using Cilium or Istio for FQDN filtering.")
	}

	// Check for complex filesystem patterns
	for _, path := range p.Filesystem.Read {
		if strings.Contains(path, "*") || strings.Contains(path, "?") {
			warnings = append(warnings, fmt.Sprintf("Filesystem path '%s' contains wildcards - not directly supported in K8s volumes", path))
		}
	}

	// Check for environment variable patterns
	for _, env := range p.Env.Allow {
		if strings.Contains(env, "*") {
			warnings = append(warnings, fmt.Sprintf("Environment variable pattern '%s' not supported - K8s requires explicit variable names", env))
		}
	}

	// Check for MCP-specific config
	if p.MCP != nil {
		warnings = append(warnings, "MCP proxy configuration detected - not applicable for container deployment")
	}

	return warnings
}

// GenerateHelmChart creates a basic Helm chart structure (placeholder)
func GenerateHelmChart(p policy.Policy, options TranslateOptions) (map[string]string, error) {
	// This would generate a complete Helm chart with templates
	// For now, returning a placeholder
	chart := make(map[string]string)

	chart["Chart.yaml"] = fmt.Sprintf(`apiVersion: v2
name: %s
description: Warden-secured application
version: 0.1.0
appVersion: "1.0"
`, sanitizeName(options.Image))

	chart["values.yaml"] = `# Default values for the chart
replicaCount: 1
image:
  repository: ""
  pullPolicy: IfNotPresent
  tag: ""

resources:
  limits:
    memory: "512Mi"
  requests:
    memory: "256Mi"

securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
`

	return chart, nil
}