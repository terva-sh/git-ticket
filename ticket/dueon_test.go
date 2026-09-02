package ticket

import (
	"context"
	"reflect"
	"testing"
)

// dueDate is the address of a date literal, for building tickets and options.
func dueDate(s string) *string { return &s }

// createDue files a ticket carrying a deadline.
func createDue(t *testing.T, s *Store, title, due string) *Ticket {
	t.Helper()
	opts := CreateOptions{Title: title, Description: "Created by a test.", Actor: testActor}
	if due != "" {
		opts.DueOn = dueDate(due)
	}
	res, err := s.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("create %q: %v", title, err)
	}
	return res.Ticket
}

// getTicket reads a ticket back from disk, so an assertion sees what a write
// actually stored rather than the struct it was handed.
func getTicket(t *testing.T, s *Store, id string) *Ticket {
	t.Helper()
	got, err := s.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	return got
}

// idsOfTickets is the order a query answered in, which is the thing under test
// in most of this file.
func idsOfTickets(ts []*Ticket) []string {
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.ID)
	}
	return out
}

// TestValidDueOnDemandsTheExactShape holds the rule that decides what reaches
// the store. An RFC3339 instant is the realistic mistake, because every other
// time value in plan 5.1 is one, and 5.1 refuses it rather than truncating it
// to its date.
func TestValidDueOnDemandsTheExactShape(t *testing.T) {
	cases := []struct {
		value string
		want  bool
		why   string
	}{
		{"2026-10-14", true, "the shape 5.1 specifies"},
		{"2026-01-04", true, "leading zeros are the shape"},
		{"2026-10-14T00:00:00Z", false, "an instant is refused, never truncated"},
		{"2026-1-4", false, "one day must have one spelling"},
		{"2026-02-30", false, "a day that does not exist"},
		{"14-10-2026", false, "the other order"},
		{"", false, "absent is nil, not an empty string"},
		{"tomorrow", false, "not a date at all"},
	}
	for _, c := range cases {
		if got := ValidDueOn(c.value); got != c.want {
			t.Errorf("ValidDueOn(%q) = %v, want %v: %s", c.value, got, c.want, c.why)
		}
	}
}

// TestSortByDueOnPutsNeverLast pins the order both commands share: earliest
// first, undated last, ID breaking a tie.
func TestSortByDueOnPutsNeverLast(t *testing.T) {
	ts := []*Ticket{
		{ID: "TKT-05", DueOn: nil},
		{ID: "TKT-04", DueOn: dueDate("2026-12-01")},
		{ID: "TKT-03", DueOn: dueDate("2026-01-05")},
		{ID: "TKT-01", DueOn: nil},
		{ID: "TKT-02", DueOn: dueDate("2026-01-05")},
	}
	SortByDueOn(ts)

	want := []string{
		"TKT-02", // 2026-01-05, and the lower ID of the two that share it
		"TKT-03", // 2026-01-05
		"TKT-04", // 2026-12-01
		"TKT-01", // undated, lower ID first
		"TKT-05", // undated
	}
	if got := idsOfTickets(ts); !reflect.DeepEqual(got, want) {
		t.Errorf("SortByDueOn = %v, want %v", got, want)
	}
}

// TestReadyOrdersByTheDeadline is the settled default: ready ranks, and it
// needs no flag to do it.
func TestReadyOrdersByTheDeadline(t *testing.T) {
	s := newTestStore(t)

	late := createDue(t, s, "Due in December", "2026-12-01")
	soon := createDue(t, s, "Due in January", "2026-01-05")
	none := createDue(t, s, "No deadline at all", "")
	for _, id := range []string{late.ID, soon.ID, none.ID} {
		mustApply(t, s, id, SetStatus{Status: StatusReady})
	}

	got, err := s.Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{soon.ID, late.ID, none.ID}
	if !reflect.DeepEqual(idsOfTickets(got), want) {
		t.Errorf("Ready = %v, want soonest first and undated last %v", idsOfTickets(got), want)
	}
}

// TestReadyOrderIsUnchangedWithoutDeadlines is the guarantee that made the
// default safe to change. In a store where nobody has set a date, the date key
// does nothing and the order is the ID order the store had before the field
// existed.
//
// If this ever fails, the settlement in plan 15 is wrong and not just the code:
// the argument for sorting without a flag was that no existing caller is
// reordered.
func TestReadyOrderIsUnchangedWithoutDeadlines(t *testing.T) {
	s := newTestStore(t)

	var ids []string
	for _, title := range []string{"First", "Second", "Third", "Fourth"} {
		one := createDue(t, s, title, "")
		mustApply(t, s, one.ID, SetStatus{Status: StatusReady})
		ids = append(ids, one.ID)
	}

	got, err := s.Ready(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by ID, which is what the store answered before due_on existed.
	// Compared as a sorted set rather than in creation order, because two
	// tickets filed in one millisecond carry no guarantee of which ID is lower.
	want := sorted(append([]string(nil), ids...))
	if !reflect.DeepEqual(idsOfTickets(got), want) {
		t.Errorf("Ready = %v, want the ID order %v", idsOfTickets(got), want)
	}
}

// TestDueByIsInclusiveAndSkipsUndated holds both halves of the bound. The
// inclusive end is what "due by the end of the month" means, and an undated
// ticket is not due by any date.
func TestDueByIsInclusiveAndSkipsUndated(t *testing.T) {
	s := newTestStore(t)

	on := createDue(t, s, "Due on the bound", "2026-06-30")
	before := createDue(t, s, "Due before it", "2026-06-01")
	after := createDue(t, s, "Due after it", "2026-07-01")
	createDue(t, s, "Never due", "")

	got, err := s.List(context.Background(), Filter{DueBy: "2026-06-30"})
	if err != nil {
		t.Fatal(err)
	}
	want := sorted([]string{on.ID, before.ID})
	if !reflect.DeepEqual(sorted(idsOfTickets(got)), want) {
		t.Errorf("DueBy = %v, want the bound itself and everything before it %v",
			idsOfTickets(got), want)
	}
	for _, g := range got {
		if g.ID == after.ID {
			t.Error("a ticket due after the bound matched it")
		}
	}
}

// TestSetDueOnRefusesAnInstant keeps the mutation as strict as the CLI, so a
// library caller cannot write what the command line refuses.
func TestSetDueOnRefusesAnInstant(t *testing.T) {
	s := newTestStore(t)
	one := mustCreate(t, s, "Rotate the signing key")

	_, err := s.Apply(context.Background(), one.ID, SetDueOn{DueOn: dueDate("2026-10-14T00:00:00Z")},
		ApplyOptions{Actor: testActor})
	if err == nil {
		t.Fatal("an instant was accepted where a date belongs")
	}
	if code := CodeOf(err); code != CodeInvalidField {
		t.Errorf("code = %q, want %q", code, CodeInvalidField)
	}

	got := getTicket(t, s, one.ID)
	if got.DueOn != nil {
		t.Errorf("DueOn = %q, want the refused write to have changed nothing", *got.DueOn)
	}
}

// TestSetDueOnClearsWithEmpty pins the clearing spelling the CLI relies on,
// where --due-on "" means no deadline.
func TestSetDueOnClearsWithEmpty(t *testing.T) {
	s := newTestStore(t)
	one := createDue(t, s, "Rotate the signing key", "2026-10-14")

	mustApply(t, s, one.ID, SetDueOn{DueOn: dueDate("")})
	if got := getTicket(t, s, one.ID); got.DueOn != nil {
		t.Errorf("DueOn = %q, want nil", *got.DueOn)
	}
}

// TestCreateRefusesAnInstant checks the same rule on the other write path. A
// bad date costs no ID and writes no file.
func TestCreateRefusesAnInstant(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Create(context.Background(), CreateOptions{
		Title: "Rotate the signing key",
		DueOn: dueDate("2026-10-14T00:00:00Z"),
		Actor: testActor,
	})
	if err == nil {
		t.Fatal("an instant was accepted where a date belongs")
	}
	if code := CodeOf(err); code != CodeInvalidField {
		t.Errorf("code = %q, want %q", code, CodeInvalidField)
	}

	all, err := s.List(context.Background(), Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Errorf("the store holds %d tickets, want the refused create to have written none", len(all))
	}
}

// TestDueOnSurvivesTheRoundTrip covers the populated path through the parser
// and the renderer. The corpus holds this too, in full.md, but a store write
// reaches it by a different road.
func TestDueOnSurvivesTheRoundTrip(t *testing.T) {
	s := newTestStore(t)
	one := createDue(t, s, "Rotate the signing key", "2026-10-14")

	got := getTicket(t, s, one.ID)
	if got.DueOn == nil || *got.DueOn != "2026-10-14" {
		t.Fatalf("DueOn = %v, want 2026-10-14", got.DueOn)
	}
}
