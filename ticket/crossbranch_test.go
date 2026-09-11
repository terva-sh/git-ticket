package ticket

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// symlinkCommitDate is when every commit in these tests was made. The ref scan
// skips a ref whose last commit is older than crossBranchWindow, and it measures
// that against the store's injected clock rather than the machine's. A commit
// dated by the machine would put the window's edge wherever the suite happened
// to run, so this sits five days before referenceInstant and every ref is read.
const symlinkCommitDate = "2026-09-25T00:00:00Z"

// newSymlinkedCrossRepo builds a repository holding one ticket on main and one
// on another branch, and returns the store directory spelled two ways: once
// through the real path and once through a symlink standing in front of it.
//
// The temporary directory is resolved before anything is built under it. macOS
// reaches TempDir through /var, which is a symlink to /private/var, so without
// that step the "real" spelling is a symlinked one too and both halves of this
// test would describe the same case.
func newSymlinkedCrossRepo(t *testing.T) (real, linked string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	root := t.TempDir()
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	repo := filepath.Join(root, "real", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
			"GIT_AUTHOR_DATE="+symlinkCommitDate,
			"GIT_COMMITTER_DATE="+symlinkCommitDate,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	git("init", "-q", "-b", "main")
	s, err := Init(repo, InitOptions{Actor: testActor, Now: fixedClock()})
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	file := func(title string) {
		t.Helper()
		if _, err := s.Create(context.Background(), CreateOptions{Title: title, Actor: testActor}); err != nil {
			t.Fatalf("create %q: %v", title, err)
		}
	}
	file("on main")
	git("add", "-A")
	git("commit", "-qm", "the store")
	git("switch", "-q", "-c", "other")
	file("only on the other branch")
	git("add", "-A")
	git("commit", "-qm", "one more")
	// Back to main, so the working tree carries the first ticket alone and the
	// second is reachable only by reading another ref.
	git("switch", "-q", "main")

	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "real"), link); err != nil {
		t.Skipf("this platform will not make a symlink: %v", err)
	}
	return filepath.Join(repo, StoreDirName), filepath.Join(link, "repo", StoreDirName)
}

// crossTitlesAt opens the store at path and lists it across branches.
func crossTitlesAt(t *testing.T, path string) []string {
	t.Helper()
	s, err := OpenWith(path, OpenOptions{Now: fixedClock()})
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	found, err := s.List(context.Background(), Filter{All: true, CrossBranch: true})
	if err != nil {
		t.Fatalf("list %s: %v", path, err)
	}
	titles := make([]string, 0, len(found))
	for _, tk := range found {
		titles = append(titles, tk.Title)
	}
	return titles
}

// TestCrossBranchReadsAStoreReachedThroughASymlink is the regression for
// TKT-01M2938X37. A cross-branch query against a store whose path still spells a
// symlink used to answer with the working tree alone: exit 0, no finding, and
// half the store missing.
//
// storePathspec took filepath.Rel of two paths in different name spaces. Root()
// is git rev-parse --show-toplevel, which resolves symlinks, and the store path
// is whatever the caller opened, which does not. The pathspec came out as a
// string of ../.. , ls-tree matched nothing, and the scan reported no other ref
// had any tickets at all.
//
// The resolved subtest is the control and is not decoration. Both spellings name
// one repository, so if the symlinked half were to pass for some reason other
// than the fix, the control is what says the repository was built correctly and
// the query works at all.
func TestCrossBranchReadsAStoreReachedThroughASymlink(t *testing.T) {
	real, linked := newSymlinkedCrossRepo(t)
	want := []string{"on main", "only on the other branch"}

	t.Run("through the resolved path", func(t *testing.T) {
		got := crossTitlesAt(t, real)
		if !sameTitles(got, want) {
			t.Fatalf("resolved spelling read %v, want %v", got, want)
		}
	})

	t.Run("through a symlinked path", func(t *testing.T) {
		got := crossTitlesAt(t, linked)
		if !sameTitles(got, want) {
			t.Fatalf("symlinked spelling read %v, want %v; the other branch was not read", got, want)
		}
	})
}

// sameTitles compares two title sets without minding the order. IDs minted in
// one millisecond do not sort by creation, so the order these come back in is
// not the order they were filed.
func sameTitles(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, g := range got {
		seen[g]++
	}
	for _, w := range want {
		if seen[w] == 0 {
			return false
		}
		seen[w]--
	}
	return true
}
