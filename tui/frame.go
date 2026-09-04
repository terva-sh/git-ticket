package tui

import "strings"

// Frame paints a full-screen application on the terminal's alternate
// screen: the caller assembles the whole frame as []string of already
// styled rows, and Draw repaints only the rows that changed since the
// previous Draw.
//
// This is the piece that is deliberately new rather than lifted.
// terva's renderer emits a chat flow into the terminal's own scrollback
// and diff-redraws a live bottom band, which is the right shape for a
// conversation and the wrong one for a list-detail-form application. A
// ticket browser owns its whole screen, wants the host scrollback
// untouched behind it, and restores the user's terminal exactly as it
// was on exit. That is what the alternate screen is for.
//
// The contract with the caller:
//   - one string per terminal row, top to bottom, styled and ready;
//   - rows longer than the terminal are clipped with TruncateToWidth,
//     never wrapped, because wrapping is a layout decision that belongs
//     to the widget that built the row;
//   - rows beyond the terminal height are dropped;
//   - a row must not contain \n or \r. Draw cuts such a row at the
//     first control byte rather than let it smear the grid.
//
// Frame is not safe for concurrent use. One goroutine draws.
type Frame struct {
	term Terminal

	cols, rows int
	prev       []string // rows as painted, post-truncation
	valid      bool     // prev describes what is on screen

	started bool
	restore func() error

	// cursor is 1-indexed; row 0 means hidden, which is the default in
	// a full-screen application where most frames have no caret.
	cursorRow, cursorCol int
}

// NewFrame returns a Frame over t. Nothing is written until Start.
func NewFrame(t Terminal) *Frame {
	return &Frame{term: t}
}

// Start enters raw mode and the alternate screen, hides the cursor,
// and registers a resize handler that invalidates the frame. Stop
// undoes all of it. A second Start before Stop is a no-op.
func (f *Frame) Start() error {
	if f.started {
		return nil
	}
	restore, err := f.term.EnterRaw()
	if err != nil {
		return err
	}
	f.restore = restore
	f.started = true
	f.term.OnResize(func() { f.valid = false })
	// The clear matters even though the alternate screen starts empty:
	// a suspended-and-resumed process re-enters an alternate screen
	// still holding its old content.
	if _, err := f.term.Write([]byte(SeqAltScreenOn + SeqHideCursor + SeqClearScreenNoHome + SeqCursorHome)); err != nil {
		f.stopState()
		return err
	}
	return nil
}

// Stop leaves the alternate screen, shows the cursor, and restores the
// tty modes. It is safe to call when Start never ran, so a caller can
// defer it unconditionally.
func (f *Frame) Stop() error {
	if !f.started {
		return nil
	}
	_, werr := f.term.Write([]byte(SeqShowCursor + SeqAltScreenOff))
	rerr := f.stopState()
	if werr != nil {
		return werr
	}
	return rerr
}

func (f *Frame) stopState() error {
	f.started = false
	f.valid = false
	f.prev = nil
	var err error
	if f.restore != nil {
		err = f.restore()
		f.restore = nil
	}
	return err
}

// Size returns the terminal size as of the last Draw, or the live size
// before any Draw. Widgets lay out against this.
func (f *Frame) Size() (cols, rows int) {
	if f.cols > 0 && f.rows > 0 {
		return f.cols, f.rows
	}
	return f.term.Size()
}

// SetCursor places the terminal cursor at 1-indexed (row, col) after
// the next Draw and shows it. It is how an editing widget gets a caret.
func (f *Frame) SetCursor(row, col int) {
	if row < 1 || col < 1 {
		f.HideCursor()
		return
	}
	f.cursorRow, f.cursorCol = row, col
}

// HideCursor hides the cursor again after the next Draw.
func (f *Frame) HideCursor() {
	f.cursorRow, f.cursorCol = 0, 0
}

// Invalidate forces the next Draw to repaint every row. The resize
// handler calls it; a host whose output was disturbed by something
// outside the Frame can too.
func (f *Frame) Invalidate() {
	f.valid = false
}

// Draw paints lines as the full frame. Unchanged rows cost nothing: the
// diff against the previous frame decides what is written, and a Draw
// with no changes writes no row at all. A size change or an Invalidate
// repaints everything.
func (f *Frame) Draw(lines []string) error {
	if !f.started {
		return nil
	}
	cols, rows := f.term.Size()
	if cols != f.cols || rows != f.rows {
		f.cols, f.rows = cols, rows
		f.valid = false
	}

	// Clip to the grid: one string per row, each within the width.
	frame := make([]string, 0, min(len(lines), rows))
	for i := 0; i < len(lines) && i < rows; i++ {
		frame = append(frame, TruncateToWidth(sanitizeRow(lines[i]), cols))
	}

	var b strings.Builder
	b.WriteString(SeqSynchronizedOn)
	if !f.valid {
		b.WriteString(SeqClearScreenNoHome + SeqCursorHome)
		for i, row := range frame {
			b.WriteString(MoveTo(i+1, 1))
			b.WriteString(row)
		}
	} else {
		for i, row := range frame {
			if i < len(f.prev) && f.prev[i] == row {
				continue
			}
			b.WriteString(MoveTo(i+1, 1))
			b.WriteString(SeqClearLine)
			b.WriteString(row)
		}
		// Rows the previous frame painted and this one does not.
		for i := len(frame); i < len(f.prev); i++ {
			b.WriteString(MoveTo(i+1, 1))
			b.WriteString(SeqClearLine)
		}
	}
	if f.cursorRow > 0 {
		b.WriteString(MoveTo(f.cursorRow, f.cursorCol))
		b.WriteString(SeqShowCursor)
	} else {
		b.WriteString(SeqHideCursor)
		// Park the cursor at the origin so a terminal that ignores the
		// hide (or a screen reader following the caret) has a stable
		// position rather than the last row the diff happened to touch.
		b.WriteString(SeqCursorHome)
	}
	b.WriteString(SeqSynchronizedOff)

	if _, err := f.term.Write([]byte(b.String())); err != nil {
		// The screen state is unknown after a failed write.
		f.valid = false
		return err
	}
	f.prev = frame
	f.valid = true
	return nil
}

// sanitizeRow cuts a row at the first byte that would move the cursor
// off the row. A stray \n in a row would otherwise shift every row
// after it and corrupt the diff's idea of what is on screen.
func sanitizeRow(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
