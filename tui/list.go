// Adapted from the picker pattern in terva's modes/dialogs (MIT), see
// NOTICE at the repository root:
//   Copyright (c) 2026 Drew Short (Terva, a hard fork of zot)
//   Copyright (c) 2026 Patric Eckhart

package tui

// List is a cursor over n rows plus the viewport that keeps the cursor
// visible. It is the shape shared by every list a TUI grows: the picker
// this adapts from terva's modes/dialogs carried an items slice with
// hand-written bounds checks per dialog, and the cursor-window math
// lived in a separate helper each dialog applied or forgot. Here the
// cursor and the window are one type, so they cannot disagree.
//
// List holds no rows. The caller owns the data and renders it; List
// answers which index is selected and which slice [start, end) is
// visible. That keeps it generic over what a row is, which is the
// property that lets a ticket list, a status picker, and a label
// picker share it.
type List struct {
	vp     Viewport
	cursor int
	total  int
}

// SetTotal tells the list how many rows exist and clamps the cursor
// into them. Call it whenever the data changes, including to zero.
func (l *List) SetTotal(n int) {
	if n < 0 {
		n = 0
	}
	l.total = n
	if l.cursor >= n {
		l.cursor = n - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
}

// Total is the row count last given to SetTotal.
func (l *List) Total() int { return l.total }

// Cursor is the selected index, or 0 when the list is empty; use
// Total to tell those apart.
func (l *List) Cursor() int { return l.cursor }

// SetCursor moves the selection directly, clamped.
func (l *List) SetCursor(i int) {
	l.cursor = i
	l.clampCursor()
}

// Window fits the viewport to rows visible rows, reveals the cursor
// with one row of padding, and returns the visible [start, end).
func (l *List) Window(rows int) (start, end int) {
	l.vp.Fit(l.total, rows)
	l.vp.RevealPadded(l.cursor, 1)
	return l.vp.Window()
}

// HandleKey moves the cursor for the standard list keys and reports
// whether it consumed the key. Beyond the arrow keys it takes the vi
// pairs j/k and g/G, the wheel, and paging, because a list you cannot
// drive without leaving the home row is a list an agent-adjacent
// human will not drive at all.
func (l *List) HandleKey(k Key) bool {
	if l.total == 0 {
		return false
	}
	switch {
	case k.Kind == KeyUp, k.Kind == KeyMouseWheelUp,
		k.Kind == KeyRune && k.Rune == 'k':
		l.cursor--
	case k.Kind == KeyDown, k.Kind == KeyMouseWheelDown,
		k.Kind == KeyRune && k.Rune == 'j':
		l.cursor++
	case k.Kind == KeyPageUp:
		l.cursor -= l.page()
	case k.Kind == KeyPageDown:
		l.cursor += l.page()
	case k.Kind == KeyHome, k.Kind == KeyRune && k.Rune == 'g':
		l.cursor = 0
	case k.Kind == KeyEnd, k.Kind == KeyRune && k.Rune == 'G':
		l.cursor = l.total - 1
	default:
		return false
	}
	l.clampCursor()
	return true
}

func (l *List) page() int {
	if p := l.vp.page(); p > 1 {
		return p
	}
	return 1
}

func (l *List) clampCursor() {
	if l.cursor >= l.total {
		l.cursor = l.total - 1
	}
	if l.cursor < 0 {
		l.cursor = 0
	}
}
