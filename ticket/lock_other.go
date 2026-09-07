//go:build !unix && !windows

package ticket

import (
	"errors"
	"os"
)

// The store lock needs flock, which this platform does not have. Rather than
// pretend to lock, every acquisition fails: a silent no-op would let two
// writers into the store and lose one of their writes.
//
// Windows is no longer one of these platforms. It takes the same lock through
// LockFileEx, in lock_windows.go.

func tryFlock(f *os.File) (bool, error) {
	return false, errors.New("the store lock needs flock, which this platform does not provide")
}

func unflock(f *os.File) error { return nil }
