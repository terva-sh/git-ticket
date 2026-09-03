package cli

import (
	"reflect"
	"slices"
	"testing"
)

// dueOnOf reads the deadline back through the JSON contract. A ticket with no
// deadline carries an explicit null, so a missing key and an absent value are
// not the same answer.
func dueOnOf(t *testing.T, dir, id string) (string, bool) {
	t.Helper()
	raw, present := showTicket(t, dir, id)["dueOn"]
	if !present {
		t.Fatalf("the ticket JSON carries no dueOn key at all")
	}
	if raw == nil {
		return "", false
	}
	s, ok := raw.(string)
	if !ok {
		t.Fatalf("dueOn = %v, want a string or null", raw)
	}
	return s, true
}

// makeDueTicket files a ticket with a deadline and moves it to ready, which is
// what the ready query filters on.
func makeDueTicket(t *testing.T, dir, title, due string) string {
	t.Helper()
	args := []string{"--json", "create", "--title", title, "--actor", "human:sothr"}
	if due != "" {
		args = append(args, "--due-on", due)
	}
	got := runCLI(t, dir, nil, args...)
	if got.code != exitOK {
		t.Fatalf("create %q: %s", title, got.stderr)
	}
	id := ticketID(t, decode(t, got.stdout))
	if r := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); r.code != exitOK {
		t.Fatalf("status ready: %s", r.stderr)
	}
	return id
}

// TestDueOnRoundTripsThroughTheCLI covers the ordinary path: a date goes in on
// create and comes back out of the JSON contract.
func TestDueOnRoundTripsThroughTheCLI(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--due-on", "2026-10-14"))

	got, set := dueOnOf(t, dir, id)
	if !set || got != "2026-10-14" {
		t.Errorf("dueOn = %q (set %v), want 2026-10-14", got, set)
	}
}

// TestCreateRefusesAnInstantOnTheCommandLine is the acceptance criterion that
// the CLI refuses an instant where a date belongs rather than truncating it.
//
// An RFC3339 instant is the realistic mistake, because every other time value
// in the format is one.
func TestCreateRefusesAnInstantOnTheCommandLine(t *testing.T) {
	dir := newStore(t)

	for _, bad := range []string{"2026-10-14T00:00:00Z", "2026-1-4", "tomorrow"} {
		got := runCLI(t, dir, nil, "create", "--title", "Rotate the key",
			"--due-on", bad, "--actor", "human:sothr")
		if got.code == exitOK {
			t.Errorf("create --due-on %q was accepted", bad)
		}
	}

	// Nothing was filed by any of them.
	list := runCLI(t, dir, nil, "--json", "list")
	if ids := idsOf(t, list); len(ids) != 0 {
		t.Errorf("the store holds %v, want nothing written by a refused create", ids)
	}
}

// TestUpdateSetsClearsAndLeavesTheDeadline covers the three states of a flag
// whose zero value is a legal instruction. Empty clears it, and no flag at all
// leaves it alone, so runUpdate has to ask which flags were given rather than
// which are non-empty.
func TestUpdateSetsClearsAndLeavesTheDeadline(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	if got := runCLI(t, dir, nil, "update", id, "--due-on", "2026-11-05",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --due-on: %s", got.stderr)
	}
	if got, set := dueOnOf(t, dir, id); !set || got != "2026-11-05" {
		t.Fatalf("dueOn = %q (set %v), want 2026-11-05", got, set)
	}

	// Another field, with no --due-on: the deadline survives.
	if got := runCLI(t, dir, nil, "update", id, "--priority", "high",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --priority: %s", got.stderr)
	}
	if got, set := dueOnOf(t, dir, id); !set || got != "2026-11-05" {
		t.Errorf("dueOn = %q (set %v), want an untouched 2026-11-05", got, set)
	}

	// An explicit empty value clears it.
	if got := runCLI(t, dir, nil, "update", id, "--due-on", "",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --due-on \"\": %s", got.stderr)
	}
	if got, set := dueOnOf(t, dir, id); set {
		t.Errorf("dueOn = %q, want it cleared", got)
	}

	// And an instant is refused here too, leaving the ticket as it was.
	if got := runCLI(t, dir, nil, "update", id, "--due-on", "2026-11-05T09:00:00Z",
		"--actor", "human:sothr"); got.code == exitOK {
		t.Error("update accepted an instant where a date belongs")
	}
	if got, set := dueOnOf(t, dir, id); set {
		t.Errorf("dueOn = %q, want the refused update to have changed nothing", got)
	}
}

// TestReadyOrdersByTheDeadlineWithNoFlag is the settled default of plan 8.
func TestReadyOrdersByTheDeadlineWithNoFlag(t *testing.T) {
	dir := newStore(t)
	late := makeDueTicket(t, dir, "Due in December", "2026-12-01")
	soon := makeDueTicket(t, dir, "Due in January", "2026-01-05")
	none := makeDueTicket(t, dir, "No deadline at all", "")

	got := runCLI(t, dir, nil, "--json", "ready")
	if got.code != exitOK {
		t.Fatalf("ready: %s", got.stderr)
	}
	want := []string{soon, late, none}
	if ids := idsOf(t, got); !reflect.DeepEqual(ids, want) {
		t.Errorf("ready = %v, want soonest first and undated last %v", ids, want)
	}
}

// TestListStaysChronologicalUntilAsked is the other half of the asymmetry. A
// list reports what exists, so the caller asks before it is reordered.
func TestListStaysChronologicalUntilAsked(t *testing.T) {
	dir := newStore(t)
	late := makeDueTicket(t, dir, "Due in December", "2026-12-01")
	soon := makeDueTicket(t, dir, "Due in January", "2026-01-05")
	none := makeDueTicket(t, dir, "No deadline at all", "")

	// The default is the ID order, which is what the store answered before
	// due_on existed. Compared against a sorted set rather than creation order,
	// because tickets filed in one millisecond carry no guarantee of which ID
	// is lower.
	byID := []string{late, soon, none}
	slices.Sort(byID)
	plain := runCLI(t, dir, nil, "--json", "list")
	if ids := idsOf(t, plain); !reflect.DeepEqual(ids, byID) {
		t.Errorf("list = %v, want the ID order %v", ids, byID)
	}

	sorted := runCLI(t, dir, nil, "--json", "list", "--sort", "due_on")
	if sorted.code != exitOK {
		t.Fatalf("list --sort due_on: %s", sorted.stderr)
	}
	want := []string{soon, late, none}
	if ids := idsOf(t, sorted); !reflect.DeepEqual(ids, want) {
		t.Errorf("list --sort due_on = %v, want %v", ids, want)
	}
}

// TestListDueByIsInclusiveAndSkipsUndated holds the bound. The inclusive end is
// what "due by the end of the month" means, and a ticket with no deadline is
// not due by any date.
func TestListDueByIsInclusiveAndSkipsUndated(t *testing.T) {
	dir := newStore(t)
	onBound := makeDueTicket(t, dir, "Due on the bound", "2026-06-30")
	before := makeDueTicket(t, dir, "Due before it", "2026-06-01")
	after := makeDueTicket(t, dir, "Due after it", "2026-07-01")
	none := makeDueTicket(t, dir, "Never due", "")

	got := runCLI(t, dir, nil, "--json", "list", "--due-by", "2026-06-30")
	if got.code != exitOK {
		t.Fatalf("list --due-by: %s", got.stderr)
	}
	ids := idsOf(t, got)
	if !slices.Contains(ids, onBound) {
		t.Error("the bound itself did not match, so --due-by is exclusive")
	}
	if !slices.Contains(ids, before) {
		t.Error("a ticket due before the bound did not match")
	}
	if slices.Contains(ids, after) {
		t.Error("a ticket due after the bound matched")
	}
	if slices.Contains(ids, none) {
		t.Error("an undated ticket matched a deadline bound")
	}
}

// TestListRefusesAValueOutsideItsSet keeps a typo from answering with a
// straight face. A bound that is not a date would otherwise compare as a string
// against every ticket.
func TestListRefusesAValueOutsideItsSet(t *testing.T) {
	dir := newStore(t)
	makeDueTicket(t, dir, "Due in December", "2026-12-01")

	// priority used to sit in this list and is now one of the orders --sort
	// takes, per plan 8. created is the replacement because it is the shape of
	// the mistake worth catching: a plausible key that does not exist.
	for _, args := range [][]string{
		{"list", "--due-by", "nonsense"},
		{"list", "--due-by", "2026-12-01T00:00:00Z"},
		{"list", "--sort", "created"},
	} {
		if got := runCLI(t, dir, nil, args...); got.code == exitOK {
			t.Errorf("%v was accepted", args)
		}
	}
}
