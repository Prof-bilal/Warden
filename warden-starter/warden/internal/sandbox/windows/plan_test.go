package windows

import (
	"strings"
	"testing"

	"github.com/warden-sandbox/warden/internal/policy"
)

func TestDeriveFilesystemGrantsDenyByDefault(t *testing.T) {
	grants, err := deriveFilesystemGrants(`C:\tools\node\node.exe`, policy.Policy{})
	if err != nil {
		t.Fatal(err)
	}
	var haveRuntime, haveExeParent bool
	for _, g := range grants {
		switch {
		case strings.EqualFold(g.Path, `C:\Windows\System32`):
			haveRuntime = true
			if g.AccessMask&accessWrite != 0 {
				t.Fatalf("runtime base granted write: %+v", g)
			}
		case strings.EqualFold(g.Path, `C:\tools\node`):
			haveExeParent = true
			if g.AccessMask&accessWrite != 0 {
				t.Fatalf("exe parent should be read-only: %+v", g)
			}
		}
	}
	if !haveRuntime || !haveExeParent {
		t.Fatalf("missing runtime or exe-parent grant: %+v", grants)
	}
	// No grant may cover a drive root.
	for _, g := range grants {
		if strings.EqualFold(g.Path, `C:\`) {
			t.Fatalf("drive root granted: %+v", g)
		}
	}
}

func TestDeriveFilesystemGrantsPolicyPaths(t *testing.T) {
	p := policy.Policy{Filesystem: policy.Filesystem{
		Read:  []string{`C:\data\in`},
		Write: []string{`C:\data\out`},
	}}
	grants, err := deriveFilesystemGrants(`C:\tools\server.exe`, p)
	if err != nil {
		t.Fatal(err)
	}
	mask := map[string]uint32{}
	for _, g := range grants {
		mask[strings.ToLower(g.Path)] = g.AccessMask
	}
	if m := mask[`c:\data\in`]; m&accessRead == 0 || m&accessWrite != 0 {
		t.Fatalf("read grant wrong: %x", m)
	}
	if m := mask[`c:\data\out`]; m&accessWrite == 0 {
		t.Fatalf("write grant missing write bit: %x", m)
	}
}

func TestDeriveFilesystemGrantsRejectsReadWriteConflict(t *testing.T) {
	p := policy.Policy{Filesystem: policy.Filesystem{
		Read:  []string{`C:\data\same`},
		Write: []string{`C:\data\same`},
	}}
	if _, err := deriveFilesystemGrants(`C:\tools\server.exe`, p); err == nil {
		t.Fatal("expected read/write conflict error")
	}
}

func TestDeriveFilesystemGrantsRejectsWriteUnderRuntimeBase(t *testing.T) {
	for _, path := range []string{`C:\Windows\System32\spool`, `C:\Windows\System32`, `C:\Windows\Temp`} {
		p := policy.Policy{Filesystem: policy.Filesystem{Write: []string{path}}}
		if _, err := deriveFilesystemGrants(`C:\tools\server.exe`, p); err == nil {
			t.Fatalf("expected error granting write to %q", path)
		}
	}
}

func TestDeriveFilesystemGrantsRejectsDriveRoot(t *testing.T) {
	p := policy.Policy{Filesystem: policy.Filesystem{Read: []string{`C:\`}}}
	if _, err := deriveFilesystemGrants(`C:\tools\server.exe`, p); err == nil {
		t.Fatal("expected error granting drive root")
	}
}

func TestBuildPlanRequiresAbsoluteCommand(t *testing.T) {
	if _, err := BuildPlan([]string{"node", "server.js"}, policy.Policy{}, "s1", "127.0.0.1:18080"); err == nil {
		t.Fatal("expected error for relative command")
	}
	if _, err := BuildPlan(nil, policy.Policy{}, "s1", "127.0.0.1:18080"); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestBuildPlanSessionBoundIdentities(t *testing.T) {
	p := policy.Policy{Network: policy.Network{Allow: []string{"api.github.com"}}}
	a, err := BuildPlan([]string{`C:\tools\server.exe`}, p, "one", "127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildPlan([]string{`C:\tools\server.exe`}, p, "two", "127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	if a.AppContainerName == b.AppContainerName {
		t.Fatalf("AppContainer monikers must be per-run: %q", a.AppContainerName)
	}
	if a.PackageSID == b.PackageSID {
		t.Fatal("package SIDs must differ per run")
	}
	if a.CapabilitySIDs[0] == b.CapabilitySIDs[0] {
		t.Fatal("capability SIDs must differ per run")
	}
	if !strings.HasPrefix(a.PackageSID, "S-1-15-3-") {
		t.Fatalf("package SID %q is not an AppContainer SID", a.PackageSID)
	}
	if !strings.HasPrefix(a.CapabilitySIDs[0], "S-1-15-7-") {
		t.Fatalf("capability SID %q is not a capability SID", a.CapabilitySIDs[0])
	}
}

func TestBuildPlanWFPAllowlist(t *testing.T) {
	plan, err := BuildPlan([]string{`C:\tools\server.exe`}, policy.Policy{}, "s", "127.0.0.1:18080")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.AllowIPs) != 1 || plan.AllowIPs[0] != "127.0.0.1" {
		t.Fatalf("AllowIPs = %v, want only loopback", plan.AllowIPs)
	}
	if len(plan.AllowPorts) != 1 || plan.AllowPorts[0] != 18080 {
		t.Fatalf("AllowPorts = %v, want [18080]", plan.AllowPorts)
	}
	// The policy's hostname allowlist is enforced by the egress proxy, not
	// by WFP: WFP may only permit the loopback bridge.
	if plan.ProxyAddress != "127.0.0.1:18080" {
		t.Fatalf("ProxyAddress = %q", plan.ProxyAddress)
	}
}

func TestQuoteArg(t *testing.T) {
	cases := []struct{ in, want string }{
		{`server.js`, `server.js`},
		{``, `""`},
		{`hello world`, `"hello world"`},
		{`say "hi"`, `"say \"hi\""`},
		// Backslashes not adjacent to a quote are literal (MS rules).
		{`trailing\ here`, `"trailing\ here"`},
		// A backslash immediately before the closing quote doubles.
		{`ends with backslash\`, `"ends with backslash\\"`},
		{`a\"b`, `"a\\\"b"`},
		{`C:\Program Files\app.exe`, `"C:\Program Files\app.exe"`},
	}
	for _, c := range cases {
		if got := quoteArg(c.in); got != c.want {
			t.Errorf("quoteArg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildCommandLine(t *testing.T) {
	got := buildCommandLine([]string{`C:\Program Files\node\node.exe`, `server.js`, `--flag`})
	want := `"C:\Program Files\node\node.exe" server.js --flag`
	if got != want {
		t.Fatalf("buildCommandLine = %q, want %q", got, want)
	}
}

func TestBuildEnvBlockDenyByDefault(t *testing.T) {
	parent := []string{"PATH=C:\\Windows", "SECRET=treasure", "GITHUB_TOKEN=abc", "GITHUB_TOKEN=def"}
	block := buildEnvBlock(parent, []string{"GITHUB_TOKEN"}, proxyEnvPairs("127.0.0.1:18080"))
	for _, banned := range []string{"SECRET", "PATH=", "DUPLICATE"} {
		if strings.Contains(block, banned) {
			t.Fatalf("env block leaked %q: %q", banned, block)
		}
	}
	if !strings.Contains(block, "GITHUB_TOKEN=def") {
		t.Fatalf("env block missing allowed var: %q", block)
	}
	if !strings.Contains(block, "HTTP_PROXY=http://127.0.0.1:18080") {
		t.Fatalf("env block missing proxy vars: %q", block)
	}
	// Duplicate allowed names keep the last value.
	if strings.Contains(block, "GITHUB_TOKEN=abc") || !strings.Contains(block, "GITHUB_TOKEN=def") {
		t.Fatalf("duplicate handling wrong: %q", block)
	}
	// Properly double-NUL terminated.
	if !strings.HasSuffix(block, "\x00\x00") {
		t.Fatal("env block not NUL-terminated")
	}
}

func TestIsRuntimeWriteDenied(t *testing.T) {
	denied := []string{`C:\Windows`, `C:\Windows\System32\config`, `c:\program files\foo`, `C:\Program Files (x86)\app`, `C:\ProgramData\x`}
	for _, p := range denied {
		if !IsRuntimeWriteDenied(p) {
			t.Errorf("IsRuntimeWriteDenied(%q) = false, want true", p)
		}
	}
	allowed := []string{`C:\data`, `C:\Users\me\AppData\Local\Temp`, `C:\tools`}
	for _, p := range allowed {
		if IsRuntimeWriteDenied(p) {
			t.Errorf("IsRuntimeWriteDenied(%q) = true, want false", p)
		}
	}
}

func TestAppContainerSIDShape(t *testing.T) {
	sid := appContainerSID("warden.test")
	if !strings.HasPrefix(sid, "S-1-15-3-") {
		t.Fatalf("sid %q not AppContainer-shaped", sid)
	}
	parts := strings.Split(sid, "-")
	if len(parts) != 8 {
		t.Fatalf("sid %q should have 8 components", sid)
	}
	// Deterministic.
	if sid != appContainerSID("warden.test") {
		t.Fatal("sid derivation not deterministic")
	}
	// Distinct monikers derive distinct SIDs.
	if sid == appContainerSID("warden.other") {
		t.Fatal("distinct monikers derived the same SID")
	}
}
