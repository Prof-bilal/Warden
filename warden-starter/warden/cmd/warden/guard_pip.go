package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/guard"
	"github.com/warden-sandbox/warden/internal/policy"
	"github.com/warden-sandbox/warden/internal/privatefile"
	"github.com/warden-sandbox/warden/internal/sandbox"
)

func guardPipInstall(ctx context.Context, registry, name, version, output, backend string, now time.Time, rules guard.ReleaseRules) error {
	release, err := guard.ReadPyPIRelease(ctx, nil, registry, name, version)
	if err != nil {
		return err
	}
	wheels, decision := guard.ReviewPyPIWheels(release, now, rules)
	spec := release.Name + "==" + release.Version
	if err := reportGuardDecision("pip", spec, decision, rules.OverrideAge); err != nil {
		return err
	}
	output, err = guardOutput("pip", release.Name+"@"+release.Version, output)
	if err != nil {
		return err
	}
	if err := installPipWheels(spec, output, backend, wheels); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Installed %s in %s. Dependencies must be reviewed and installed separately.\n", spec, output)
	return nil
}

func pipWheelInputs(spec string, wheels []guard.PyPIWheel) (links, requirements []byte, err error) {
	if len(wheels) == 0 {
		return nil, nil, fmt.Errorf("no reviewed wheel available")
	}
	var page, line strings.Builder
	page.WriteString("<!doctype html><html><body>\n")
	line.WriteString(spec)
	for _, wheel := range wheels {
		u, err := url.Parse(wheel.URL)
		if err != nil || u.Scheme != "https" || u.Host != "files.pythonhosted.org" || u.User != nil ||
			u.RawQuery != "" || u.Fragment != "" || path.Base(u.Path) != wheel.Filename {
			return nil, nil, fmt.Errorf("invalid PyPI wheel URL for %q", wheel.Filename)
		}
		fmt.Fprintf(&page, "<a href=\"%s\">%s</a>\n", html.EscapeString(u.String()), html.EscapeString(wheel.Filename))
		fmt.Fprintf(&line, " --hash=sha256:%s", wheel.SHA256)
	}
	page.WriteString("</body></html>\n")
	line.WriteByte('\n')
	return []byte(page.String()), []byte(line.String()), nil
}

func installPipWheels(spec, output, backend string, wheels []guard.PyPIWheel) error {
	links, requirements, err := pipWheelInputs(spec, wheels)
	if err != nil {
		return err
	}
	if _, err := sandbox.Resolve(backend); err != nil {
		return err
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		python, err = exec.LookPath("python")
		if err != nil {
			return fmt.Errorf("Python with pip is required")
		}
	}
	python, err = filepath.Abs(python)
	if err != nil {
		return err
	}
	realPython, err := filepath.EvalSymlinks(python)
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp(filepath.Dir(output), ".warden-wheels-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	if err := privatefile.Protect(scratch, true); err != nil {
		return err
	}
	page := filepath.Join(scratch, "wheels.html")
	req := filepath.Join(scratch, "requirements.txt")
	if err := writeExclusive(page, links); err != nil {
		return err
	}
	if err := writeExclusive(req, requirements); err != nil {
		return err
	}
	runtimeRoot := pythonRuntimeDir(realPython)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	read := []string{scratch, runtimeRoot}
	venv := pythonRuntimeDir(python)
	if info, err := os.Stat(filepath.Join(venv, "pyvenv.cfg")); err == nil && info.Mode().IsRegular() {
		read = append(read, venv)
	} else {
		python = realPython
	}
	for _, dir := range read[1:] {
		if policy.Within(home, dir) {
			return fmt.Errorf("Python runtime %q must not grant access to the entire home directory", dir)
		}
	}
	certificates, err := pipCertificates(read[1:])
	if err != nil {
		return err
	}
	cert := filepath.Join(scratch, "registry-ca.crt")
	if err := writeExclusive(cert, certificates); err != nil {
		return err
	}
	if err := os.Mkdir(output, 0700); err != nil {
		return fmt.Errorf("pip output must be a new directory: %w", err)
	}
	if err := privatefile.Protect(output, true); err != nil {
		return err
	}
	p := policy.Policy{
		Filesystem: policy.Filesystem{Read: read, Write: []string{output}, ProtectSecrets: true},
		Network:    policy.Network{Allow: []string{"files.pythonhosted.org"}},
		Limits:     policy.Limits{MemoryMB: 1024, TimeoutS: 600},
	}
	env := map[string]string{"HOME": output, "PIP_CONFIG_FILE": os.DevNull, "TMPDIR": output, "TEMP": output, "TMP": output}
	// shortcut: install one reviewed wheel, enable dependencies once the resolver checks every wheel in the graph.
	args := []string{python, "-I", "-m", "pip", "--isolated", "--disable-pip-version-check",
		"--use-deprecated=legacy-certs", "--cert", cert, "install",
		"--require-hashes", "--only-binary=:all:", "--no-index", "--find-links", page,
		"--no-cache-dir", "--no-deps", "--no-compile", "--target", output, "-r", req}
	return runInstallSandboxed(args, p, backend, env)
}

func pipCertificates(runtimes []string) ([]byte, error) {
	candidates := []string{"/etc/ssl/certs/ca-certificates.crt", "/etc/ssl/cert.pem",
		"/etc/ca-certificates/extracted/tls-ca-bundle.pem", "/etc/pki/tls/certs/ca-bundle.crt"}
	for _, root := range runtimes {
		candidates = append(candidates, filepath.Join(root, "Lib", "site-packages", "pip", "_vendor", "certifi", "cacert.pem"))
		paths, _ := filepath.Glob(filepath.Join(root, "lib", "python*", "site-packages", "pip", "_vendor", "certifi", "cacert.pem"))
		candidates = append(candidates, paths...)
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		const limit = 4 << 20
		data, err := io.ReadAll(io.LimitReader(file, limit+1))
		_ = file.Close()
		if err != nil || len(data) > limit {
			continue
		}
		var public bytes.Buffer
		for block, rest := pem.Decode(data); block != nil; block, rest = pem.Decode(rest) {
			if block.Type == "CERTIFICATE" {
				if _, err := x509.ParseCertificate(block.Bytes); err == nil {
					_ = pem.Encode(&public, &pem.Block{Type: "CERTIFICATE", Bytes: block.Bytes})
				}
			}
		}
		if public.Len() > 0 {
			return public.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("no public CA certificate bundle found for pip")
}

func pythonRuntimeDir(executable string) string {
	dir := filepath.Dir(executable)
	if strings.EqualFold(filepath.Base(dir), "bin") || strings.EqualFold(filepath.Base(dir), "scripts") {
		return filepath.Dir(dir)
	}
	return dir
}
