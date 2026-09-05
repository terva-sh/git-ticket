package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeRelease serves a GitHub-shaped latest release from httptest: the
// JSON at the API path, and every asset under /dl/. No test in this
// file reaches the real network, which is the promise the suite makes.
func fakeRelease(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc(releasePath, func(w http.ResponseWriter, r *http.Request) {
		type asset struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		}
		rel := struct {
			TagName string  `json:"tag_name"`
			Assets  []asset `json:"assets"`
		}{TagName: tag}
		for name := range assets {
			rel.Assets = append(rel.Assets, asset{Name: name, URL: srv.URL + "/dl/" + name})
		}
		json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/dl/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/dl/")
		body, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// asRelease makes the test binary report a version, because an in-tree
// test build reports devel and would only ever exercise the refusal.
func asRelease(t *testing.T, version string) {
	t.Helper()
	old := runningVersion
	runningVersion = func() string { return version }
	t.Cleanup(func() { runningVersion = old })
}

// runSelfUpdateCLI invokes the CLI with the release API and the target
// executable injected.
func runSelfUpdateCLI(t *testing.T, api, executable string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, Env{
		Dir:        t.TempDir(),
		Getenv:     func(string) string { return "" },
		Stdout:     &stdout,
		Stderr:     &stderr,
		ReleaseAPI: api,
		Executable: executable,
	})
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// tarball packs one file at the archive root, the shape
// .goreleaser.yaml publishes.
func tarball(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func checksumLine(name string, body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]) + "  " + name + "\n"
}

// TestSelfUpdateCheckGradesTheGap is the graded bucket of plan 10.2:
// the exit status names the highest version component that moved.
func TestSelfUpdateCheckGradesTheGap(t *testing.T) {
	cases := []struct {
		current, latest string
		gap             string
		code            int
	}{
		{"v0.7.1", "v0.7.1", "none", 0},
		{"v0.7.1", "v0.7.2", "patch", 10},
		{"v0.7.1", "v0.8.0", "minor", 11},
		{"v0.7.1", "v1.0.0", "major", 12},
		// Ahead of the latest happens between a tag push and the
		// release job finishing; treating it as an update would
		// downgrade, per plan 12.6.
		{"v0.9.0", "v0.8.0", "none", 0},
	}
	for _, tc := range cases {
		t.Run(tc.current+"_to_"+tc.latest, func(t *testing.T) {
			asRelease(t, tc.current)
			srv := fakeRelease(t, tc.latest, nil)
			res := runSelfUpdateCLI(t, srv.URL, "", "self-update", "--check")
			if res.code != tc.code {
				t.Errorf("exit %d, want %d; stderr: %s", res.code, tc.code, res.stderr)
			}
			if tc.gap == "none" && !strings.Contains(res.stdout, "up to date") {
				t.Errorf("up-to-date run did not say so: %q", res.stdout)
			}
			if tc.gap != "none" && !strings.Contains(res.stdout, tc.gap+" update is available") {
				t.Errorf("output does not name the %s gap: %q", tc.gap, res.stdout)
			}
		})
	}
}

// TestSelfUpdateCheckJSONEnvelope is the self-update kind of plan
// 10.7: asset and target stay null under --check, and applied can
// never be true in a no-op mode.
func TestSelfUpdateCheckJSONEnvelope(t *testing.T) {
	asRelease(t, "v0.7.1")
	srv := fakeRelease(t, "v0.8.0", nil)
	res := runSelfUpdateCLI(t, srv.URL, "", "self-update", "--check", "--json")
	if res.code != 11 {
		t.Fatalf("exit %d, want 11; stderr: %s", res.code, res.stderr)
	}
	var env selfUpdateEnvelope
	if err := json.Unmarshal([]byte(res.stdout), &env); err != nil {
		t.Fatalf("stdout is not one envelope: %v\n%s", err, res.stdout)
	}
	if env.Kind != "self-update" || env.SchemaVersion != schemaVersion {
		t.Errorf("kind %q schemaVersion %d", env.Kind, env.SchemaVersion)
	}
	if env.Current != "v0.7.1" || env.Latest != "v0.8.0" || env.Gap != "minor" {
		t.Errorf("current %q latest %q gap %q", env.Current, env.Latest, env.Gap)
	}
	if env.Asset != nil || env.Target != nil {
		t.Error("--check resolved asset or target; plan 10.7 says it stops before either")
	}
	if env.Applied {
		t.Error("a no-op mode reported applied: true")
	}
}

// TestSelfUpdateDryRunNamesAssetAndTarget: the second no-op mode
// resolves what the update would touch and still writes nothing.
func TestSelfUpdateDryRunNamesAssetAndTarget(t *testing.T) {
	asRelease(t, "v0.7.1")
	srv := fakeRelease(t, "v0.8.0", nil)
	target := filepath.Join(t.TempDir(), "git-ticket")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := runSelfUpdateCLI(t, srv.URL, target, "self-update", "--dry-run")
	if res.code != 11 {
		t.Fatalf("exit %d, want 11; stderr: %s", res.code, res.stderr)
	}
	wantAsset := fmt.Sprintf("git-ticket_0.8.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	if !strings.Contains(res.stdout, wantAsset) || !strings.Contains(res.stdout, target) {
		t.Errorf("dry-run does not name %s and %s:\n%s", wantAsset, target, res.stdout)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "old binary" {
		t.Errorf("dry-run touched the target: %q, %v", got, err)
	}
}

// TestSelfUpdateApplies is the whole path: download, verify, extract,
// atomic replace.
func TestSelfUpdateApplies(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake asset is a tar.gz; windows wants the zip path")
	}
	asRelease(t, "v0.7.1")
	newBin := []byte("#!/bin/true new binary\n")
	asset := fmt.Sprintf("git-ticket_0.8.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive := tarball(t, "git-ticket", newBin)
	srv := fakeRelease(t, "v0.8.0", map[string][]byte{
		asset:           archive,
		"checksums.txt": []byte(checksumLine(asset, archive)),
	})
	target := filepath.Join(t.TempDir(), "git-ticket")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := runSelfUpdateCLI(t, srv.URL, target, "self-update")
	if res.code != 0 {
		t.Fatalf("exit %d; stderr: %s", res.code, res.stderr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newBin) {
		t.Errorf("target holds %q, want the new binary", got)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("replaced binary is not executable: %v", info.Mode())
	}
	if !strings.Contains(res.stdout, "updated") || !strings.Contains(res.stdout, "v0.8.0") {
		t.Errorf("apply did not report what it did: %q", res.stdout)
	}
}

// TestSelfUpdateChecksumMismatchRefuses: the sha256 gate holds before
// anything on disk moves.
func TestSelfUpdateChecksumMismatchRefuses(t *testing.T) {
	asRelease(t, "v0.7.1")
	asset := fmt.Sprintf("git-ticket_0.8.0_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive := tarball(t, "git-ticket", []byte("evil"))
	srv := fakeRelease(t, "v0.8.0", map[string][]byte{
		asset:           archive,
		"checksums.txt": []byte(strings.Repeat("0", 64) + "  " + asset + "\n"),
	})
	target := filepath.Join(t.TempDir(), "git-ticket")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := runSelfUpdateCLI(t, srv.URL, target, "self-update")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1", res.code)
	}
	if !strings.Contains(res.stderr, "checksum mismatch") {
		t.Errorf("stderr does not name the mismatch: %q", res.stderr)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old binary" {
		t.Errorf("a failed checksum still replaced the target: %q", got)
	}
}

// TestSelfUpdateDevelRefuses: a working-tree build is somebody's
// development tree, and the refusal names the repair.
func TestSelfUpdateDevelRefuses(t *testing.T) {
	asRelease(t, "devel")
	res := runSelfUpdateCLI(t, "http://127.0.0.1:0", "", "self-update", "--check")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1", res.code)
	}
	if !strings.Contains(res.stderr, "just install") {
		t.Errorf("refusal does not name the repair: %q", res.stderr)
	}
}

// TestSelfUpdateRejectsBothFlags: the two no-op modes are one question
// at two levels of detail.
func TestSelfUpdateRejectsBothFlags(t *testing.T) {
	asRelease(t, "v0.7.1")
	res := runSelfUpdateCLI(t, "http://127.0.0.1:0", "", "self-update", "--check", "--dry-run")
	if res.code != 1 {
		t.Fatalf("exit %d, want 1", res.code)
	}
}
