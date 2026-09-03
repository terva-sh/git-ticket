package ticket

import (
	"context"
	"reflect"
	"testing"
)

// createWithPriority files a ticket at a given priority, optionally with a
// deadline. An empty priority takes the store default, which is normal.
func createWithPriority(t *testing.T, s *Store, title, priority, due string) *Ticket {
	t.Helper()
	opts := CreateOptions{
		Title:       title,
		Description: "Created by a test.",
		Actor:       testActor,
		Priority:    priority,
	}
	if due != "" {
		opts.DueOn = dueDate(due)
	}
	res, err := s.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	return res.Ticket
}

// TestSortByPriorityRanksUrgentFirst is the order plan 8 settles: priority,
// then the deadline, then the ID.
//
// The expectations are literal rather than derived from Priorities, because a
// test written in terms of the code moves with the code and cannot catch a
// change to it. Reordering the constants would leave a derived expectation
// passing.
func TestSortByPriorityRanksUrgentFirst(t *testing.T) {
	ts := []*Ticket{
		{ID: "TKT-06", Priority: "low"},
		{ID: "TKT-05", Priority: "normal"},
		{ID: "TKT-04", Priority: "high"},
		{ID: "TKT-03", Priority: "urgent", DueOn: dueDate("2026-12-01")},
		{ID: "TKT-02", Priority: "urgent", DueOn: dueDate("2026-01-05")},
		{ID: "TKT-01", Priority: "urgent"},
	}
	SortByPriority(ts)

	want := []string{
		"TKT-02", // urgent, and the earliest deadline of the three
		"TKT-03", // urgent, later deadline
		"TKT-01", // urgent, undated, which is never and so last in its band
		"TKT-04", // high
		"TKT-05", // normal
		"TKT-06", // low
	}
	if got := idsOfTickets(ts); !reflect.DeepEqual(got, want) {
		t.Errorf("SortByPriority = %v, want %v", got, want)
	}
}

// TestPriorityOutranksTheDeadline is the case that decided the order, spelled
// out on its own because it is the one a reader will come here to check.
//
// A low ticket due next week against an urgent with no date. Deadline-first
// puts the low one on top and can only be corrected by inventing a date for the
// urgent one. Priority-first is corrected by raising a priority, which is what
// the field means.
func TestPriorityOutranksTheDeadline(t *testing.T) {
	ts := []*Ticket{
		{ID: "TKT-low-but-dated", Priority: "low", DueOn: dueDate("2026-10-07")},
		{ID: "TKT-urgent-undated", Priority: "urgent"},
	}
	SortByPriority(ts)

	if got := idsOfTickets(ts)[0]; got != "TKT-urgent-undated" {
		t.Errorf("first = %s, want the urgent ticket. A deadline does not outrank a priority", got)
	}
}

// TestSortByPriorityDegradesToTheKeysBeneathIt is what made the key safe to
// apply without a flag, the same argument the deadline key was given.
//
// Every ticket carries normal until somebody says otherwise, so in a store
// where nobody has set a priority the key changes nothing.
func TestSortByPriorityDegradesToTheKeysBeneathIt(t *testing.T) {
	t.Run("falls through to the deadline", func(t *testing.T) {
		ts := []*Ticket{
			{ID: "TKT-03", Priority: "normal", DueOn: dueDate("2026-12-01")},
			{ID: "TKT-02", Priority: "normal"},
			{ID: "TKT-01", Priority: "normal", DueOn: dueDate("2026-01-05")},
		}
		SortByPriority(ts)

		want := []string{"TKT-01", "TKT-03", "TKT-02"}
		if got := idsOfTickets(ts); !reflect.DeepEqual(got, want) {
			t.Errorf("= %v, want the deadline order %v", got, want)
		}
	})

	t.Run("falls through to the ID", func(t *testing.T) {
		ts := []*Ticket{
			{ID: "TKT-03", Priority: "normal"},
			{ID: "TKT-01", Priority: "normal"},
			{ID: "TKT-02", Priority: "normal"},
		}
		SortByPriority(ts)

		want := []string{"TKT-01", "TKT-02", "TKT-03"}
		if got := idsOfTickets(ts); !reflect.DeepEqual(got, want) {
			t.Errorf("= %v, want the ID order %v", got, want)
		}
	})
}

// TestAnUnknownPriorityRanksBelowLow holds the rule for a value 5.1 does not
// define. check reports it as invalid; the query still has to put it somewhere,
// and below every value somebody set on purpose is the only safe place.
func TestAnUnknownPriorityRanksBelowLow(t *testing.T) {
	ts := []*Ticket{
		{ID: "TKT-02", Priority: "catastrophic"},
		{ID: "TKT-01", Priority: "low"},
	}
	SortByPriority(ts)

	want := []string{"TKT-01", "TKT-02"}
	if got := idsOfTickets(ts); !reflect.DeepEqual(got, want) {
		t.Errorf("= %v, want an unrecognized priority last %v", got, want)
	}
}

// TestReadyRanksByPriority runs the order through the store rather than the
// comparator, because Ready choosing the wrong sort function is exactly the
// regression a comparator test cannot see.
func TestReadyRanksByPriority(t *testing.T) {
	s := newTestStore(t)

	low := createWithPriority(t, s, "Low but due soon", "low", "2026-10-07")
	urgent := createWithPriority(t, s, "Urgent with no date", "urgent", "")
	normal := createWithPriority(t, s, "Ordinary work", "normal", "")
	for _, id := range []string{low.ID, urgent.ID, normal.ID} {
		mustApply(t, s, id, SetStatus{Status: StatusReady})
	}

	got, err := s.Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{urgent.ID, normal.ID, low.ID}
	if !reflect.DeepEqual(idsOfTickets(got), want) {
		t.Errorf("Ready = %v, want urgent first and the dated low ticket last %v",
			idsOfTickets(got), want)
	}
}
