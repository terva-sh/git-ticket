package view

import (
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// sortFixtures is a set the orders disagree about, so a wrong order
// cannot pass by luck. IDs run a, b, c chronologically.
func sortFixtures() []*ticket.Ticket {
	a := mk("TKT-01ARZ3NDEKTSV4RRFFQ69G5FAV", "done", "low", "Oldest, done, low")
	a.UpdatedAt = ticket.Now(time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC))
	b := mk("TKT-01BX5ZZKBKACTAV9WEVGEMMVRZ", "in-progress", "normal", "Middle, moving, normal")
	b.UpdatedAt = ticket.Now(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	c := mk("TKT-01CY5ZZKBKACTAV9WEVGEMMVS0", "ready", "urgent", "Newest, ready, urgent")
	c.UpdatedAt = ticket.Now(time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC))
	return []*ticket.Ticket{a, b, c}
}

func pressO(v *ListView, times int) {
	for i := 0; i < times; i++ {
		v.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'o'})
	}
}

// TestSortCycleReordersTheRows walks o through the whole cycle and
// checks one distinguishing row order per stop, ending back at the
// chronological default.
func TestSortCycleReordersTheRows(t *testing.T) {
	v := newTestList(fixed(sortFixtures()...))

	if got := shownTitles(v); got != "Oldest, done, low|Middle, moving, normal|Newest, ready, urgent" {
		t.Fatalf("default order = %q", got)
	}

	pressO(v, 1) // due_on: nobody dated, so chronological, byte for byte
	if got := shownTitles(v); got != "Oldest, done, low|Middle, moving, normal|Newest, ready, urgent" {
		t.Fatalf("due_on with no dates = %q, want the ID order", got)
	}

	pressO(v, 1) // priority: urgent first, low last
	if got := shownTitles(v); got != "Newest, ready, urgent|Middle, moving, normal|Oldest, done, low" {
		t.Fatalf("priority order = %q", got)
	}

	pressO(v, 1) // updated_at: newest write first
	if got := shownTitles(v); got != "Oldest, done, low|Newest, ready, urgent|Middle, moving, normal" {
		t.Fatalf("updated_at order = %q", got)
	}

	pressO(v, 1) // status: working set first
	if got := shownTitles(v); got != "Middle, moving, normal|Newest, ready, urgent|Oldest, done, low" {
		t.Fatalf("status order = %q", got)
	}

	pressO(v, 1) // back to the default
	if got := shownTitles(v); got != "Oldest, done, low|Middle, moving, normal|Newest, ready, urgent" {
		t.Fatalf("after a full cycle = %q, want the ID order back", got)
	}
}

// TestSortHeaderNamesTheOrder: the header carries the active token,
// because an invisible sort mode reads as a broken list.
func TestSortHeaderNamesTheOrder(t *testing.T) {
	v := newTestList(fixed(sortFixtures()...))
	if head := plain(v, 100, 10)[0]; !strings.Contains(head, "sort: id") {
		t.Fatalf("header = %q, want the default order named", head)
	}
	pressO(v, 2)
	if head := plain(v, 100, 10)[0]; !strings.Contains(head, "sort: priority") {
		t.Fatalf("header = %q, want sort: priority", head)
	}
}

// TestSortSurvivesTheFilter: the two compose, and sorting must not
// reorder the unfiltered list behind the filter's back.
func TestSortSurvivesTheFilter(t *testing.T) {
	v := newTestList(fixed(sortFixtures()...))
	pressO(v, 2) // priority
	typeFilter(v, "status:ready status:done")
	if got := shownTitles(v); got != "Newest, ready, urgent|Oldest, done, low" {
		t.Fatalf("filtered and sorted = %q", got)
	}

	// Clearing the filter and returning to the default must restore
	// the pristine chronological order.
	v.HandleKey(tui.Key{Kind: tui.KeyEsc})
	pressO(v, 3) // priority -> updated_at -> status -> id
	if got := shownTitles(v); got != "Oldest, done, low|Middle, moving, normal|Newest, ready, urgent" {
		t.Fatalf("restored order = %q, want the ID order untouched", got)
	}
}

// TestSortKeepsTheSelection: the cursor follows the ticket through a
// re-sort, not the row number, same as Reload.
func TestSortKeepsTheSelection(t *testing.T) {
	v := newTestList(fixed(sortFixtures()...))
	v.HandleKey(tui.Key{Kind: tui.KeyDown}) // select the middle ticket
	selected := v.SelectedID()
	pressO(v, 4) // status: the middle ticket is in-progress and moves to row 0
	if v.SelectedID() != selected {
		t.Fatalf("selection moved from %s to %s across a re-sort", selected, v.SelectedID())
	}
}
