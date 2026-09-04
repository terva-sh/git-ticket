// Package tui is the terminal foundation for the git-ticket TUI: a tty
// abstraction, the escape-sequence vocabulary, ANSI-aware width helpers,
// and an alternate-screen frame renderer.
//
// The foundation is lifted from terva's packages/tui (MIT):
//
//	Copyright (c) 2026 Drew Short (Terva, a hard fork of zot)
//	Copyright (c) 2026 Patric Eckhart
//
// The lift is recorded on TKT-01M1QBSD. terva's renderer paints a chat
// flow into scrollback; the Frame renderer here is new, because a
// list-detail-form application wants the alternate screen instead.
package tui

import (
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// Terminal abstracts the real terminal for tests.
type Terminal interface {
	io.Writer
	// Size returns (cols, rows).
	Size() (int, int)
	// OnResize registers a callback invoked on SIGWINCH (best effort).
	OnResize(func())
	// EnterRaw puts the tty into raw mode. Returns a restore func.
	EnterRaw() (restore func() error, err error)
	// ReadByte reads one byte of input. Blocks.
	ReadByte() (byte, error)
	// PeekByteTimeout reads one byte of input but returns (0, false, nil)
	// if no byte arrives within the timeout. Used to disambiguate bare
	// Esc from the start of an escape sequence.
	PeekByteTimeout(time.Duration) (byte, bool, error)
	// SetNonblock sets stdin to non-blocking mode (used by paste handling).
	// May be a no-op on some platforms.
	SetNonblock(bool) error
}

// ProcTerm is a Terminal bound to the current process's tty.
type ProcTerm struct {
	out       *os.File
	in        *os.File
	resizeCBs []func()
}

// NewProcTerm returns a Terminal bound to stdin/stdout.
func NewProcTerm() *ProcTerm {
	return &ProcTerm{out: os.Stdout, in: os.Stdin}
}

func (p *ProcTerm) Write(b []byte) (int, error) { return p.out.Write(b) }

func (p *ProcTerm) Size() (int, int) {
	w, h, err := term.GetSize(int(p.out.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

func (p *ProcTerm) OnResize(fn func()) {
	p.resizeCBs = append(p.resizeCBs, fn)
	p.installResizeHandler()
}

func (p *ProcTerm) EnterRaw() (func() error, error) {
	fd := int(p.in.Fd())
	if !term.IsTerminal(fd) {
		return func() error { return nil }, nil
	}
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(fd, st) }, nil
}

func (p *ProcTerm) ReadByte() (byte, error) {
	var b [1]byte
	_, err := p.in.Read(b[:])
	return b[0], err
}

// PeekByteTimeout uses a platform-specific non-blocking read to decide
// whether another byte is available within d. If not, returns (0, false, nil).
func (p *ProcTerm) PeekByteTimeout(d time.Duration) (byte, bool, error) {
	return peekStdin(p.in, d)
}

// ---- terminal control sequences ----

const (
	SeqHideCursor  = "\x1b[?25l"
	SeqShowCursor  = "\x1b[?25h"
	SeqClearScreen = "\x1b[2J\x1b[H"
	// SeqClearScreenNoHome is SeqClearScreen without the trailing
	// cursor-home (\x1b[H). Use it whenever a MoveTo(...) follows
	// immediately. VS Code's integrated terminal interprets a bare
	// CUP-no-args during a clear as "snap the viewport scrollbar to
	// row 0", which makes the user's scroll position jump on every
	// repaint. The explicit MoveTo we emit afterwards positions the
	// cursor identically without triggering that snap.
	SeqClearScreenNoHome = "\x1b[2J"
	SeqCursorHome        = "\x1b[H"
	SeqClearToEnd        = "\x1b[0J"
	SeqClearLine         = "\x1b[2K"
	SeqBracketedPasteOn  = "\x1b[?2004h"
	SeqBracketedPasteOff = "\x1b[?2004l"
	SeqAltScreenOn       = "\x1b[?1049h"
	SeqAltScreenOff      = "\x1b[?1049l"
	SeqSynchronizedOn    = "\x1b[?2026h"
	SeqSynchronizedOff   = "\x1b[?2026l"
)

// MoveTo moves the cursor to 1-indexed (row, col).
func MoveTo(row, col int) string {
	return "\x1b[" + itoa(row) + ";" + itoa(col) + "H"
}

// small itoa to avoid pulling strconv into the hot path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [4]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
