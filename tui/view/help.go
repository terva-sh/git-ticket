package view

import "github.com/terva-sh/git-ticket/tui"

// HelpView is the ? page: every key the list state accepts, and the
// filter token syntax, which no other screen documents. It exists
// because the footer fits 80 columns and so cannot name everything.
// A control the UI never prints might as well not exist, so the rule
// is: every binding appears here or in the footer, and
// TestEveryListKeyIsHintedSomewhere fails the suite when one does not.
type HelpView struct{}

// HandleKey closes the page on ?, q, Esc, or Enter, and quits the
// application on Ctrl+C, matching the detail view's escape ladder.
func (h *HelpView) HandleKey(k tui.Key) (back, quit bool) {
	switch {
	case k.Kind == tui.KeyCtrlC:
		return false, true
	case k.Kind == tui.KeyEsc, k.Kind == tui.KeyEnter,
		k.Kind == tui.KeyRune && (k.Rune == 'q' || k.Rune == '?'):
		return true, false
	}
	return false, false
}

// helpLines is the page body. Static, because the bindings are static:
// the Actions a host leaves unwired still answer on the key, saying so
// in the footer, which is itself a form of discoverability.
var helpLines = []string{
	"  \x1b[1mKeys\x1b[22m",
	"",
	"  ↑/↓ j/k      move the selection",
	"  g/G          jump to the first or last row",
	"  PgUp/PgDn    move a page at a time",
	"  Enter, l     open the selected ticket",
	"  n            create a new ticket",
	"  e            edit the selected ticket's title and description",
	"  s            change status, permitted transitions only",
	"  c            claim the selected ticket",
	"  u            release your claim",
	"  /            filter the list",
	"  r            reload from the store",
	"  ?            this page",
	"  Esc          clear the filter, then quit",
	"  q, Ctrl+C    quit",
	"",
	"  \x1b[1mFilter\x1b[22m",
	"",
	"  The list narrows as you type. A token names a field and a bare",
	"  word matches the title:",
	"",
	"    status:ready  label:mcp  assignee:human:sothr  editor",
	"",
	"  Repeating a field offers alternatives; different fields must all",
	"  match. Enter keeps the filter, Esc drops it.",
	"",
	"  The detail view, the status picker, and the form print their own",
	"  keys in their footers.",
}

// Render lays the page out. It pads to rows so the Frame overwrites
// every line of whatever was underneath.
func (h *HelpView) Render(cols, rows int) []string {
	if rows < 4 {
		rows = 4
	}
	out := make([]string, 0, rows)
	for _, l := range helpLines {
		if len(out) == rows-1 {
			break
		}
		out = append(out, l)
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  Esc/q/? close · Ctrl+C quit"))
	return out
}
