package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type runConfig struct {
	Backend         string
	Mode            string
	WardenPath      string
	ProbePath       string
	MatrixPath      string
	PolicyDir       string
	OutDir          string
	Filter          string
	Group           string
	Image           string
	Timeout         time.Duration
	RequireComplete bool
	List            bool
	Verbose         bool
	ForceBuild      bool
	NoNetns         bool
}

func main() {
	var cfg runConfig
	flag.StringVar(&cfg.Backend, "backend", "auto", "backend to test: auto, linux, docker, seatbelt, windows")
	flag.StringVar(&cfg.Mode, "mode", "auto", "test network mode: auto, netns, host")
	flag.StringVar(&cfg.WardenPath, "warden", "", "path to a prebuilt warden binary")
	flag.StringVar(&cfg.ProbePath, "probe", "", "path to a prebuilt egress probe binary")
	flag.StringVar(&cfg.MatrixPath, "matrix", "", "path to matrix.yaml")
	flag.StringVar(&cfg.PolicyDir, "policies", "", "path to policy fixtures")
	flag.StringVar(&cfg.OutDir, "out", "", "evidence output directory")
	flag.StringVar(&cfg.Filter, "filter", "", "regexp filter on case IDs")
	flag.StringVar(&cfg.Group, "group", "", "only run one matrix group (A-G)")
	flag.StringVar(&cfg.Image, "image", "", "docker image (default WARDEN_DOCKER_IMAGE or alpine:3.20)")
	flag.DurationVar(&cfg.Timeout, "timeout", 90*time.Second, "per-case wall-clock timeout")
	flag.BoolVar(&cfg.RequireComplete, "require-complete", false, "exit nonzero when any case is BLOCKED or NOT_RUN")
	flag.BoolVar(&cfg.List, "list", false, "list matrix cases and exit")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "print live case progress")
	flag.BoolVar(&cfg.ForceBuild, "build", false, "rebuild warden and probe even if paths were supplied")
	flag.BoolVar(&cfg.NoNetns, "no-netns", false, "skip the Linux test-network namespace (host mode)")
	flag.Parse()

	rest := os.Args[1:]

	// Phase 0: enter the isolated test network before touching collectors.
	if runtime.GOOS == "linux" && os.Getenv(envInNS) != "1" && !cfg.NoNetns {
		mode := cfg.Mode
		if mode == "auto" {
			mode = "netns"
		}
		if mode == "netns" {
			ok, why := netnsSupported()
			if !ok {
				fmt.Fprintf(os.Stderr, "egress runner: BLOCKED: cannot create test network namespace: %s\n", why)
				os.Exit(2)
			}
			if err := reexecInNetns(rest); err != nil {
				if ee, isExit := err.(*exec.ExitError); isExit {
					os.Exit(ee.ExitCode())
				}
				fmt.Fprintf(os.Stderr, "egress runner: re-exec under unshare failed: %v\n", err)
				os.Exit(2)
			}
			return
		}
	}

	if err := runSuite(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "egress runner: %v\n", err)
		os.Exit(2)
	}
}

func findModuleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func findRepoRoot(moduleRoot string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = moduleRoot
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return moduleRoot
}

func runSuite(cfg *runConfig) error {
	moduleRoot := findModuleRoot()
	repoRoot := findRepoRoot(moduleRoot)
	stamp := time.Now().UTC().Format("20060102T150405Z")
	short := make([]byte, 3)
	_, _ = rand.Read(short)
	runID := "r" + strconvTime(time.Now()) + hex.EncodeToString(short)
	zone := strings.ToLower(runID) + ".egress.test"

	if cfg.MatrixPath == "" {
		cfg.MatrixPath = filepath.Join(moduleRoot, "testdata", "egress", "matrix.yaml")
	}
	if cfg.PolicyDir == "" {
		cfg.PolicyDir = filepath.Join(moduleRoot, "testdata", "egress", "policies")
	}
	backend := resolveBackend(cfg)
	if backend == "" {
		return fmt.Errorf("no sandbox backend available; capability report requires a host backend")
	}
	mode := cfg.Mode
	if mode == "auto" {
		if runtime.GOOS == "linux" && backend != "seatbelt" && backend != "windows" {
			mode = "netns"
		} else {
			mode = "host"
		}
	}

	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join(repoRoot, "evidence", "egress", runtime.GOOS, stamp, backend)
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return err
	}
	w, err := newEvidenceWriter(cfg.OutDir)
	if err != nil {
		return err
	}
	defer w.close()

	matrix, err := loadMatrix(cfg.MatrixPath)
	if err != nil {
		return err
	}
	if cfg.List {
		for _, c := range matrix.Cases {
			req := ""
			if len(c.Requires) > 0 {
				req = " requires=" + strings.Join(c.Requires, ",")
			}
			fmt.Printf("%-4s %-8s %s%s\n", c.ID, c.Kind, c.Title, req)
		}
		return nil
	}

	dnsIP := "192.0.2.53"
	if mode == "host" {
		dnsIP = "127.0.0.1"
	}
	var ns *netnsState
	if os.Getenv(envInNS) == "1" {
		tmpRoot := filepath.Join(cfg.OutDir, "netns")
		if err := os.MkdirAll(tmpRoot, 0o755); err != nil {
			return err
		}
		ns, err = setupNetns(tmpRoot, dnsIP)
		if err != nil {
			return fmt.Errorf("setup test network: %w", err)
		}
	} else {
		ns = &netnsState{Notes: []string{"runner not inside the Linux test network namespace"}}
	}

	// Build or locate the two binaries.
	wardenPath, probePath, harnessDir, err := prepareBinaries(cfg, moduleRoot)
	if err != nil {
		return err
	}
	caPEM, cert, err := generateTLSMaterial(zone)
	if err != nil {
		return err
	}
	caFile := filepath.Join(harnessDir, "ca.pem")
	if err := os.WriteFile(caFile, caPEM, 0o644); err != nil {
		return err
	}

	caps := detectCapabilities(cfg, backend, wardenPath)
	wardenVer, wardenCommit := wardenIdentity(wardenPath, moduleRoot)
	meta := RunMeta{
		RunID: runID, Stamp: stamp, StartedAt: time.Now().UTC(),
		OS: runtime.GOOS, Arch: runtime.GOARCH, Kernel: kernelRelease(),
		Platform: runtime.GOOS + "/" + runtime.GOARCH, Backend: backend, Mode: mode,
		Privilege: privilegeString(), WardenPath: wardenPath, WardenVer: wardenVer,
		WardenCommit: wardenCommit, ProbePath: probePath, MatrixPath: cfg.MatrixPath,
		Command: os.Args, Capabilities: caps, Zone: zone,
	}
	if ns != nil {
		meta.Capabilities["ipv4"] = ns.IPv4
		meta.Capabilities["ipv6"] = ns.IPv6
		meta.Capabilities["netns_addresses"] = ns.Addresses
		if len(ns.Notes) > 0 {
			meta.Limitations = append(meta.Limitations, ns.Notes...)
		}
	}
	if err := w.writeRunMeta(meta); err != nil {
		return err
	}

	allowedIP, allowedIP6, deniedIP, deniedIP6, metaIP, privateIP, loopIP := "192.0.2.1", "2001:db8::1", "192.0.2.66", "2001:db8::2", "169.254.169.254", "10.0.0.1", "127.0.0.1"
	lanIP := detectLANIP()
	if mode == "host" {
		allowedIP, allowedIP6 = "127.0.0.1", ""
		deniedIP, deniedIP6 = lanIP, ""
		metaIP, privateIP = lanIP, lanIP
		loopIP = "127.0.0.1"
	}
	collectors := newCollectors(collectorConfig{
		Mode: mode, Zone: zone, DNSIP: dnsIP,
		AllowedIP: allowedIP, AllowedIP6: allowedIP6,
		DeniedIP: deniedIP, DeniedIP6: deniedIP6,
		MetaIP: metaIP, PrivateIP: privateIP, LoopIP: loopIP, LANIP: lanIP,
		Cert: cert,
	}, filepath.Join(cfg.OutDir, "collectors"))
	if err := collectors.Start(); err != nil {
		return fmt.Errorf("start collectors: %w", err)
	}
	defer collectors.Close()
	if mode == "netns" && !collectors.V6OK() {
		meta.Limitations = append(meta.Limitations, "IPv6 test addresses unavailable; IPv6 cases are UNSUPPORTED")
	}

	runner := &Runner{
		cfg: cfg, backend: backend, mode: mode, zone: zone, runID: runID, dnsIP: dnsIP,
		moduleRoot: moduleRoot, harnessDir: harnessDir,
		wardenPath: wardenPath, probePath: probePath, caFile: caFile,
		collectors: collectors,
		allowedIP:  allowedIP, deniedIP: deniedIP, metaIP: metaIP,
		privateIP: privateIP, loopIP: loopIP, lanIP: lanIP,
		image: cfg.Image,
	}

	var filter *regexp.Regexp
	if cfg.Filter != "" {
		filter, err = regexp.Compile(cfg.Filter)
		if err != nil {
			return fmt.Errorf("bad -filter: %w", err)
		}
	}
	var results []CaseResult
	for _, c := range matrix.Cases {
		if !strings.EqualFold(c.Group, cfg.Group) && cfg.Group != "" {
			continue
		}
		if filter != nil && !filter.MatchString(c.ID) {
			continue
		}
		if cfg.Verbose {
			fmt.Printf("[%s] %s\n", c.ID, c.Title)
		}
		res := runner.runCase(c)
		results = append(results, res)
		if err := w.writeCase(res); err != nil {
			return err
		}
		if cfg.Verbose {
			fmt.Printf("  -> %s %s\n", res.Classification, res.Reason)
		}
	}

	meta.FinishedAt = time.Now().UTC()
	if err := w.writeRunMeta(meta); err != nil {
		return err
	}
	summary := Summary{
		RunID: runID, Backend: backend, Mode: mode,
		Counts: classifyCounts(results), Total: len(results), Results: results,
	}
	summary.Verdict, summary.VerdictReason = verdict(results)
	if err := writeJSON(filepath.Join(cfg.OutDir, "summary.json"), summary); err != nil {
		return err
	}
	report := renderReport(meta, summary)
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "report.md"), []byte(report), 0o644); err != nil {
		return err
	}
	fmt.Printf("backend=%s mode=%s cases=%d pass=%d fail=%d warn=%d blocked=%d unsupported=%d not_run=%d verdict=%s\n",
		backend, mode, summary.Total, summary.Counts[ClassPass], summary.Counts[ClassFail],
		summary.Counts[ClassWarn], summary.Counts[ClassBlocked], summary.Counts[ClassUnsupported],
		summary.Counts[ClassNotRun], summary.Verdict)
	fmt.Printf("evidence: %s\n", cfg.OutDir)
	if summary.Counts[ClassFail] > 0 {
		return fmt.Errorf("egress integrity failures: see %s", cfg.OutDir)
	}
	if cfg.RequireComplete && (summary.Counts[ClassBlocked] > 0 || summary.Counts[ClassNotRun] > 0) {
		return fmt.Errorf("incomplete run: blocked=%d not_run=%d", summary.Counts[ClassBlocked], summary.Counts[ClassNotRun])
	}
	return nil
}

func strconvTime(t time.Time) string { return fmt.Sprintf("%d", t.Unix()) }

func resolveBackend(cfg *runConfig) string {
	if cfg.Backend != "" && cfg.Backend != "auto" {
		return cfg.Backend
	}
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("bwrap"); err == nil {
			return "linux"
		}
		if dockerUsable() {
			return "docker"
		}
	case "darwin":
		if _, err := exec.LookPath("sandbox-exec"); err == nil {
			return "seatbelt"
		}
		if dockerUsable() {
			return "docker"
		}
	case "windows":
		if dockerUsable() {
			return "docker"
		}
		return "windows"
	}
	return ""
}

func dockerUsable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").Run() == nil
}

// prepareBinaries builds (or validates) the warden and probe executables.
// Building happens before the test network namespace's resolver is used by
// the workload; a missing module cache is a hard, explainable failure.
func prepareBinaries(cfg *runConfig, moduleRoot string) (wardenPath, probePath, harnessDir string, err error) {
	harnessDir = filepath.Join(cfg.OutDir, "harness")
	if err := os.MkdirAll(harnessDir, 0o755); err != nil {
		return "", "", "", err
	}
	wardenPath = cfg.WardenPath
	probePath = cfg.ProbePath
	if cfg.ForceBuild || wardenPath == "" {
		wardenPath = filepath.Join(harnessDir, "warden")
		if err := buildGo(cfg, moduleRoot, "./cmd/warden", wardenPath); err != nil {
			return "", "", "", fmt.Errorf("build warden: %w", err)
		}
	}
	if cfg.ForceBuild || probePath == "" {
		probePath = filepath.Join(harnessDir, "probe")
		if runtime.GOOS == "windows" {
			probePath += ".exe"
		}
		if err := buildGo(cfg, moduleRoot, "./testdata/egress/probe", probePath); err != nil {
			return "", "", "", fmt.Errorf("build probe: %w", err)
		}
	}
	_ = os.Chmod(harnessDir, 0o755)
	_ = os.Chmod(wardenPath, 0o755)
	_ = os.Chmod(probePath, 0o755)
	return wardenPath, probePath, harnessDir, nil
}

func buildGo(cfg *runConfig, moduleRoot, pkg, out string) error {
	args := []string{"build", "-o", out, pkg}
	cmd := exec.Command("go", args...)
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build %s: %v: %s", pkg, err, strings.TrimSpace(string(outBytes)))
	}
	return nil
}

func wardenIdentity(wardenPath, moduleRoot string) (version, commit string) {
	if out, err := exec.Command(wardenPath, "version").CombinedOutput(); err == nil {
		version = strings.TrimSpace(string(out))
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = moduleRoot
	if out, err := cmd.Output(); err == nil {
		commit = strings.TrimSpace(string(out))
	}
	return version, commit
}

func detectCapabilities(cfg *runConfig, backend, wardenPath string) map[string]any {
	caps := map[string]any{
		"goos": runtime.GOOS, "goarch": runtime.GOARCH,
		"backend": backend, "warden_path": wardenPath,
	}
	for _, tool := range []string{"bwrap", "setpriv", "strace", "unshare", "ip", "docker", "sandbox-exec", "python3", "node", "bash", "sh"} {
		_, err := exec.LookPath(tool)
		caps["tool_"+tool] = err == nil
	}
	caps["docker_usable"] = dockerUsable()
	if backend == "docker" {
		image := cfg.Image
		if image == "" {
			image = os.Getenv("WARDEN_DOCKER_IMAGE")
		}
		if image == "" {
			image = "alpine:3.20"
		}
		caps["docker_image"] = image
		caps["docker_image_present"] = exec.Command("docker", "image", "inspect", image).Run() == nil
	}
	return caps
}

func detectLANIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "127.0.0.1"
	}
	return host
}

func kernelRelease() string {
	b, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func privilegeString() string {
	if os.Geteuid() == 0 {
		return "uid=0"
	}
	return fmt.Sprintf("uid=%d", os.Geteuid())
}

func renderReport(meta RunMeta, s Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Egress and Enforcement Integrity Report\n\n")
	fmt.Fprintf(&b, "- Run ID: `%s`\n- Started: %s\n- Finished: %s\n- Platform: %s\n- Backend: `%s`\n- Mode: `%s`\n- Privilege: %s\n- Warden: `%s` (%s)\n- Commit: `%s`\n- Zone: `%s`\n- Evidence command: `%s`\n- Evidence: `%s`\n\n",
		meta.RunID, meta.StartedAt.Format(time.RFC3339), meta.FinishedAt.Format(time.RFC3339),
		meta.Platform, meta.Backend, meta.Mode, meta.Privilege, meta.WardenPath, meta.WardenVer, meta.WardenCommit, meta.Zone, strings.Join(meta.Command, " "), "")
	fmt.Fprintf(&b, "## Verdict\n\n**%s** - %s\n\n", s.Verdict, s.VerdictReason)
	fmt.Fprintf(&b, "## Summary\n\n| Classification | Count |\n|---|---|\n")
	for _, k := range []string{ClassPass, ClassFail, ClassWarn, ClassBlocked, ClassUnsupported, ClassNotRun} {
		fmt.Fprintf(&b, "| %s | %d |\n", k, s.Counts[k])
	}
	fmt.Fprintf(&b, "| Total | %d |\n\n", s.Total)
	fmt.Fprintf(&b, "WARN, UNSUPPORTED, BLOCKED and NOT_RUN are never counted as PASS.\n\n")
	fmt.Fprintf(&b, "## Cases\n\n| ID | Title | Expected | Result | Reason |\n|---|---|---|---|---|\n")
	for _, r := range s.Results {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", r.ID, escapePipe(r.Title), r.Expected, r.Classification, escapePipe(r.Reason))
	}
	fmt.Fprintf(&b, "\n## Capabilities\n\n")
	for k, v := range meta.Capabilities {
		fmt.Fprintf(&b, "- `%s`: %v\n", k, v)
	}
	if len(meta.Limitations) > 0 {
		fmt.Fprintf(&b, "\n## Limitations\n\n")
		for _, l := range meta.Limitations {
			fmt.Fprintf(&b, "- %s\n", l)
		}
	}
	fmt.Fprintf(&b, "\n## Failing and inconclusive cases\n\n")
	for _, r := range s.Results {
		if r.Classification == ClassPass {
			continue
		}
		fmt.Fprintf(&b, "### %s (%s) - %s\n\n%s\n\n", r.ID, r.Classification, r.Title, r.Reason)
	}
	return b.String()
}

func escapePipe(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
