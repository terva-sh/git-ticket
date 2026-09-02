package ticket

import (
	"context"
	"reflect"
	"testing"
)

// finish walks a ticket to done, because 6.2 has no shortcut from draft.
func finish(t *testing.T, s *Store, id string) {
	t.Helper()
	mustApply(t, s, id, SetStatus{Status: StatusReady})
	mustApply(t, s, id, SetStatus{Status: StatusInProgress})
	mustApply(t, s, id, SetStatus{Status: StatusDone})
}

// child creates a ticket parented to epic.
func child(t *testing.T, s *Store, epic, title string) *Ticket {
	t.Helper()
	c := mustCreate(t, s, title)
	mustApply(t, s, c.ID, SetParent{Parent: &epic})
	return c
}

func readinessFor(t *testing.T, s *Store, id string) Readiness {
	t.Helper()
	all, err := s.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return all[id]
}

// TestBlocksOnChildrenGatesOnTheDecomposition pins the derivation: an epic that
// blocks on children waits for them, and it names them in a field of their own
// rather than in Blocking.
func TestBlocksOnChildrenGatesOnTheDecomposition(t *testing.T) {
	s := newTestStore(t)

	epic := mustCreate(t, s, "Ship token refresh")
	mustApply(t, s, epic.ID, SetBlocksOn{BlocksOn: BlocksOnChildren})
	mustApply(t, s, epic.ID, SetStatus{Status: StatusReady})

	one := child(t, s, epic.ID, "Refresh endpoint")
	two := child(t, s, epic.ID, "Rotate on expiry")

	r := readinessFor(t, s, epic.ID)
	if !r.Blocked {
		t.Error("an epic with unfinished children is blocked")
	}
	if r.Ready {
		t.Error("a blocked epic is not ready")
	}
	want := []string{one.ID, two.ID}
	if len(r.BlockingChildren) != 2 {
		t.Fatalf("BlockingChildren = %v, want both children", r.BlockingChildren)
	}
	got := append([]string(nil), r.BlockingChildren...)
	if !reflect.DeepEqual(sorted(got), sorted(want)) {
		t.Errorf("BlockingChildren = %v, want %v", got, want)
	}
	// The whole point of the separate field: a consumer reading Blocking gets
	// dependencies and only dependencies, exactly as it did before.
	if len(r.Blocking) != 0 {
		t.Errorf("Blocking = %v, want empty: a child is not a dependency", r.Blocking)
	}

	// Finishing one leaves the other in the way.
	finish(t, s, one.ID)
	r = readinessFor(t, s, epic.ID)
	if !reflect.DeepEqual(r.BlockingChildren, []string{two.ID}) {
		t.Errorf("BlockingChildren = %v, want just %s", r.BlockingChildren, two.ID)
	}

	// Finishing both clears it.
	finish(t, s, two.ID)
	r = readinessFor(t, s, epic.ID)
	if r.Blocked || len(r.BlockingChildren) != 0 {
		t.Errorf("an epic whose children are all done is not blocked: %+v", r)
	}
	if !r.Ready {
		t.Error("an unblocked, unclaimed, ready epic is ready")
	}
}

// TestBlocksOnChildrenWithNoChildrenIsNotBlocked pins the settled answer.
//
// Blocking it would put a ticket in blocked with nothing to name as the
// blocker, which section 8 refuses for a draft on the same grounds. check
// reports the state as blocks_on_no_children instead.
func TestBlocksOnChildrenWithNoChildrenIsNotBlocked(t *testing.T) {
	s := newTestStore(t)

	epic := mustCreate(t, s, "Undecomposed epic")
	mustApply(t, s, epic.ID, SetBlocksOn{BlocksOn: BlocksOnChildren})
	mustApply(t, s, epic.ID, SetStatus{Status: StatusReady})

	r := readinessFor(t, s, epic.ID)
	if r.Blocked {
		t.Error("an epic with no children has no blocker to name, so it is not blocked")
	}
	if len(r.BlockingChildren) != 0 {
		t.Errorf("BlockingChildren = %v, want empty", r.BlockingChildren)
	}
	if !r.Ready {
		t.Error("status is the guard here, not readiness")
	}
}

// TestBlocksOnIsAdditive is the regression guard on the correction that
// implementing this forced.
//
// The field says what gates a ticket beyond its dependencies. It never switches
// dependency gating off. An earlier draft made the enum selective with none as
// the default, which would have disabled dependency blocking for every ticket
// in every store the moment the field shipped, since none renders on all of
// them.
func TestBlocksOnIsAdditive(t *testing.T) {
	s := newTestStore(t)

	// none is the default and every migrated ticket carries it. Dependencies
	// must still gate.
	blocker := mustCreate(t, s, "Has to land first")
	waiter := mustCreate(t, s, "Waits on the blocker")
	mustApply(t, s, waiter.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, waiter.ID, AddDependency{ID: blocker.ID})
	if got := readinessFor(t, s, waiter.ID); !got.Blocked ||
		!reflect.DeepEqual(got.Blocking, []string{blocker.ID}) {
		t.Errorf("blocks_on none must leave dependency gating alone: %+v", got)
	}

	// children adds the child edge and keeps the dependency edge.
	epic := mustCreate(t, s, "Epic with both kinds of edge")
	mustApply(t, s, epic.ID, SetBlocksOn{BlocksOn: BlocksOnChildren})
	mustApply(t, s, epic.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, epic.ID, AddDependency{ID: blocker.ID})
	kid := child(t, s, epic.ID, "One piece of it")

	r := readinessFor(t, s, epic.ID)
	if !reflect.DeepEqual(r.Blocking, []string{blocker.ID}) {
		t.Errorf("Blocking = %v, want the dependency still gating", r.Blocking)
	}
	if !reflect.DeepEqual(r.BlockingChildren, []string{kid.ID}) {
		t.Errorf("BlockingChildren = %v, want the child too", r.BlockingChildren)
	}

	// Clearing the children alone does not unblock it, because the dependency
	// is still in the way.
	finish(t, s, kid.ID)
	if got := readinessFor(t, s, epic.ID); !got.Blocked || len(got.Blocking) != 1 {
		t.Errorf("the dependency outlives the children: %+v", got)
	}
}

// TestBlocksOnNoneIgnoresChildren keeps the default inert. A parent that never
// asked to gate on its children does not start doing so because somebody
// parented a ticket to it.
func TestBlocksOnNoneIgnoresChildren(t *testing.T) {
	s := newTestStore(t)

	parent := mustCreate(t, s, "Ordinary parent")
	mustApply(t, s, parent.ID, SetStatus{Status: StatusReady})
	child(t, s, parent.ID, "An unfinished child")

	r := readinessFor(t, s, parent.ID)
	if r.Blocked || len(r.BlockingChildren) != 0 {
		t.Errorf("blocks_on none ignores children entirely: %+v", r)
	}
	if !r.Ready {
		t.Error("a parent with blocks_on none is ready despite an open child")
	}
}

func sorted(in []string) []string {
	out := append([]string(nil), in...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
