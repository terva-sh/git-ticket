package tui_test

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
	"github.com/terva-sh/git-ticket/tui/tuitest"
)

// started returns a Frame on a FakeTerm with Start already run, and
// fails the test if the lifecycle refuses.
func started(t *testing.T, cols, rows int) (*tui.Frame, *tuitest.FakeTerm) {
	t.Helper()
	term := tuitest.NewFakeTerm(cols, rows)
	f := tui.NewFrame(term)
	if err := f.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return f, term
}

func TestStartEntersAndStopLeavesTheAltScreen(t *testing.T) {
	f, term := started(t, 40, 10)
	out := term.Output()
	if !strings.Contains(out, tui.SeqAltScreenOn) {
		t.Fatalf("Start did not enter the alternate screen: %q", out)
	}
	if !strings.Contains(out, tui.SeqHideCursor) {
		t.Fatalf("Start did not hide the cursor: %q", out)
	}
	if err := f.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	out = term.Output()
	if !strings.Contains(out, tui.SeqAltScreenOff) {
		t.Fatalf("Stop did not leave the alternate screen: %q", out)
	}
	if !strings.HasSuffix(out, tui.SeqShowCursor+tui.SeqAltScreenOff) {
		t.Fatalf("Stop should restore the cursor before leaving: %q", out)
	}
	// Stop twice is fine, and Stop without Start is fine.
	if err := f.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if err := tui.NewFrame(tuitest.NewFakeTerm(5, 5)).Stop(); err != nil {
		t.Fatalf("Stop without Start: %v", err)
	}
}

func TestDrawPaintsTheFrame(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	s := term.Screen()
	for i, want := range []string{"alpha", "beta", "gamma", "", ""} {
		if got := s.Row(i); got != want {
			t.Fatalf("row %d = %q, want %q", i, got, want)
		}
	}
}

func TestDrawBeforeStartWritesNothing(t *testing.T) {
	term := tuitest.NewFakeTerm(40, 5)
	f := tui.NewFrame(term)
	if err := f.Draw([]string{"quiet"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if out := term.Output(); out != "" {
		t.Fatalf("Draw before Start wrote %q", out)
	}
}

func TestDrawRewritesOnlyTheChangedRows(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"stable one", "changing", "stable two"}); err != nil {
		t.Fatalf("first Draw: %v", err)
	}
	before := len(term.Output())
	if err := f.Draw([]string{"stable one", "changed!", "stable two"}); err != nil {
		t.Fatalf("second Draw: %v", err)
	}
	delta := term.Output()[before:]
	if !strings.Contains(delta, "changed!") {
		t.Fatalf("changed row was not rewritten: %q", delta)
	}
	if strings.Contains(delta, "stable one") || strings.Contains(delta, "stable two") {
		t.Fatalf("unchanged rows were rewritten: %q", delta)
	}
	// An identical Draw writes no row at all.
	before = len(term.Output())
	if err := f.Draw([]string{"stable one", "changed!", "stable two"}); err != nil {
		t.Fatalf("third Draw: %v", err)
	}
	delta = term.Output()[before:]
	if strings.Contains(delta, "stable") || strings.Contains(delta, "changed") {
		t.Fatalf("no-change Draw rewrote content: %q", delta)
	}
}

func TestDrawClearsTheRowsAShorterFrameDropped(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"one", "two", "three"}); err != nil {
		t.Fatalf("first Draw: %v", err)
	}
	if err := f.Draw([]string{"one"}); err != nil {
		t.Fatalf("second Draw: %v", err)
	}
	s := term.Screen()
	if got := s.Row(0); got != "one" {
		t.Fatalf("row 0 = %q, want %q", got, "one")
	}
	for i := 1; i < 3; i++ {
		if got := s.Row(i); got != "" {
			t.Fatalf("dropped row %d still shows %q", i, got)
		}
	}
}

func TestDrawClipsARowToTheTerminalWidth(t *testing.T) {
	f, term := started(t, 10, 3)
	if err := f.Draw([]string{"0123456789ABCDEF"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if got := term.Screen().Row(0); got != "0123456789" {
		t.Fatalf("row 0 = %q, want the 10-cell clip", got)
	}
}

func TestDrawDropsTheRowsBelowTheTerminal(t *testing.T) {
	f, term := started(t, 20, 2)
	if err := f.Draw([]string{"top", "bottom", "overflow"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	s := term.Screen()
	if got := s.Row(0); got != "top" {
		t.Fatalf("row 0 = %q", got)
	}
	if got := s.Row(1); got != "bottom" {
		t.Fatalf("row 1 = %q", got)
	}
	if strings.Contains(term.Output(), "overflow") {
		t.Fatalf("a row below the terminal was written")
	}
}

func TestResizeForcesAFullRepaint(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"held row", "second row"}); err != nil {
		t.Fatalf("first Draw: %v", err)
	}
	term.Resize(30, 4)
	before := len(term.Output())
	if err := f.Draw([]string{"held row", "second row"}); err != nil {
		t.Fatalf("Draw after resize: %v", err)
	}
	delta := term.Output()[before:]
	if !strings.Contains(delta, "held row") || !strings.Contains(delta, "second row") {
		t.Fatalf("resize did not repaint every row: %q", delta)
	}
	if got := term.Screen().Row(0); got != "held row" {
		t.Fatalf("row 0 after resize = %q", got)
	}
}

func TestInvalidateForcesAFullRepaint(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"kept"}); err != nil {
		t.Fatalf("first Draw: %v", err)
	}
	f.Invalidate()
	before := len(term.Output())
	if err := f.Draw([]string{"kept"}); err != nil {
		t.Fatalf("Draw after Invalidate: %v", err)
	}
	if delta := term.Output()[before:]; !strings.Contains(delta, "kept") {
		t.Fatalf("Invalidate did not repaint the unchanged row: %q", delta)
	}
}

func TestSetCursorPlacesAndShowsTheCaret(t *testing.T) {
	f, term := started(t, 40, 5)
	f.SetCursor(2, 7)
	if err := f.Draw([]string{"first", "second"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	s := term.Screen()
	if !s.CursorVisible() {
		t.Fatalf("cursor hidden after SetCursor")
	}
	if x, y := s.Cursor(); x != 6 || y != 1 {
		t.Fatalf("cursor at (%d,%d), want (6,1)", x, y)
	}
	f.HideCursor()
	if err := f.Draw([]string{"first", "second"}); err != nil {
		t.Fatalf("second Draw: %v", err)
	}
	if s.CursorVisible() {
		t.Fatalf("cursor still visible after HideCursor")
	}
}

func TestDrawCutsARowAtAControlByte(t *testing.T) {
	f, term := started(t, 40, 5)
	if err := f.Draw([]string{"good\nsmuggled", "next"}); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	s := term.Screen()
	if got := s.Row(0); got != "good" {
		t.Fatalf("row 0 = %q, want %q", got, "good")
	}
	if got := s.Row(1); got != "next" {
		t.Fatalf("row 1 = %q, want %q", got, "next")
	}
}

func TestSizeAnswersTheLiveSizeBeforeAndAfterDraw(t *testing.T) {
	f, term := started(t, 33, 7)
	if c, r := f.Size(); c != 33 || r != 7 {
		t.Fatalf("Size before Draw = (%d,%d)", c, r)
	}
	if err := f.Draw(nil); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	term.Resize(20, 4)
	if err := f.Draw(nil); err != nil {
		t.Fatalf("Draw after resize: %v", err)
	}
	if c, r := f.Size(); c != 20 || r != 4 {
		t.Fatalf("Size after resize Draw = (%d,%d)", c, r)
	}
}
