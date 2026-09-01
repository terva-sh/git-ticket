package ticket

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// One lock guards the whole store, per plan 7.2. Contention is rare, and
// per-ticket locking would complicate the multi-file operations in check and
// archive for no gain.

// lockPollInterval is how often acquisition retries while it waits. The lock is
// held for the length of one file read and one rename, so a short poll costs
// nothing and returns quickly when the holder leaves.
const lockPollInterval = 20 * time.Millisecond

// storeLock is a held lock. Release closes the file, which the kernel treats as
// releasing the flock.
type storeLock struct {
	file *os.File
	path string
}

// lockPath returns the file that guards this store.
//
// It lives under the common Git directory, so every worktree of the repository
// shares one lock and separate worktrees serialize correctly. A store outside a
// repository falls back to a lock inside the store itself, which does not
// coordinate worktrees because there are none to coordinate.
//
// The answer is cached, because finding it runs git and every mutation needs
// it.
func (s *Store) lockPath() string {
	s.lockOnce.Do(func() {
		if common := gitCommonDir(s.path); common != "" {
			s.lockFile = filepath.Join(common, "git-ticket", "store.lock")
			return
		}
		s.lockFile = filepath.Join(s.path, ".lock")
	})
	return s.lockFile
}

// gitCommonDir asks git for the common Git directory of the repository holding
// dir, which worktrees share. The answer may be relative to dir, so it is
// resolved before use.
func gitCommonDir(dir string) string {
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

// lock acquires the store lock, blocking up to the configured timeout. It
// returns lock_timeout when the wait runs out.
func (s *Store) lock() (*storeLock, error) {
	path := s.lockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, &Error{Code: CodeLockTimeout, Message: "cannot create the lock directory: " + err.Error(), Err: err}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, &Error{Code: CodeLockTimeout, Message: "cannot open the lock file: " + err.Error(), Err: err}
	}

	timeout := s.lockTimeout
	if timeout <= 0 {
		timeout = s.config.Lock.Timeout.Duration()
	}
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}
	// The deadline uses the real clock and not the store's. The store clock is
	// for what lands in a file, and a test freezes it; a frozen deadline would
	// make a contended acquisition spin forever.
	deadline := time.Now().Add(timeout)
	for {
		ok, err := tryFlock(f)
		if err != nil {
			f.Close()
			return nil, &Error{Code: CodeLockTimeout, Message: err.Error(), Err: err}
		}
		if ok {
			return &storeLock{file: f, path: path}, nil
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, &Error{
				Code:    CodeLockTimeout,
				Message: "another process holds the store lock",
				Details: map[string]string{"lock": path, "waited": timeout.String()},
			}
		}
		time.Sleep(lockPollInterval)
	}
}

// release drops the lock. The kernel also releases it if the process dies, so
// there is no stale-lock breaker to get wrong.
func (l *storeLock) release() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := unflock(l.file)
	if cerr := l.file.Close(); err == nil {
		err = cerr
	}
	l.file = nil
	return err
}
