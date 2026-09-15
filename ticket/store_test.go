package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestInitRefusesAStoreDirectoryAsRoot runs the mistake terva reported: reading
// Open's signature first, then passing the store path to Init. It used to build
// .tickets/.tickets and say nothing, and the first symptom arrived somewhere
// else entirely, when a Discover from inside the store found the buried one.
func TestInitRefusesAStoreDirectoryAsRoot(t *testing.T) {
	root := t.TempDir()
	s, err := Init(root, InitOptions{Actor: testActor, Now: fixedClock()})
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	_, err = Init(s.Path(), InitOptions{Actor: testActor, Now: fixedClock()})
	if CodeOf(err) != CodeInvalidRoot {
		t.Fatalf("init with the store path as root = %v, want %s", err, CodeInvalidRoot)
	}
	// The caller's next move is to pass the parent, so the message has to name
	// it. A code alone would leave them guessing at the same signature twice.
	if msg := err.Error(); !strings.Contains(msg, root) {
		t.Errorf("message does not name the parent to use instead: %s", msg)
	}
	if _, statErr := os.Stat(filepath.Join(s.Path(), StoreDirName)); !os.IsNotExist(statErr) {
		t.Error("a nested store was created despite the refusal")
	}
	// The store that was already there is untouched by the refusal.
	if _, err := Open(s.Path()); err != nil {
		t.Errorf("the real store no longer opens: %v", err)
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
	// v1.2 against v1.2.0 is the case the allowlist exists for: two spellings
	// of one milestone that nothing else can tell apart.
	if !cfg.KnownMilestone("v1.2") || cfg.KnownMilestone("v1.2.0") {
		t.Errorf("milestone allowlist = %v, want v1.2 known and v1.2.0 unknown", cfg.Milestones)
	}
	// A ticket naming no milestone is not a ticket naming a wrong one.
	if !cfg.KnownMilestone("") {
		t.Error("the empty milestone is always permitted")
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

// TestInitAdoptsADirectoryWithNoConfig covers the dead end plan 12.8 promised
// nobody would hit. `git am` of an export writes ticket files and no config.yml,
// and Init used to refuse because the directory was there while every read
// refused because the config was not. Two commands disagreed about the same
// directory and no CLI path repaired it.
//
// What resolves it is one definition of a store rather than a special case for
// adoption: config.yml is what Open keys on, so it is what Init keys on too.
func TestInitAdoptsADirectoryWithNoConfig(t *testing.T) {
	root := t.TempDir()
	landed := filepath.Join(root, StoreDirName, "draft")
	if err := os.MkdirAll(landed, 0o755); err != nil {
		t.Fatal(err)
	}
	// Written the way git am leaves it: a ticket file and nothing else.
	//
	// An epic rather than a task, because the epics index lists epics only. A
	// task arrives and renders the same empty index a fresh store gets, so it
	// cannot tell a rebuilt index from an unconditional one.
	body := "---\nschema: 3\nid: TKT-01M1PQ7TB0X4V2Z9J5K6M8N0Q1\ntitle: An arrived epic\n" +
		"type: epic\nstatus: draft\npriority: normal\n---\n\n## Description\n\nArrived by git am.\n"
	if err := os.WriteFile(filepath.Join(landed, "TKT-01M1PQ7TB0X4V2Z9J5K6M8N0Q1.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := Init(root, InitOptions{})
	if err != nil {
		t.Fatalf("Init refused a directory with no config: %v", err)
	}
	all, err := s.List(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("listing an adopted store: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("adopted store holds %d tickets, want the 1 that arrived", len(all))
	}

	// The epics index is built from what is here rather than written empty, so
	// an adopted store is not born reporting a staleness warning either, which
	// is the same rule a fresh store already got.
	parsed, err := s.load()
	if err != nil {
		t.Fatal(err)
	}
	if _, stale := s.epicsIndexStale(parsed); stale {
		t.Error("an adopted store was born with a stale epics index")
	}
}

// TestInitStillRefusesARealStore is the other half. The refusal exists to stop a
// second store clobbering a first, and widening it to adoption must not cost
// that.
func TestInitStillRefusesARealStore(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, InitOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, InitOptions{}); CodeOf(err) != CodeStoreExists {
		t.Errorf("second Init = %v, want %s", err, CodeStoreExists)
	}
}
