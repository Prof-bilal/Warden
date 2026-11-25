package guard

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestFetchNPMArchiveUsesInspectedBytes(t *testing.T) {
	archive := createNPMGuardArchive(t, npmArchiveFile{
		"package/package.json",
		`{"name":"@example/tool","version":"1.0.0","scripts":{"postinstall":"node setup"}}`,
	})
	sum := sha512.Sum512(archive)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/@example/tool":
			_, _ = fmt.Fprintf(w, `{"name":"@example/tool","versions":{"1.0.0":{"name":"@example/tool","version":"1.0.0","dist":{"tarball":%q,"integrity":%q}}}}`,
				"http://"+r.Host+"/tool.tgz", integrity)
		case "/tool.tgz":
			if got := r.Header.Get("Accept-Encoding"); got != "identity" {
				t.Errorf("archive download encoding = %q, want identity", got)
			}
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	release, err := ReadNPMRelease(ctx, server.Client(), server.URL, "@example/tool", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	got, facts, err := FetchNPMArchive(ctx, server.Client(), server.URL, release)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, archive) || !slices.Equal(facts.InstallScripts, []string{"postinstall"}) {
		t.Fatalf("download lost archive bytes or install facts: %+v", facts)
	}
	release.Tarball = "http://other-registry.invalid/tool.tgz"
	if _, _, err := FetchNPMArchive(ctx, server.Client(), server.URL, release); err == nil {
		t.Fatal("expected archive URL outside the registry to be rejected")
	}
	_, got, decision, err := ReviewNPMInstall(ctx, server.Client(), server.URL, "@example/tool", "1.0.0", time.Now(), ReleaseRules{})
	if err != nil || len(got) != 0 || len(decision.Blocks) != 1 {
		t.Fatalf("archive lifecycle check must block installation: decision %+v, error %v", decision, err)
	}
	rules := ReleaseRules{ScriptAllowlist: []string{"@example/tool@1.0.0"}}
	_, got, decision, err = ReviewNPMInstall(ctx, server.Client(), server.URL, "@example/tool", "1.0.0", time.Now(), rules)
	if err != nil || !bytes.Equal(got, archive) || len(decision.Blocks) != 0 {
		t.Fatalf("exact allowlist must return reviewed archive: decision %+v, error %v", decision, err)
	}
}

type npmArchiveFile struct {
	name string
	body string
}

func createNPMGuardArchive(t *testing.T, files ...npmArchiveFile) []byte {
	t.Helper()
	var out bytes.Buffer
	compressed := gzip.NewWriter(&out)
	archive := tar.NewWriter(compressed)
	for _, file := range files {
		if err := archive.WriteHeader(&tar.Header{
			Name: file.name, Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(file.body)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(archive, file.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestInspectNPMArchive(t *testing.T) {
	manifest := npmArchiveFile{"package/package.json", `{"name":"@example/tool","version":"1.0.0","scripts":{"install":"build","test":"test"}}`}
	archive := createNPMGuardArchive(t, manifest, npmArchiveFile{"package/binding.gyp", "{}"})
	sum := sha512.Sum512(archive)
	release := NPMRelease{
		Facts:     ReleaseFacts{Name: "@example/tool", Version: "1.0.0", InstallScripts: []string{"preinstall"}},
		Integrity: "sha512-" + base64.StdEncoding.EncodeToString(sum[:]),
	}
	facts, err := InspectNPMArchive(release, archive)
	if err != nil {
		t.Fatal(err)
	}
	if !facts.HasBindingGYP || !slices.Equal(facts.InstallScripts, []string{"preinstall", "install"}) {
		t.Fatalf("archive facts = %+v", facts)
	}
	if _, err := InspectNPMArchive(release, append(archive, 0)); err == nil {
		t.Fatal("expected archive integrity mismatch")
	}
	release.Integrity = ""
	if _, err := InspectNPMArchive(release, createNPMGuardArchive(t, manifest, manifest)); err == nil {
		t.Fatal("expected duplicate manifest to be rejected")
	}
	release.Facts.Version = "2.0.0"
	if _, err := InspectNPMArchive(release, archive); err == nil {
		t.Fatal("expected mismatched release identity to be rejected")
	}
}

func TestNPMIntegrityPrefersStrongestDigest(t *testing.T) {
	data := []byte("archive bytes")
	weak := sha256.Sum256(data)
	strong := sha512.Sum512([]byte("different archive bytes"))
	integrity := "sha256-" + base64.StdEncoding.EncodeToString(weak[:]) +
		" sha512-" + base64.StdEncoding.EncodeToString(strong[:])
	if err := checkNPMIntegrity(integrity, data); err == nil {
		t.Fatal("expected mismatched SHA-512 to take precedence over matching SHA-256")
	}
}

func TestNPMArchiveOfflineEligibility(t *testing.T) {
	release := NPMRelease{Facts: ReleaseFacts{Name: "fixture", Version: "1.0.0"}}
	for _, tc := range []struct {
		name, declaration string
		shrinkwrap        bool
		offline           bool
	}{
		{name: "standalone", offline: true},
		{name: "empty dependencies", declaration: `,"dependencies":{}`, offline: true},
		{name: "regular", declaration: `,"dependencies":{"other":"1.0.0"}`},
		{name: "optional", declaration: `,"optionalDependencies":{"other":"1.0.0"}`},
		{name: "peer", declaration: `,"peerDependencies":{"other":"1.0.0"}`},
		{name: "bundled", declaration: `,"bundleDependencies":["other"]`},
		{name: "shrinkwrap", shrinkwrap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := []npmArchiveFile{{"package/package.json", `{"name":"fixture","version":"1.0.0"` + tc.declaration + "}"}}
			if tc.shrinkwrap {
				files = append(files, npmArchiveFile{"package/npm-shrinkwrap.json", "{}"})
			}
			facts, err := InspectNPMArchive(release, createNPMGuardArchive(t, files...))
			if err != nil || facts.NPMCanInstallOffline != tc.offline {
				t.Fatalf("offline = %v, error %v", facts.NPMCanInstallOffline, err)
			}
		})
	}
}
