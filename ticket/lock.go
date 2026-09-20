package ticket

import (
	"errors"
	"path/filepath"

	"github.com/terva-sh/git-ticket/internal/filelock"
)

// One lock guards the whole store, per plan 7.2. Contention is rare, and
// per-ticket locking would complicate the multi-file operations in check and
// archive for no gain.

// storeLock is a held lock, a filelock.Lock under the store's own name so
// that release reads the same everywhere it is called.
type storeLock struct {
	*filelock.Lock
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

// gitCommonDir is filelock.GitCommonDir, kept under this name because the
// store's callers read it that way.
func gitCommonDir(dir string) string { return filelock.GitCommonDir(dir) }

// lock acquires the store lock, blocking up to the configured timeout. It
// returns lock_timeout when the wait runs out.
func (s *Store) lock() (*storeLock, error) {
	path := s.lockPath()
	timeout := s.lockTimeout
	if timeout <= 0 {
		timeout = s.config.Lock.Timeout.Duration()
	}
	if timeout <= 0 {
		timeout = DefaultLockTimeout
	}
	l, err := filelock.Acquire(path, timeout)
	if errors.Is(err, filelock.ErrTimeout) {
		return nil, &Error{
			Code:    CodeLockTimeout,
			Message: "another process holds the store lock",
			Details: map[string]string{"lock": path, "waited": timeout.String()},
		}
	}
	if err != nil {
		return nil, &Error{Code: CodeLockTimeout, Message: err.Error(), Err: err}
	}
	return &storeLock{Lock: l}, nil
}

// release drops the lock. The kernel also releases it if the process dies, so
// there is no stale-lock breaker to get wrong.
func (l *storeLock) release() error {
	if l == nil {
		return nil
	}
	return l.Lock.Release()
}
