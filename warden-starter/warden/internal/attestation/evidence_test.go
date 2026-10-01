package attestation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/warden-sandbox/warden/internal/mcpbridge"
)

func TestBindingsTrustExpiryAndRevocation(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "artifact")
	_ = os.Mkdir(artifact, 0700)
	entry := filepath.Join(artifact, "entry")
	_ = os.WriteFile(entry, []byte("reviewed code"), 0600)
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	keyPath := filepath.Join(dir, "key.pub")
	_ = os.WriteFile(keyPath, []byte(hex.EncodeToString(pub)), 0600)
	in := Inputs{Artifact: artifact, Policy: filepath.Join(dir, "policy"), Rules: filepath.Join(dir, "rules"), Manifest: filepath.Join(dir, "manifest"), Warden: filepath.Join(dir, "warden"), PublicKey: keyPath, Revocations: filepath.Join(dir, "revocations"), Command: []string{entry}}
	for _, p := range []string{in.Policy, in.Rules, in.Manifest, in.Warden} {
		_ = os.WriteFile(p, []byte(p), 0600)
	}
	now := time.Now().UTC()
	writeRev := func(rev Revocations) { b, _ := json.Marshal(rev); _ = os.WriteFile(in.Revocations, b, 0600) }
	rev := Revocations{Version: 1, ExpiresAt: now.Add(time.Hour), Evidence: []string{}, Artifacts: []string{}}
	writeRev(rev)
	tree, _ := TreeHash(artifact)
	ph, _ := FileHash(in.Policy)
	rh, _ := FileHash(in.Rules)
	mh, _ := FileHash(in.Manifest)
	wh, _ := FileHash(in.Warden)
	argv, _ := json.Marshal(in.Command)
	r := Report{Version: 1, ArtifactSHA256: tree, PolicySHA256: ph, RulesSHA256: rh, ManifestSHA256: mh, WardenSHA256: wh, CommandSHA256: mcpbridge.Hash(argv), Platform: runtime.GOOS + "/" + runtime.GOARCH, Backend: "linux", Transport: "stdio", Protocol: mcpbridge.Protocol, IssuedAt: now, ExpiresAt: now.Add(time.Hour), Checks: []Check{{Name: "allowed", Decision: "allow", Passed: true}, {Name: "denied", Decision: "deny", Passed: true}}}
	envelope := filepath.Join(dir, "evidence.json")
	writeReport := func(r Report) {
		b, e := Sign(r, key)
		if e != nil {
			t.Fatal(e)
		}
		_ = os.WriteFile(envelope, b, 0600)
	}
	writeReport(r)
	check := func(want string) {
		t.Helper()
		_, status, e := Verify(envelope, in, now)
		if status != want || (want == "checks-passing" && e != nil) || (want != "checks-passing" && e == nil) {
			t.Fatalf("status %s error %v; want %s", status, e, want)
		}
	}
	check("checks-passing")
	_ = os.WriteFile(entry, []byte("changed"), 0600)
	check("mismatch")
	_ = os.WriteFile(entry, []byte("reviewed code"), 0600)
	check("checks-passing")
	original, _ := os.ReadFile(in.Policy)
	_ = os.WriteFile(in.Policy, []byte("new grants"), 0600)
	check("mismatch")
	_ = os.WriteFile(in.Policy, original, 0600)
	rev.Artifacts = []string{tree}
	writeRev(rev)
	check("revoked")
	rev.Artifacts = nil
	rev.ExpiresAt = now.Add(-time.Second)
	writeRev(rev)
	check("revocation-unchecked")
	rev.ExpiresAt = now.Add(time.Hour)
	writeRev(rev)
	r.ExpiresAt = now.Add(-time.Second)
	writeReport(r)
	check("expired")
	r.ExpiresAt = now.Add(time.Hour)
	writeReport(r)
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	_ = os.WriteFile(keyPath, []byte(hex.EncodeToString(other)), 0600)
	check("untrusted")
	_ = os.WriteFile(keyPath, []byte(hex.EncodeToString(pub)), 0600)
	var env Envelope
	_, _ = DecodeFile(envelope, &env)
	env.Payload = env.Payload[:len(env.Payload)-2] + "aa"
	b, _ := json.Marshal(env)
	_ = os.WriteFile(envelope, b, 0600)
	if _, _, e := Verify(envelope, in, now); e == nil {
		t.Fatal("forged payload accepted")
	}
}
func TestArtifactSymlinks(t *testing.T) {
	dir := t.TempDir()
	tree := filepath.Join(dir, "tree")
	_ = os.Mkdir(tree, 0700)
	_ = os.WriteFile(filepath.Join(tree, "code"), []byte("code"), 0600)
	if e := os.Symlink("code", filepath.Join(tree, "internal")); e != nil {
		t.Skip("symlinks unavailable")
	}
	if _, e := TreeHash(tree); e != nil {
		t.Fatal(e)
	}
	_ = os.WriteFile(filepath.Join(dir, "secret"), []byte("secret"), 0600)
	_ = os.Symlink("../secret", filepath.Join(tree, "external"))
	if _, e := TreeHash(tree); e == nil {
		t.Fatal("external code not bound")
	}
}
