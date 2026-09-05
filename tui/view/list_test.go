package view

import (
	"errors"
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

func mk(id, status, priority, title string) *ticket.Ticket {
	return &ticket.Ticket{ID: id, Status: status, Priority: priority, Title: title}
}

// Three IDs with distinct early bytes, so eight characters abbreviate
// them, plus a pair identical to the last character for the
// shortest-unique test.
var (
	tktA     = mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "ready", "high", "Fix the flux capacitor")
	tktB     = mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "in-progress", "normal", "Paint the shed")
	tktC     = mk("TKT-01CY5ZZKBKACTAV9WEVGEMMVS0", "blocked", "low", "Wait for paint")
	tktTwinA = mk("TKT-01TWINZZZZZZZZZZZZZZZZZZZA", "ready", "normal", "First twin")
	tktTwinB = mk("TKT-01TWINZZZZZZZZZZZZZZZZZZZB", "ready", "normal", "Second twin")
)

func fixed(ts ...*ticket.Ticket) Lister {
	return func() ([]*ticket.Ticket, error) { return ts, nil }
}

// newTestList is NewListView with no write surface, which is what
// every rendering and navigation test wants.
func newTestList(l Lister) *ListView { return NewListView(l, Actions{}) }

// plain renders the view and strips the styling, because these tests
// assert content and the styling is not the content.
func plain(v *ListView, cols, rows int) []string {
	out := v.Render(cols, rows)
	for i := range out {
		out[i] = tui.StripANSI(out[i])
	}
	return out
}

func TestListShowsOpenWork(t *testing.T) {
	v := newTestList(fixed(tktA, tktB, tktC))
	rows := plain(v, 100, 10)

	if !strings.Contains(rows[0], "ID") || !strings.Contains(rows[0], "STATUS") || !strings.Contains(rows[0], "TITLE") {
		t.Fatalf("header = %q", rows[0])
	}
	body := strings.Join(rows, "\n")
	for _, want := range []string{
		"TKT-01ARZ3ND", "ready", "high", "Fix the flux capacitor",
		"in-progress", "Paint the shed",
		"blocked", "Wait for paint",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered view lacks %q:\n%s", want, body)
		}
	}
	if foot := rows[len(rows)-1]; !strings.Contains(foot, "3 open tickets") {
		t.Fatalf("footer = %q", foot)
	}
}

func TestShortIDsAreShortestUnique(t *testing.T) {
	v := newTestList(fixed(tktTwinA, tktTwinB))
	body := strings.Join(plain(v, 120, 10), "\n")
	// The twins differ only in their last character, so nothing shorter
	// than the full body distinguishes them.
	if !strings.Contains(body, "TKT-01TWINZZZZZZZZZZZZZZZZZZZA") ||
		!strings.Contains(body, "TKT-01TWINZZZZZZZZZZZZZZZZZZZB") {
		t.Fatalf("twin IDs were abbreviated into ambiguity:\n%s", body)
	}

	v = newTestList(fixed(tktA, tktB))
	body = strings.Join(plain(v, 120, 10), "\n")
	if strings.Contains(body, tktA.ID) {
		t.Fatalf("distinct IDs were not abbreviated:\n%s", body)
	}
	if !strings.Contains(body, "TKT-01ARZ3ND") {
		t.Fatalf("expected the eight-character abbreviation:\n%s", body)
	}
}

func TestNavigationMovesTheSelection(t *testing.T) {
	v := newTestList(fixed(tktA, tktB, tktC))
	if got := v.SelectedID(); got != tktA.ID {
		t.Fatalf("initial selection = %s", got)
	}
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'})
	if got := v.SelectedID(); got != tktB.ID {
		t.Fatalf("after j selection = %s", got)
	}
	v.HandleKey(tui.Key{Kind: tui.KeyDown})
	v.HandleKey(tui.Key{Kind: tui.KeyDown}) // clamped at the end
	if got := v.SelectedID(); got != tktC.ID {
		t.Fatalf("after down down selection = %s", got)
	}
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'g'})
	if got := v.SelectedID(); got != tktA.ID {
		t.Fatalf("after g selection = %s", got)
	}
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'G'})
	if got := v.SelectedID(); got != tktC.ID {
		t.Fatalf("after G selection = %s", got)
	}

	rows := plain(v, 100, 10)
	marked := ""
	for _, r := range rows {
		if strings.HasPrefix(r, "▸") {
			marked = r
		}
	}
	if !strings.Contains(marked, "Wait for paint") {
		t.Fatalf("marker sits on %q, want the selected row", marked)
	}
}

func TestScrollingWindowsTheList(t *testing.T) {
	// Ten tickets in a pane with four body rows.
	var ts []*ticket.Ticket
	for i := 0; i < 10; i++ {
		id := "TKT-01" + strings.Repeat("A", 23) + string(rune('A'+i))
		ts = append(ts, mk(id, "ready", "normal", "Ticket number "+string(rune('A'+i))))
	}
	v := newTestList(fixed(ts...))
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'G'})
	rows := plain(v, 100, 6)
	body := strings.Join(rows, "\n")
	if !strings.Contains(body, "Ticket number J") {
		t.Fatalf("bottom of the list is not visible after G:\n%s", body)
	}
	if strings.Contains(body, "Ticket number A") {
		t.Fatalf("top of the list should have scrolled out:\n%s", body)
	}
}

func TestReloadKeepsTheSelectedTicket(t *testing.T) {
	// The reload reorders the slice and inserts a new ticket in front.
	first := []*ticket.Ticket{tktA, tktB, tktC}
	second := []*ticket.Ticket{tktC, tktB, tktA}
	call := 0
	v := newTestList(func() ([]*ticket.Ticket, error) {
		call++
		if call == 1 {
			return first, nil
		}
		return second, nil
	})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'}) // select B
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'r'})
	if got := v.SelectedID(); got != tktB.ID {
		t.Fatalf("selection after reload = %s, want %s", got, tktB.ID)
	}
}

func TestReloadErrorKeepsTheRowsAndNamesTheError(t *testing.T) {
	call := 0
	v := newTestList(func() ([]*ticket.Ticket, error) {
		call++
		if call == 1 {
			return []*ticket.Ticket{tktA}, nil
		}
		return nil, errors.New("store walked away")
	})
	v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'r'})
	rows := plain(v, 100, 8)
	body := strings.Join(rows, "\n")
	if !strings.Contains(body, "Fix the flux capacitor") {
		t.Fatalf("error dropped the previous rows:\n%s", body)
	}
	if !strings.Contains(rows[len(rows)-1], "store walked away") {
		t.Fatalf("footer does not name the error: %q", rows[len(rows)-1])
	}
}

func TestEmptyStoreSaysSo(t *testing.T) {
	v := newTestList(fixed())
	body := strings.Join(plain(v, 80, 8), "\n")
	if !strings.Contains(body, "No open work.") {
		t.Fatalf("empty view:\n%s", body)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, k := range []tui.Key{
		{Kind: tui.KeyRune, Rune: 'q'},
		{Kind: tui.KeyEsc},
		{Kind: tui.KeyCtrlC},
	} {
		v := newTestList(fixed(tktA))
		if !v.HandleKey(k) {
			t.Fatalf("key %+v did not quit", k)
		}
	}
	v := newTestList(fixed(tktA))
	if v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'}) {
		t.Fatalf("j should not quit")
	}
}
