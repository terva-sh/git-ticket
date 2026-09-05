package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// TestDetailRendersStyledMarkdown holds the wiring, not the renderer:
// a section with a heading and a Go fence reaches the screen styled,
// and StripANSI of the same rows still reads as the prose the author
// wrote, which is plan 14's "reads correctly as written" carried into
// the styled view.
func TestDetailRendersStyledMarkdown(t *testing.T) {
	tk := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Styled")
	tk.Body = ticket.Body{
		Description: "### Rollback plan\n\nrun `just ci` first\n\n```go\nfunc main() {}\n```",
	}
	d := NewDetailView(tk)
	rows := d.Render(80, 30)
	raw := strings.Join(rows, "\n")
	plain := tui.StripANSI(raw)

	for _, want := range []string{"### Rollback plan", "run just ci first", "func main() {}"} {
		if !strings.Contains(plain, want) {
			t.Errorf("detail lost %q:\n%s", want, plain)
		}
	}
	if !strings.Contains(raw, "\x1b[1m\x1b[38;5;111m### Rollback plan") {
		t.Error("the heading row is not bold and accented")
	}
	// The fence went through chroma if any row carries a 256-color
	// sequence that is not the accent or muted chrome.
	if !strings.Contains(raw, "\x1b[38;5;") {
		t.Error("no 256-color styling reached the detail rows")
	}
}

// TestDetailStyledProseStillWrapsToTheWidth re-pins the wrap claim the
// plain renderer already made, because the styled path replaced it: a
// long paragraph folds inside the width, and no visible row overflows.
func TestDetailStyledProseStillWrapsToTheWidth(t *testing.T) {
	tk := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "normal", "Wraps")
	tk.Body = ticket.Body{
		Description: strings.Repeat("the drift compounds under load and ", 8),
	}
	d := NewDetailView(tk)
	rows := d.Render(60, 30)
	sawFold := 0
	for i, r := range rows {
		if w := tui.VisibleWidth(r); w > 60 {
			t.Errorf("row %d is %d columns wide, over the 60 the view was given: %q", i, w, tui.StripANSI(r))
		}
		if strings.Contains(r, "drift compounds") {
			sawFold++
		}
	}
	if sawFold < 2 {
		t.Errorf("the long paragraph occupies %d rows; it did not wrap", sawFold)
	}
}
