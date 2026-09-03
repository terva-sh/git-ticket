package ticket

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig puts a valid config.yml in dir, which is what makes dir a store
// per plan 4.
func writeConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, configFile), RenderConfig(DefaultConfig()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOpenRefusesADirectoryThatIsNotAStore holds the rule in plan 4. Before it,
// existence was the whole test, so any directory opened as a store reporting no
// tickets.
//
// The empty directory is the case that matters. A caller that lists and sees
// nothing concludes there is no work, and nothing in that answer says it was
// handed the wrong path.
func TestOpenRefusesADirectoryThatIsNotAStore(t *testing.T) {
	cases := []struct {
		name  string
		build func(t *testing.T, dir string)
	}{
		{"empty", func(*testing.T, string) {}},
		{"unrelated files", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"the store directories but no config", func(t *testing.T, dir string) {
			for _, d := range storeDirs {
				if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"config.yml is a directory", func(t *testing.T, dir string) {
			if err := os.MkdirAll(filepath.Join(dir, configFile), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			c.build(t, dir)

			s, err := Open(dir)
			if err == nil {
				t.Fatalf("opened %s as a store holding %d tickets, want %s",
					dir, len(mustList(t, s)), CodeStoreNotFound)
			}
			if CodeOf(err) != CodeStoreNotFound {
				t.Errorf("open = %v, want %s", err, CodeStoreNotFound)
			}
			// The message names the file it wanted, because "no ticket store
			// here" leaves a reader guessing what would make it one.
			if !strings.Contains(err.Error(), configFile) {
				t.Errorf("error does not name %s: %v", configFile, err)
			}
		})
	}
}

// TestOpenAcceptsAStoreWithNoSubdirectories is the regression behind the rule
// being config.yml alone.
//
// The marker was first settled as config.yml and tickets/ together. Git tracks
// no empty directory, so a store whose open work is finished loses tickets/ on
// the next clone, and requiring it rejected a real store for having nothing in
// progress.
func TestOpenAcceptsAStoreWithNoSubdirectories(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir)

	s, err := Open(dir)
	if err != nil {
		t.Fatalf("open a store with config.yml and no directories: %v", err)
	}
	if got := mustList(t, s); len(got) != 0 {
		t.Errorf("listed %d tickets in an empty store", len(got))
	}
}

// TestAClonedStoreOpens runs the case that overruled the first answer, end to
// end through git rather than by reasoning about it.
//
// Init writes four directories and three files. A commit records the files and
// none of the directories, because git tracks no empty directory, so the clone
// has a store that any directory-based marker would reject.
func TestAClonedStoreOpens(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	origin := filepath.Join(root, "origin")
	if err := os.MkdirAll(origin, 0o755); err != nil {
		t.Fatal(err)
	}
	git(origin, "init", "-q", "-b", "main")
	if _, err := Init(origin, InitOptions{Actor: testActor, Now: fixedClock()}); err != nil {
		t.Fatalf("init store: %v", err)
	}
	git(origin, "add", "-A")
	git(origin, "commit", "-qm", "a store with no tickets yet")
	git(root, "clone", "-q", origin, "clone")

	cloned := filepath.Join(root, "clone", StoreDirName)
	// Assert the premise, so this cannot pass for the wrong reason. If git ever
	// tracked empty directories the clone would carry tickets/ and the test
	// below would prove nothing about the marker.
	if _, err := os.Stat(filepath.Join(cloned, ticketsDir)); err == nil {
		t.Fatalf("the clone has %s/, so this test no longer covers the case it was written for", ticketsDir)
	}
	if _, err := Open(cloned); err != nil {
		t.Fatalf("open a freshly cloned store: %v", err)
	}
}

// mustList is List with the error already handled, for assertions that care
// about the count rather than the failure.
func mustList(t *testing.T, s *Store) []*Ticket {
	t.Helper()
	got, err := s.List(context.Background(), Filter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return got
}
