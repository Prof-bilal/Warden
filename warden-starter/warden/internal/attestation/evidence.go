// Package attestation binds creator test results to reviewed bytes and a trusted
// Ed25519 issuer. A signature attests provenance, not absence of vulnerabilities.
package attestation

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/warden-sandbox/warden/internal/mcpbridge"
	"github.com/warden-sandbox/warden/internal/policy"
)

type Check struct {
	Name     string `json:"name"`
	Decision string `json:"decision"`
	Passed   bool   `json:"passed"`
}
type Report struct {
	Version        int       `json:"version"`
	ArtifactSHA256 string    `json:"artifact_sha256"`
	PolicySHA256   string    `json:"policy_sha256"`
	RulesSHA256    string    `json:"rules_sha256"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	CommandSHA256  string    `json:"command_sha256"`
	WardenSHA256   string    `json:"warden_sha256"`
	WardenVersion  string    `json:"warden_version"`
	Platform       string    `json:"platform"`
	Backend        string    `json:"backend"`
	Transport      string    `json:"transport"`
	Protocol       string    `json:"protocol"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Issuer         string    `json:"issuer"`
	Checks         []Check   `json:"checks"`
}
type Envelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	KeyID     string `json:"key_id"`
}
type Manifest struct {
	Version int    `json:"version"`
	Checks  []Test `json:"checks"`
}
type Test struct {
	Name     string          `json:"name"`
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params"`
	Decision string          `json:"decision"`
	Contains string          `json:"contains,omitempty"`
}
type Revocations struct {
	Version   int       `json:"version"`
	ExpiresAt time.Time `json:"expires_at"`
	Evidence  []string  `json:"revoked_evidence"`
	Artifacts []string  `json:"revoked_artifacts"`
}

func FileHash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, 1024*1024*1024+1))
	if e != nil {
		return "", e
	}
	if n > 1024*1024*1024 {
		return "", fmt.Errorf("file exceeds 1 GiB")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// TreeHash binds every file and internal symlink under a dedicated artifact
// directory. External symlinks, special files and oversized trees fail closed.
func TreeHash(path string) (string, error) {
	root, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		return "", e
	}
	info, e := os.Stat(root)
	if e != nil || !info.IsDir() {
		return "", fmt.Errorf("artifact must be a dedicated directory")
	}
	var rows []string
	var total int64
	e = filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if path == root || d.IsDir() {
			return nil
		}
		if len(rows) >= 20000 {
			return fmt.Errorf("artifact file limit")
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.Type()&os.ModeSymlink != 0 {
			target, e := os.Readlink(path)
			if e != nil {
				return e
			}
			resolved, e := filepath.EvalSymlinks(path)
			if e != nil || !policy.Within(resolved, root) {
				return fmt.Errorf("artifact symlink escapes tree")
			}
			row, _ := json.Marshal([]string{rel, "symlink", filepath.ToSlash(target)})
			rows = append(rows, string(row))
			return nil
		}
		i, e := d.Info()
		if e != nil || !i.Mode().IsRegular() {
			return fmt.Errorf("artifact contains special file")
		}
		total += i.Size()
		if total > 1024*1024*1024 {
			return fmt.Errorf("artifact exceeds 1 GiB")
		}
		digest, e := FileHash(path)
		if e != nil {
			return e
		}
		row, _ := json.Marshal([]string{rel, "file", digest})
		rows = append(rows, string(row))
		return nil
	})
	if e != nil {
		return "", e
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("empty artifact tree")
	}
	sort.Strings(rows)
	return mcpbridge.Hash([]byte(strings.Join(rows, "\n") + "\n")), nil
}
func DecodeFile(path string, out any) ([]byte, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	if len(b) > 1024*1024 || mcpbridge.StrictJSON(b) != nil {
		return nil, fmt.Errorf("invalid or oversized JSON")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		return nil, e
	}
	return b, nil
}
func Key(path string, private bool) ([]byte, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	key, e := hex.DecodeString(strings.TrimSpace(string(b)))
	size := ed25519.PublicKeySize
	if private {
		size = ed25519.PrivateKeySize
	}
	if e != nil || len(key) != size {
		return nil, fmt.Errorf("invalid Ed25519 key")
	}
	if private {
		derived := ed25519.NewKeyFromSeed(key[:32])
		if !ed25519.PrivateKey(key).Equal(derived) {
			return nil, fmt.Errorf("inconsistent private key")
		}
	}
	return key, nil
}
func Sign(r Report, key ed25519.PrivateKey) ([]byte, error) {
	r.Issuer = mcpbridge.Hash(key.Public().(ed25519.PublicKey))
	payload, e := json.Marshal(r)
	if e != nil {
		return nil, e
	}
	return json.MarshalIndent(Envelope{base64.RawURLEncoding.EncodeToString(payload), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, payload)), r.Issuer}, "", "  ")
}

type Inputs struct {
	Artifact, Policy, Rules, Manifest, PublicKey, Revocations, Warden string
	Command                                                           []string
}

func Verify(envelope string, in Inputs, now time.Time) (Report, string, error) {
	var out Report
	var env Envelope
	if _, e := DecodeFile(envelope, &env); e != nil {
		return out, "invalid", e
	}
	pub, e := Key(in.PublicKey, false)
	if e != nil {
		return out, "untrusted", e
	}
	payload, e := base64.RawURLEncoding.DecodeString(env.Payload)
	if e != nil || mcpbridge.StrictJSON(payload) != nil {
		return out, "invalid", fmt.Errorf("invalid payload")
	}
	sig, e := base64.RawURLEncoding.DecodeString(env.Signature)
	if e != nil || env.KeyID != mcpbridge.Hash(pub) || !ed25519.Verify(pub, payload, sig) {
		return out, "untrusted", fmt.Errorf("signature or trusted issuer mismatch")
	}
	if json.Unmarshal(payload, &out) != nil || out.Version != 1 || out.Issuer != env.KeyID {
		return out, "invalid", fmt.Errorf("invalid report schema")
	}
	if out.IssuedAt.After(now.Add(time.Minute)) || !out.ExpiresAt.After(now) || out.ExpiresAt.Sub(out.IssuedAt) > 14*24*time.Hour || !out.ExpiresAt.After(out.IssuedAt) {
		return out, "expired", fmt.Errorf("evidence expired or timestamp invalid")
	}
	if out.Platform != runtime.GOOS+"/"+runtime.GOARCH {
		return out, "platform-mismatch", fmt.Errorf("evidence is bound to another platform")
	}
	if in.Revocations == "" {
		return out, "revocation-unchecked", fmt.Errorf("fresh trusted --revocations required")
	}
	var rev Revocations
	if _, e = DecodeFile(in.Revocations, &rev); e != nil || rev.Version != 1 || !rev.ExpiresAt.After(now) {
		return out, "revocation-unchecked", fmt.Errorf("invalid or expired trusted revocation snapshot")
	}
	for _, hash := range rev.Evidence {
		if hash == mcpbridge.Hash(payload) {
			return out, "revoked", fmt.Errorf("evidence revoked")
		}
	}
	for _, hash := range rev.Artifacts {
		if hash == out.ArtifactSHA256 {
			return out, "revoked", fmt.Errorf("artifact revoked")
		}
	}
	artifact, e := TreeHash(in.Artifact)
	if e != nil {
		return out, "mismatch", e
	}
	command, _ := json.Marshal(in.Command)
	if artifact != out.ArtifactSHA256 || mcpbridge.Hash(command) != out.CommandSHA256 {
		return out, "mismatch", fmt.Errorf("artifact or launcher changed")
	}
	for _, pair := range [][2]string{{in.Policy, out.PolicySHA256}, {in.Rules, out.RulesSHA256}, {in.Manifest, out.ManifestSHA256}, {in.Warden, out.WardenSHA256}} {
		hash, e := FileHash(pair[0])
		if e != nil || hash != pair[1] {
			return out, "mismatch", fmt.Errorf("policy, rules, manifest or Warden binary changed")
		}
	}
	allowed, denied := false, false
	for _, check := range out.Checks {
		if !check.Passed {
			return out, "failed", fmt.Errorf("failed check")
		}
		if check.Decision == "allow" {
			allowed = true
		}
		if check.Decision == "deny" {
			denied = true
		}
	}
	if !allowed || !denied || out.Transport != "stdio" || out.Protocol != mcpbridge.Protocol || out.Backend == "" || len(out.Checks) < 2 {
		return out, "invalid", fmt.Errorf("allowed/denied workflow evidence required")
	}
	return out, "checks-passing", nil
}
func Badge(status string) string {
	label := "policy included"
	color := "#4063D8"
	if status == "checks-passing" {
		label = "checks passing"
		color = "#24845C"
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="218" height="28" role="img" aria-label="Warden: %s"><rect width="84" height="28" fill="#181C25"/><rect x="84" width="134" height="28" fill="%s"/><g fill="white" font-family="Verdana,sans-serif" font-size="11" text-anchor="middle"><text x="42" y="18">Warden</text><text x="151" y="18">%s</text></g></svg>`, label, color, label)
}
