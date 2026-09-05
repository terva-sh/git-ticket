package ticket

import (
	"testing"
	"time"
)

// These tests hold the two orders TKT-01M1S0WN added to plan 8. The CLI
// and the TUI both delegate here, so this is the one place the direction
// of each order is pinned.

func stamped(id string, at time.Time) *Ticket {
	return &Ticket{ID: id, UpdatedAt: Now(at)}
}

func TestSortByUpdatedIsNewestFirst(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }
	ts := []*Ticket{
		stamped("TKT-01", day(2)),
		stamped("TKT-02", day(5)),
		{ID: "TKT-03"},            // never written: the zero time lands last
		stamped("TKT-04", day(5)), // ties break by ID, so TKT-02 first
	}
	SortByUpdated(ts)
	want := []string{"TKT-02", "TKT-04", "TKT-01", "TKT-03"}
	for i, id := range want {
		if ts[i].ID != id {
			t.Fatalf("position %d = %s, want %s", i, ts[i].ID, id)
		}
	}
}

func TestSortByStatusIsWorkingSetFirst(t *testing.T) {
	ts := []*Ticket{
		{ID: "TKT-01", Status: StatusArchived},
		{ID: "TKT-02", Status: StatusDraft},
		{ID: "TKT-03", Status: StatusInProgress},
		{ID: "TKT-04", Status: StatusDone},
		{ID: "TKT-05", Status: StatusReady},
		{ID: "TKT-06", Status: StatusBlocked},
		{ID: "TKT-07", Status: StatusReview},
		{ID: "TKT-08", Status: "nonsense"},       // undefined ranks last, after archived
		{ID: "TKT-09", Status: StatusInProgress}, // equal statuses stay chronological
	}
	SortByStatus(ts)
	want := []string{
		"TKT-03", "TKT-09", // in-progress
		"TKT-07", // review
		"TKT-06", // blocked
		"TKT-05", // ready
		"TKT-02", // draft
		"TKT-04", // done
		"TKT-01", // archived
		"TKT-08", // undefined
	}
	for i, id := range want {
		if ts[i].ID != id {
			t.Fatalf("position %d = %s, want %s", i, ts[i].ID, id)
		}
	}
}
