package view

import (
	"sync"

	"github.com/terva-sh/git-ticket/tui"
)

// Run drives the list view on term until the user quits or the input
// ends. It owns the whole terminal lifecycle: Frame.Start before the
// first paint, Frame.Stop on every way out.
//
// The loop is one goroutine reading keys and drawing, which is the
// topology the Frame's no-concurrency contract wants. The one outside
// caller is the resize handler, which arrives on a signal goroutine,
// so the draw path is behind a mutex: a resize repaints immediately
// rather than waiting for the next keypress, and the two writers
// cannot interleave inside a frame.
func Run(term tui.Terminal, relist Lister) error {
	v := NewListView(relist)
	f := tui.NewFrame(term)
	if err := f.Start(); err != nil {
		return err
	}
	defer f.Stop()

	var mu sync.Mutex
	draw := func() {
		mu.Lock()
		defer mu.Unlock()
		// The live size, not Frame.Size: after a resize Frame.Size still
		// answers with the previous Draw's geometry, and a layout built
		// for the old size would be clipped to the new one.
		cols, rows := term.Size()
		_ = f.Draw(v.Render(cols, rows))
	}

	// Frame.Start registered its own invalidation first, so by the time
	// this runs the repaint is full, not a stale diff.
	term.OnResize(draw)
	draw()

	r := tui.NewReaderWithPeek(term.ReadByte, term.PeekByteTimeout)
	for {
		k, err := r.Read()
		if err != nil {
			// The input ending (EOF, a closed pty) is a quit, not a
			// failure: the terminal is gone, there is nobody to report to.
			return nil
		}
		mu.Lock()
		quit := v.HandleKey(k)
		mu.Unlock()
		if quit {
			return nil
		}
		draw()
	}
}
