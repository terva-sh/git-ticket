package ticket

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newWorktreeRepo builds a repository with a committed store and adds a second
// worktree of it, returning the store directory in each. This is the shape the
// lock exists for: one repository, two working trees, an agent in each.
//
// The store is committed before the worktree is added, so the linked worktree
// checks out the same tickets rather than getting an empty directory. That is
// the real case. Two agents are looking at one ledger from two places.
func newWorktreeRepo(t *testing.T) (primary, linked string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	root := t.TempDir()
	main := filepath.Join(root, "main")
	if err := os.MkdirAll(main, 0o755); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		// A test must not depend on the developer's git identity, and commit
		// refuses without one.
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	git(main, "init", "-q", "-b", "main")
	if _, err := Init(main, InitOptions{Actor: testActor, Now: fixedClock()}); err != nil {
		t.Fatalf("init store: %v", err)
	}
	git(main, "add", "-A")
	git(main, "commit", "-qm", "the store")
	git(main, "worktree", "add", "-q", filepath.Join(root, "linked"), "-b", "other")

	return filepath.Join(main, ".tickets"), filepath.Join(root, "linked", ".tickets")
}

// TestWorktreesShareOneLock is the plan 14 acceptance criterion "worktrees
// sharing a Git directory share the lock", and the promise 7.3 makes that
// separate worktrees serialize.
//
// TestTwoProcessesOneWins does not cover this. Both of its processes open the
// same store directory, so it passes whether the lock is keyed on the Git
// common directory or on the store path. This test is the one that can tell
// those apart, which matters because the failure is silent: nothing errors,
// two agents in two worktrees simply both write.
func TestWorktreesShareOneLock(t *testing.T) {
	primary, linked := newWorktreeRepo(t)

	a, err := OpenWith(primary, OpenOptions{Now: fixedClock()})
	if err != nil {
		t.Fatalf("open the primary store: %v", err)
	}
	// A short timeout, so proving contention costs milliseconds rather than
	// the ten-second default.
	b, err := OpenWith(linked, OpenOptions{Now: fixedClock(), LockTimeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("open the linked store: %v", err)
	}

	// Two different store directories, which is what makes the shared lock a
	// claim worth testing rather than a tautology.
	if a.Path() == b.Path() {
		t.Fatalf("both stores are %s, so this proves nothing", a.Path())
	}
	if a.lockPath() != b.lockPath() {
		t.Errorf("the worktrees do not share a lock:\n  primary: %s\n  linked:  %s",
			a.lockPath(), b.lockPath())
	}

	// The behaviour the shared path buys. Holding it in one worktree stops a
	// write from the other.
	held, err := a.lock()
	if err != nil {
		t.Fatalf("taking the lock in the primary worktree: %v", err)
	}
	defer held.release()

	_, err = b.Create(context.Background(), CreateOptions{
		Title: "Written from the linked worktree",
		Actor: testActor,
	})
	if CodeOf(err) != CodeLockTimeout {
		t.Fatalf("a write from the linked worktree returned %v, want %s", err, CodeLockTimeout)
	}

	// And the lock is advisory in the right direction: released, the same
	// write goes through.
	if err := held.release(); err != nil {
		t.Fatalf("releasing: %v", err)
	}
	if _, err := b.Create(context.Background(), CreateOptions{
		Title: "Written from the linked worktree",
		Actor: testActor,
	}); err != nil {
		t.Errorf("the write still fails after the lock was released: %v", err)
	}
}

// TestLockPathFallsBackOutsideARepository covers the other branch of
// lockPath. A store with no repository around it has no worktrees to
// coordinate, so the lock lives inside the store.
//
// Worth pinning because the fallback is what runs when the criterion above
// does not apply, and a change that broke it would otherwise show up as a
// lock file appearing somewhere surprising.
func TestLockPathFallsBackOutsideARepository(t *testing.T) {
	s := newTestStore(t)
	want := filepath.Join(s.Path(), ".lock")
	if got := s.lockPath(); got != want {
		t.Errorf("lockPath() = %s, want %s", got, want)
	}

	// It is the file the lock actually takes, not just the path it reports.
	held, err := s.lock()
	if err != nil {
		t.Fatalf("locking: %v", err)
	}
	defer held.release()
	if _, err := os.Stat(want); err != nil {
		t.Errorf("no lock file at %s: %v", want, err)
	}
}
