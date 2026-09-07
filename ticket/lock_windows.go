//go:build windows

package ticket

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryFlock takes an exclusive lock without blocking, the same contract
// lock_unix.go fills with flock. Windows has no flock, and LockFileEx is the
// equivalent: it locks a byte range of an open handle, and the kernel drops
// that lock when the handle closes or the process dies. So there is no stale
// lock to break here either.
//
// The range is one byte at offset zero. The lock file carries no content, and
// Windows permits locking a range past the end of a file, so the range only has
// to be one every caller agrees on.
func tryFlock(f *os.File) (bool, error) {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		&overlapped,
	)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION):
		// Another holder has it. Report contention rather than failure, so
		// lock() polls until its deadline. This is the branch EWOULDBLOCK takes
		// on unix, and getting it wrong turns every contended write into an
		// immediate lock_timeout.
		return false, nil
	default:
		// ERROR_IO_PENDING is deliberately not contention. It arises only on a
		// handle opened with FILE_FLAG_OVERLAPPED, which os.OpenFile does not
		// use, so reaching it means an assumption here is wrong. lock() reports
		// the message, which names the condition, rather than spinning to a
		// timeout that would blame a holder that does not exist.
		return false, err
	}
}

func unflock(f *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &overlapped)
}
