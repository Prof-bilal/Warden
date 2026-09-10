// Package container provides policy-to-container translation functionality.
// This package converts Warden policies into container-native security controls
// like Docker security options, seccomp profiles, and Kubernetes SecurityContext.
package container

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/warden-sandbox/warden/internal/policy"
)

// ContainerManifest represents the complete container security configuration
type ContainerManifest struct {
	// Docker-specific settings
	Docker DockerConfig `json:"docker,omitempty"`
	
	// Kubernetes-specific settings
	Kubernetes KubernetesConfig `json:"kubernetes,omitempty"`
	
	// Common settings
	Seccomp SeccompProfile `json:"seccomp,omitempty"`
}

// DockerConfig contains Docker-specific security settings
type DockerConfig struct {
	// Security options for docker run
	SecurityOpt []string `json:"security_opt,omitempty"`
	
	// Read-only root filesystem
	ReadOnly bool `json:"read_only,omitempty"`
	
	// Tmpfs mounts for writable areas
	Tmpfs []string `json:"tmpfs,omitempty"`
	
	// Volume binds for read-only mounts
	Volumes []VolumeMount `json:"volumes,omitempty"`
	
	// Network mode
	NetworkMode string `json:"network_mode,omitempty"`
	
	// Capabilities to drop
	CapDrop []string `json:"cap_drop,omitempty"`
	
	// Memory limit
	Memory string `json:"memory,omitempty"`
	
	// CPU limit
	CPUs string `json:"cpus,omitempty"`
}

// KubernetesConfig contains Kubernetes-specific security settings
type KubernetesConfig struct {
	// SecurityContext for the pod
	SecurityContext PodSecurityContext `json:"security_context,omitempty"`
	
	// Container SecurityContext
	ContainerSecurityContext ContainerSecurityContext `json:"container_security_context,omitempty"`
	
	// NetworkPolicy for egress control
	NetworkPolicy NetworkPolicySpec `json:"network_policy,omitempty"`
	
	// Resource limits and requests
	Resources ResourceRequirements `json:"resources,omitempty"`
	
	// Volume mounts
	VolumeMounts []VolumeMountSpec `json:"volume_mounts,omitempty"`
	
	// Volumes
	Volumes []VolumeSpec `json:"volumes,omitempty"`
}

// VolumeMount represents a Docker volume mount
type VolumeMount struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	ReadOnly    bool   `json:"read_only,omitempty"`
	Type        string `json:"type,omitempty"` // bind, volume, tmpfs
}

// PodSecurityContext maps to Kubernetes PodSecurityContext
type PodSecurityContext struct {
	RunAsNonRoot       *bool              `json:"run_as_non_root,omitempty" yaml:"runAsNonRoot,omitempty"`
	RunAsUser          *int64             `json:"run_as_user,omitempty" yaml:"runAsUser,omitempty"`
	RunAsGroup         *int64             `json:"run_as_group,omitempty" yaml:"runAsGroup,omitempty"`
	FSGroup            *int64             `json:"fs_group,omitempty" yaml:"fsGroup,omitempty"`
	SeccompProfile     *SeccompProfileRef `json:"seccomp_profile,omitempty" yaml:"seccompProfile,omitempty"`
	SupplementalGroups []int64            `json:"supplemental_groups,omitempty" yaml:"supplementalGroups,omitempty"`
}

// ContainerSecurityContext maps to Kubernetes Container SecurityContext
type ContainerSecurityContext struct {
	AllowPrivilegeEscalation *bool                `json:"allow_privilege_escalation,omitempty" yaml:"allowPrivilegeEscalation,omitempty"`
	ReadOnlyRootFilesystem   *bool                `json:"read_only_root_filesystem,omitempty" yaml:"readOnlyRootFilesystem,omitempty"`
	RunAsNonRoot             *bool                `json:"run_as_non_root,omitempty" yaml:"runAsNonRoot,omitempty"`
	RunAsUser                *int64               `json:"run_as_user,omitempty" yaml:"runAsUser,omitempty"`
	RunAsGroup               *int64               `json:"run_as_group,omitempty" yaml:"runAsGroup,omitempty"`
	Capabilities             *CapabilitiesSpec    `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	SeccompProfile           *SeccompProfileRef   `json:"seccomp_profile,omitempty" yaml:"seccompProfile,omitempty"`
}

// CapabilitiesSpec defines container capabilities
type CapabilitiesSpec struct {
	Add  []string `json:"add,omitempty" yaml:"add,omitempty"`
	Drop []string `json:"drop,omitempty" yaml:"drop,omitempty"`
}

// SeccompProfileRef references a seccomp profile
type SeccompProfileRef struct {
	Type             string `json:"type" yaml:"type"` // RuntimeDefault, Localhost, Unconfined
	LocalhostProfile string `json:"localhost_profile,omitempty" yaml:"localhostProfile,omitempty"`
}

// NetworkPolicySpec defines network access rules
type NetworkPolicySpec struct {
	PodSelector map[string]string   `json:"pod_selector,omitempty" yaml:"podSelector,omitempty"`
	PolicyTypes []string            `json:"policy_types,omitempty" yaml:"policyTypes,omitempty"`
	Ingress     []NetworkPolicyRule `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Egress      []NetworkPolicyRule `json:"egress,omitempty" yaml:"egress,omitempty"`
}

// NetworkPolicyRule defines a single network rule
type NetworkPolicyRule struct {
	Ports []NetworkPolicyPort `json:"ports,omitempty" yaml:"ports,omitempty"`
	To    []NetworkPolicyPeer `json:"to,omitempty" yaml:"to,omitempty"`
	From  []NetworkPolicyPeer `json:"from,omitempty" yaml:"from,omitempty"`
}

// NetworkPolicyPort defines port access
type NetworkPolicyPort struct {
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Port     string `json:"port,omitempty" yaml:"port,omitempty"`
}

// NetworkPolicyPeer defines network endpoints
type NetworkPolicyPeer struct {
	NamespaceSelector map[string]string `json:"namespace_selector,omitempty" yaml:"namespaceSelector,omitempty"`
	PodSelector       map[string]string `json:"pod_selector,omitempty" yaml:"podSelector,omitempty"`
}

// ResourceRequirements defines resource limits and requests
type ResourceRequirements struct {
	Limits   map[string]string `json:"limits,omitempty" yaml:"limits,omitempty"`
	Requests map[string]string `json:"requests,omitempty" yaml:"requests,omitempty"`
}

// VolumeMountSpec defines a Kubernetes volume mount
type VolumeMountSpec struct {
	Name      string `json:"name" yaml:"name"`
	MountPath string `json:"mount_path" yaml:"mountPath"`
	ReadOnly  bool   `json:"read_only,omitempty" yaml:"readOnly,omitempty"`
	SubPath   string `json:"sub_path,omitempty" yaml:"subPath,omitempty"`
}

// VolumeSpec defines a Kubernetes volume
type VolumeSpec struct {
	Name      string          `json:"name" yaml:"name"`
	HostPath  *HostPathVolume `json:"host_path,omitempty" yaml:"hostPath,omitempty"`
	EmptyDir  *EmptyDirVolume `json:"empty_dir,omitempty" yaml:"emptyDir,omitempty"`
	ConfigMap *ConfigMapVolume `json:"config_map,omitempty" yaml:"configMap,omitempty"`
}

// HostPathVolume represents a host path volume
type HostPathVolume struct {
	Path string `json:"path" yaml:"path"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

// EmptyDirVolume represents an empty directory volume
type EmptyDirVolume struct {
	Medium    string `json:"medium,omitempty" yaml:"medium,omitempty"`
	SizeLimit string `json:"size_limit,omitempty" yaml:"sizeLimit,omitempty"`
}

// ConfigMapVolume represents a config map volume
type ConfigMapVolume struct {
	Name string `json:"name" yaml:"name"`
}

// SeccompProfile represents a seccomp security profile
type SeccompProfile struct {
	DefaultAction string        `json:"default_action" yaml:"defaultAction"`
	Architectures []string      `json:"architectures,omitempty" yaml:"architectures,omitempty"`
	Syscalls      []SyscallRule `json:"syscalls,omitempty" yaml:"syscalls,omitempty"`
}

// SyscallRule defines rules for system calls
type SyscallRule struct {
	Names  []string `json:"names" yaml:"names"`
	Action string   `json:"action" yaml:"action"`
}

// TranslatePolicy converts a Warden policy to container security manifests
func TranslatePolicy(p policy.Policy, options TranslateOptions) (*ContainerManifest, error) {
	manifest := &ContainerManifest{}

	// Generate Docker configuration
	dockerConfig, err := generateDockerConfig(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate Docker config: %w", err)
	}
	manifest.Docker = *dockerConfig

	// Generate Kubernetes configuration
	k8sConfig, err := generateKubernetesConfig(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate Kubernetes config: %w", err)
	}
	manifest.Kubernetes = *k8sConfig

	// Generate seccomp profile
	seccompProfile, err := generateSeccompProfile(p, options)
	if err != nil {
		return nil, fmt.Errorf("generate seccomp profile: %w", err)
	}
	manifest.Seccomp = *seccompProfile

	return manifest, nil
}

// TranslateOptions configures the policy translation process
type TranslateOptions struct {
	// Container image to use
	Image string
	
	// Working directory inside container
	WorkingDir string
	
	// Whether to emit warnings for unsupported features
	EmitWarnings bool
	
	// Target platform (docker, kubernetes, podman)
	Platform string
	
	// Namespace for Kubernetes resources
	Namespace string
}

// generateDockerConfig creates Docker-specific security configuration
func generateDockerConfig(p policy.Policy, options TranslateOptions) (*DockerConfig, error) {
	config := &DockerConfig{
		ReadOnly:    true,
		NetworkMode: "none", // Start with no network
		CapDrop:     []string{"ALL"},
		SecurityOpt: []string{
			"no-new-privileges:true",
		},
	}

	// Handle filesystem permissions
	for _, readPath := range p.Filesystem.Read {
		mount := VolumeMount{
			Source:   readPath,
			Target:   readPath,
			ReadOnly: true,
			Type:     "bind",
		}
		config.Volumes = append(config.Volumes, mount)
	}

	for _, writePath := range p.Filesystem.Write {
		if strings.HasPrefix(writePath, "/tmp") || writePath == "/tmp" {
			// Use tmpfs for /tmp
			config.Tmpfs = append(config.Tmpfs, writePath+":rw,noexec,nosuid,nodev,size=100m")
		} else {
			// Bind mount for other writable paths
			mount := VolumeMount{
				Source:   writePath,
				Target:   writePath,
				ReadOnly: false,
				Type:     "bind",
			}
			config.Volumes = append(config.Volumes, mount)
		}
	}

	// Handle network access
	if len(p.Network.Allow) > 0 {
		// Enable networking if hosts are allowed
		config.NetworkMode = "bridge"
		// Note: Docker doesn't have hostname-based filtering built-in
		// This would require additional network proxy or firewall rules
	}

	// Handle resource limits
	if p.Limits.MemoryMB > 0 {
		config.Memory = fmt.Sprintf("%dm", p.Limits.MemoryMB)
	}

	return config, nil
}

// generateKubernetesConfig creates Kubernetes-specific security configuration
func generateKubernetesConfig(p policy.Policy, options TranslateOptions) (*KubernetesConfig, error) {
	config := &KubernetesConfig{}

	// Security contexts
	runAsNonRoot := true
	readOnlyRootFS := true
	allowPrivEsc := false

	config.ContainerSecurityContext = ContainerSecurityContext{
		RunAsNonRoot:             &runAsNonRoot,
		ReadOnlyRootFilesystem:   &readOnlyRootFS,
		AllowPrivilegeEscalation: &allowPrivEsc,
		Capabilities: &CapabilitiesSpec{
			Drop: []string{"ALL"},
		},
		SeccompProfile: &SeccompProfileRef{
			Type: "RuntimeDefault",
		},
	}

	config.SecurityContext = PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		SeccompProfile: &SeccompProfileRef{
			Type: "RuntimeDefault",
		},
	}

	// Handle filesystem mounts
	volumeIndex := 0
	for _, readPath := range p.Filesystem.Read {
		volumeName := fmt.Sprintf("read-volume-%d", volumeIndex)
		
		// Volume definition
		volume := VolumeSpec{
			Name: volumeName,
			HostPath: &HostPathVolume{
				Path: readPath,
				Type: "DirectoryOrCreate",
			},
		}
		config.Volumes = append(config.Volumes, volume)
		
		// Volume mount
		mount := VolumeMountSpec{
			Name:      volumeName,
			MountPath: readPath,
			ReadOnly:  true,
		}
		config.VolumeMounts = append(config.VolumeMounts, mount)
		volumeIndex++
	}

	for _, writePath := range p.Filesystem.Write {
		volumeName := fmt.Sprintf("write-volume-%d", volumeIndex)
		
		var volume VolumeSpec
		if strings.HasPrefix(writePath, "/tmp") || writePath == "/tmp" {
			// Use emptyDir for /tmp
			volume = VolumeSpec{
				Name: volumeName,
				EmptyDir: &EmptyDirVolume{
					Medium:    "Memory",
					SizeLimit: "100Mi",
				},
			}
		} else {
			// Use hostPath for other writable directories
			volume = VolumeSpec{
				Name: volumeName,
				HostPath: &HostPathVolume{
					Path: writePath,
					Type: "DirectoryOrCreate",
				},
			}
		}
		config.Volumes = append(config.Volumes, volume)
		
		mount := VolumeMountSpec{
			Name:      volumeName,
			MountPath: writePath,
			ReadOnly:  false,
		}
		config.VolumeMounts = append(config.VolumeMounts, mount)
		volumeIndex++
	}

	// Handle network policy
	if len(p.Network.Allow) == 0 {
		// Deny all egress
		config.NetworkPolicy = NetworkPolicySpec{
			PolicyTypes: []string{"Egress"},
			Egress:      []NetworkPolicyRule{}, // Empty = deny all
		}
	} else {
		// Allow specific hosts (note: K8s NetworkPolicy works with IPs/CIDRs, not hostnames)
		config.NetworkPolicy = NetworkPolicySpec{
			PolicyTypes: []string{"Egress"},
			Egress: []NetworkPolicyRule{
				{
					// This is a simplified example - real implementation would need
					// hostname-to-IP resolution or DNS-based filtering
					Ports: []NetworkPolicyPort{
						{Protocol: "TCP", Port: "80"},
						{Protocol: "TCP", Port: "443"},
					},
				},
			},
		}
	}

	// Handle resource limits
	config.Resources = ResourceRequirements{
		Limits:   make(map[string]string),
		Requests: make(map[string]string),
	}

	if p.Limits.MemoryMB > 0 {
		config.Resources.Limits["memory"] = fmt.Sprintf("%dMi", p.Limits.MemoryMB)
		config.Resources.Requests["memory"] = fmt.Sprintf("%dMi", p.Limits.MemoryMB/2) // Request half of limit
	}

	return config, nil
}

// generateSeccompProfile creates a seccomp security profile
func generateSeccompProfile(p policy.Policy, options TranslateOptions) (*SeccompProfile, error) {
	profile := &SeccompProfile{
		DefaultAction: "SCMP_ACT_ERRNO",
		Architectures: []string{"SCMP_ARCH_X86_64", "SCMP_ARCH_X86", "SCMP_ARCH_X32"},
	}

	// Allow essential system calls for containerized processes
	essentialSyscalls := []string{
		"read", "write", "open", "close", "stat", "fstat", "lstat", "poll",
		"lseek", "mmap", "mprotect", "munmap", "brk", "rt_sigaction",
		"rt_sigprocmask", "rt_sigreturn", "ioctl", "pread64", "pwrite64",
		"readv", "writev", "access", "pipe", "select", "sched_yield",
		"mremap", "msync", "mincore", "madvise", "shmget", "shmat", "shmctl",
		"dup", "dup2", "pause", "nanosleep", "getitimer", "alarm",
		"setitimer", "getpid", "sendfile", "socket", "connect", "accept",
		"sendto", "recvfrom", "sendmsg", "recvmsg", "shutdown", "bind",
		"listen", "getsockname", "getpeername", "socketpair", "setsockopt",
		"getsockopt", "clone", "fork", "vfork", "execve", "exit", "wait4",
		"kill", "uname", "semget", "semop", "semctl", "shmdt", "msgget",
		"msgsnd", "msgrcv", "msgctl", "fcntl", "flock", "fsync", "fdatasync",
		"truncate", "ftruncate", "getdents", "getcwd", "chdir", "fchdir",
		"rename", "mkdir", "rmdir", "creat", "link", "unlink", "symlink",
		"readlink", "chmod", "fchmod", "chown", "fchown", "lchown", "umask",
		"gettimeofday", "getrlimit", "getrusage", "sysinfo", "times", "ptrace",
		"getuid", "syslog", "getgid", "setuid", "setgid", "geteuid", "getegid",
		"setpgid", "getppid", "getpgrp", "setsid", "setreuid", "setregid",
		"getgroups", "setgroups", "setresuid", "getresuid", "setresgid",
		"getresgid", "getpgid", "setfsuid", "setfsgid", "getsid", "capget",
		"capset", "rt_sigpending", "rt_sigtimedwait", "rt_sigqueueinfo",
		"rt_sigsuspend", "sigaltstack", "utime", "mknod", "uselib",
		"personality", "ustat", "statfs", "fstatfs", "sysfs", "getpriority",
		"setpriority", "sched_setparam", "sched_getparam", "sched_setscheduler",
		"sched_getscheduler", "sched_get_priority_max", "sched_get_priority_min",
		"sched_rr_get_interval", "mlock", "munlock", "mlockall", "munlockall",
		"vhangup", "modify_ldt", "pivot_root", "prctl", "arch_prctl",
		"adjtimex", "setrlimit", "chroot", "sync", "acct", "settimeofday",
		"mount", "umount2", "swapon", "swapoff", "reboot", "sethostname",
		"setdomainname", "iopl", "ioperm", "create_module", "init_module",
		"delete_module", "get_kernel_syms", "query_module", "quotactl",
		"nfsservctl", "getpmsg", "putpmsg", "afs_syscall", "tuxcall",
		"security", "gettid", "readahead", "setxattr", "lsetxattr", "fsetxattr",
		"getxattr", "lgetxattr", "fgetxattr", "listxattr", "llistxattr",
		"flistxattr", "removexattr", "lremovexattr", "fremovexattr", "tkill",
		"time", "futex", "sched_setaffinity", "sched_getaffinity",
		"set_thread_area", "io_setup", "io_destroy", "io_getevents",
		"io_submit", "io_cancel", "get_thread_area", "lookup_dcookie",
		"epoll_create", "epoll_ctl_old", "epoll_wait_old", "remap_file_pages",
		"getdents64", "set_tid_address", "restart_syscall", "semtimedop",
		"fadvise64", "timer_create", "timer_settime", "timer_gettime",
		"timer_getoverrun", "timer_delete", "clock_settime", "clock_gettime",
		"clock_getres", "clock_nanosleep", "exit_group", "epoll_wait",
		"epoll_ctl", "tgkill", "utimes", "vserver", "mbind", "set_mempolicy",
		"get_mempolicy", "mq_open", "mq_unlink", "mq_timedsend", "mq_timedreceive",
		"mq_notify", "mq_getsetattr", "kexec_load", "waitid", "add_key",
		"request_key", "keyctl", "ioprio_set", "ioprio_get", "inotify_init",
		"inotify_add_watch", "inotify_rm_watch", "migrate_pages", "openat",
		"mkdirat", "mknodat", "fchownat", "futimesat", "newfstatat", "unlinkat",
		"renameat", "linkat", "symlinkat", "readlinkat", "fchmodat", "faccessat",
		"pselect6", "ppoll", "unshare", "set_robust_list", "get_robust_list",
		"splice", "tee", "sync_file_range", "vmsplice", "move_pages",
		"utimensat", "epoll_pwait", "signalfd", "timerfd_create", "eventfd",
		"fallocate", "timerfd_settime", "timerfd_gettime", "accept4", "signalfd4",
		"eventfd2", "epoll_create1", "dup3", "pipe2", "inotify_init1",
		"preadv", "pwritev", "rt_tgsigqueueinfo", "perf_event_open",
	}

	profile.Syscalls = append(profile.Syscalls, SyscallRule{
		Names:  essentialSyscalls,
		Action: "SCMP_ACT_ALLOW",
	})

	return profile, nil
}

// RenderDockerCommand generates a docker run command from the policy
func RenderDockerCommand(p policy.Policy, options TranslateOptions) ([]string, error) {
	manifest, err := TranslatePolicy(p, options)
	if err != nil {
		return nil, fmt.Errorf("translate policy: %w", err)
	}

	cmd := []string{"docker", "run"}

	// Add security options
	for _, opt := range manifest.Docker.SecurityOpt {
		cmd = append(cmd, "--security-opt", opt)
	}

	// Add read-only root filesystem
	if manifest.Docker.ReadOnly {
		cmd = append(cmd, "--read-only")
	}

	// Add tmpfs mounts
	for _, tmpfs := range manifest.Docker.Tmpfs {
		cmd = append(cmd, "--tmpfs", tmpfs)
	}

	// Add volume mounts
	for _, volume := range manifest.Docker.Volumes {
		mountSpec := volume.Source + ":" + volume.Target
		if volume.ReadOnly {
			mountSpec += ":ro"
		}
		cmd = append(cmd, "-v", mountSpec)
	}

	// Add network mode
	if manifest.Docker.NetworkMode != "" {
		cmd = append(cmd, "--network", manifest.Docker.NetworkMode)
	}

	// Add capability drops
	for _, cap := range manifest.Docker.CapDrop {
		cmd = append(cmd, "--cap-drop", cap)
	}

	// Add resource limits
	if manifest.Docker.Memory != "" {
		cmd = append(cmd, "--memory", manifest.Docker.Memory)
	}
	if manifest.Docker.CPUs != "" {
		cmd = append(cmd, "--cpus", manifest.Docker.CPUs)
	}

	// Add image
	if options.Image != "" {
		cmd = append(cmd, options.Image)
	}

	// Add command
	if len(p.Command) > 0 {
		cmd = append(cmd, p.Command...)
	}

	return cmd, nil
}

// SaveSeccompProfile writes the seccomp profile to a file
func SaveSeccompProfile(profile SeccompProfile, path string) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal seccomp profile: %w", err)
	}

	return writeFile(path, data, 0o644)
}

// writeFile is a helper function for writing files (would use os.WriteFile in real implementation)
func writeFile(path string, data []byte, perm int) error {
	// This would be implemented using os.WriteFile
	return fmt.Errorf("file writing not implemented in this demo")
}