//go:build unix

package ticket

import (
	"errors"
	"os"
	"syscall"
)

// tryFlock takes an exclusive lock without blocking. It reports whether the
// lock was taken. flock is used rather than a lock file the tool creates and
// deletes, because the kernel releases it when the holder dies and there is no
// stale lock to break.
func tryFlock(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK):
		return false, nil
	default:
		return false, err
	}
}

func unflock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
