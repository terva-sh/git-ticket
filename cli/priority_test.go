package cli

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// makePriorityTicket files a ticket at a priority, optionally with a deadline,
// and promotes it so ready can see it.
func makePriorityTicket(t *testing.T, dir, title, priority, due string) string {
	t.Helper()
	args := []string{"--json", "create", "--title", title, "--actor", "human:sothr", "--priority", priority}
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

// TestReadyRanksByPriorityFromTheCLI is plan 8 through the binary. ready ranks
// and needs no flag to do it, and priority is the first key.
//
// The low ticket carries the only deadline in the store, so a run that put it
// first would be ordering by the deadline.
func TestReadyRanksByPriorityFromTheCLI(t *testing.T) {
	dir := newStore(t)

	lowDated := makePriorityTicket(t, dir, "Low but due soon", "low", "2026-10-07")
	urgent := makePriorityTicket(t, dir, "Urgent with no date", "urgent", "")
	normal := makePriorityTicket(t, dir, "Ordinary work", "normal", "")

	want := []string{urgent, normal, lowDated}
	if got := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); !reflect.DeepEqual(got, want) {
		t.Errorf("ready = %v, want urgent first and the dated low ticket last %v", got, want)
	}
}

// TestListSortPriority covers the flag added so a field that ranks one command
// can order the other, per section 8's "one field has one order".
//
// The expectation is spelled out rather than compared against ready, because
// two commands sharing one sort function agree with each other whether or not
// that function is right.
func TestListSortPriority(t *testing.T) {
	dir := newStore(t)

	lowDated := makePriorityTicket(t, dir, "Low but due soon", "low", "2026-10-07")
	urgent := makePriorityTicket(t, dir, "Urgent with no date", "urgent", "")
	normal := makePriorityTicket(t, dir, "Ordinary work", "normal", "")

	want := []string{urgent, normal, lowDated}
	got := idsOf(t, runCLI(t, dir, nil, "--json", "list", "--sort", "priority"))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("list --sort priority = %v, want %v", got, want)
	}

	// The default is unchanged: ID order, so reordering a report stays something
	// the caller asks for.
	//
	// Assert what the default is rather than what it is not. "Not the priority
	// order" reads like the stronger claim and is a coin flip: three tickets filed
	// in one millisecond share ten characters of ULID timestamp and differ only in
	// the random half, so the ID order lands on the priority order about one run
	// in six. That is how this shipped green and failed CI two commits later.
	byID := idsOf(t, runCLI(t, dir, nil, "--json", "list"))
	wantByID := append([]string(nil), byID...)
	sort.Strings(wantByID)
	if !reflect.DeepEqual(byID, wantByID) {
		t.Errorf("list with no --sort = %v, want ID order %v", byID, wantByID)
	}
}

// TestSortPriorityIsAccepted is the positive half of
// TestListRefusesAValueOutsideItsSet, which used to name priority as its
// invalid example.
func TestSortPriorityIsAccepted(t *testing.T) {
	dir := newStore(t)
	makePriorityTicket(t, dir, "Something to list", "high", "")

	if got := runCLI(t, dir, nil, "list", "--sort", "priority"); got.code != exitOK {
		t.Errorf("list --sort priority exited %d: %s", got.code, got.stderr)
	}
}

// TestSortHelpNamesEveryOrder keeps the help text from naming a set the
// validation does not accept. Both read sortOrders, so this fails when somebody
// adds an order and hand-writes the help string instead.
func TestSortHelpNamesEveryOrder(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "list", "--help")
	text := got.stdout + got.stderr

	for _, order := range sortOrders {
		if !strings.Contains(text, order) {
			t.Errorf("the --sort help does not name %q:\n%s", order, text)
		}
	}
}
