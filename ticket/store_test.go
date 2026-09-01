package ticket

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// referenceInstant is the fixed clock every time-dependent test injects. The
// corpus records its expectations against it, and a test that read the system
// clock would start failing on its own.
var referenceInstant = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

func fixedClock() func() time.Time {
	return func() time.Time { return referenceInstant }
}

func TestInitCreatesAStore(t *testing.T) {
	root := t.TempDir()
	s, err := Init(root, InitOptions{
		Actor: Actor{ID: "human:sothr", Name: "Drew Short"},
		Now:   fixedClock(),
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if want := filepath.Join(root, StoreDirName); s.Path() != want {
		t.Errorf("store path = %s, want %s", s.Path(), want)
	}
	for _, p := range []string{"config.yml", "README.md", "tickets", "archive"} {
		if _, err := os.Stat(filepath.Join(s.Path(), p)); err != nil {
			t.Errorf("init did not create %s: %v", p, err)
		}
	}
	if got := s.Config().Actors; len(got) != 1 || got[0].ID != "human:sothr" {
		t.Errorf("config actors = %+v, want the init actor", got)
	}
	if got := s.Config().Lock.Timeout.Duration(); got != DefaultLockTimeout {
		t.Errorf("lock timeout = %s, want %s", got, DefaultLockTimeout)
	}

	if _, err := Init(root, InitOptions{}); CodeOf(err) != CodeStoreExists {
		t.Errorf("second init = %v, want %s", err, CodeStoreExists)
	}
}

func TestOpenMissingStore(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "nope"))
	if CodeOf(err) != CodeStoreNotFound {
		t.Errorf("open of a missing store = %v, want %s", err, CodeStoreNotFound)
	}
}

func TestDiscoverWalksUpToTheGitRoot(t *testing.T) {
	// outside/ holds a store that must never be found, because the walk stops
	// at the repository root below it.
	outside := t.TempDir()
	if _, err := Init(outside, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(outside, "repo")
	nested := filepath.Join(repo, "internal", "auth")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	// A worktree has a .git file rather than a directory; either marks a root.
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Discover(nested); CodeOf(err) != CodeStoreNotFound {
		t.Errorf("discover crossed the repository root: %v", err)
	}

	if _, err := Init(repo, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	s, err := Discover(nested)
	if err != nil {
		t.Fatalf("discover from a nested directory: %v", err)
	}
	if want := filepath.Join(repo, StoreDirName); s.Path() != want {
		t.Errorf("discovered %s, want %s", s.Path(), want)
	}
}

// TestConfigRoundTrip pins config.yml against the corpus: the config this tool
// writes is the config the fixtures were written by hand to contain.
func TestConfigRoundTrip(t *testing.T) {
	path := filepath.Join(corpusDir, "stores", "clean", "store", "config.yml")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseConfig(want)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if got := string(RenderConfig(cfg)); got != string(want) {
		t.Errorf("render(parse(config.yml)) != config.yml\n%s", diffLines(string(want), got))
	}
	if !cfg.KnownLabel("auth") || cfg.KnownLabel("frobnicate") {
		t.Errorf("label allowlist = %v, want auth known and frobnicate unknown", cfg.Labels)
	}
}

func TestStoreOutsideARepositoryHasNoRoot(t *testing.T) {
	root := t.TempDir()
	s, err := Init(root, InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// A temp directory is not inside a repository, so there is nothing for a
	// references path to resolve against.
	if got := s.Root(); got != "" {
		t.Errorf("root = %q, want empty for a store outside a repository", got)
	}
}
