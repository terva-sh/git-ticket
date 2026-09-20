// Package filelock is an exclusive advisory lock on a file, held for the
// length of one read and one rename, with a bounded wait. The ticket store
// takes one per store, per plan 7.2, and the layout package takes one per
// canvas directory, per 12.10; both need the same primitive and neither can
// import the other, so it lives here.
//
// The lock is the kernel's, flock on unix and LockFileEx on Windows, so it is
// released when the holder dies and there is no stale lock to break.
package filelock

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PollInterval is how often Acquire retries while it waits. The lock is held
// briefly, so a short poll costs nothing and returns quickly when the holder
// leaves.
const PollInterval = 20 * time.Millisecond

// ErrTimeout is what Acquire returns when the wait runs out. A caller that
// publishes its own code wraps it.
var ErrTimeout = errors.New("another process holds the lock")

// Lock is a held lock. Release closes the file, which releases the lock.
type Lock struct {
	file *os.File
	Path string
}

// Acquire takes the lock at path, creating the file and its directory if they
// are missing, and blocks up to timeout. The deadline uses the real clock:
// a frozen test clock would make a contended acquisition spin forever.
func Acquire(path string, timeout time.Duration) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("cannot create the lock directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot open the lock file: %w", err)
	}
	deadline := time.Now().Add(timeout)
	for {
		ok, err := TryLock(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		if ok {
			return &Lock{file: f, Path: path}, nil
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, fmt.Errorf("%w: %s, waited %s", ErrTimeout, path, timeout)
		}
		time.Sleep(PollInterval)
	}
}

// Release drops the lock. Releasing twice, or releasing nil, is harmless.
func (l *Lock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := Unlock(l.file)
	if cerr := l.file.Close(); err == nil {
		err = cerr
	}
	l.file = nil
	return err
}

// GitCommonDir asks git for the common Git directory of the repository holding
// dir, which every worktree shares, so a lock placed under it serialises the
// worktrees too. It is empty outside a repository or without git.
func GitCommonDir(dir string) string {
	out, err := runGit(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(out)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}
	return path
}

// runGit is this package's one git helper, per plan 7.4, which names the
// commands it may run and holds every package to one helper.
func runGit(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
