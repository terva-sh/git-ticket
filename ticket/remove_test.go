package ticket

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func mustRemove(t *testing.T, s *Store, ref string, o RemoveOptions) *RemoveResult {
	t.Helper()
	res, err := s.Remove(context.Background(), ref, o)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	return res
}

// refuses runs a removal that is meant to fail and returns the coded error.
// Every refusal has to leave the file where it was, so this asserts that too:
// a refusal that deletes anyway is the failure this whole command exists to
// avoid.
func refuses(t *testing.T, s *Store, ref string, o RemoveOptions, want string) *Error {
	t.Helper()
	before, err := s.Get(context.Background(), ref)
	if err != nil {
		t.Fatalf("get before: %v", err)
	}
	_, err = s.Remove(context.Background(), ref, o)
	if err == nil {
		t.Fatalf("remove succeeded, want %s", want)
	}
	if got := CodeOf(err); got != want {
		t.Fatalf("code = %q, want %q (%v)", got, want, err)
	}
	if _, statErr := os.Stat(before.Path); statErr != nil {
		t.Errorf("the refusal deleted the file anyway: %v", statErr)
	}
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("not a coded error: %v", err)
	}
	return e
}

// TestRemoveDeletesATicketNobodyTouched is the case section 9.1 is for: a
// ticket filed by mistake, before anybody worked it.
func TestRemoveDeletesATicketNobodyTouched(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Filed against the wrong repository")
	path := tk.Path

	res := mustRemove(t, s, tk.ID, RemoveOptions{})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the file is still there: %v", err)
	}
	// 9.1 has remove return the ticket it removed, which is the last state
	// rather than a new one. A caller that wants it back writes this out again.
	if res.Ticket.ID != tk.ID {
		t.Errorf("returned ID = %s, want %s", res.Ticket.ID, tk.ID)
	}
	if res.Ticket.Title != tk.Title {
		t.Errorf("the returned ticket is not the one removed: title = %q", res.Ticket.Title)
	}
	if len(res.PathsChanged) != 1 || res.PathsChanged[0] != path {
		t.Errorf("pathsChanged = %v, want [%s]", res.PathsChanged, path)
	}
	if len(res.Dangling) != 0 {
		t.Errorf("a removal that broke nothing reported %v", res.Dangling)
	}
}

// TestRemoveLeavesTheStoreCheckable is the finding that shapes 9.1: rm is not
// the problem, because deleting an unreferenced ticket leaves check green. The
// refusals are the product, so this holds the easy half in place.
func TestRemoveLeavesTheStoreCheckable(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "The ticket that stays")
	doomed := mustCreate(t, s, "The ticket filed by mistake")

	mustRemove(t, s, doomed.ID, RemoveOptions{})

	rep, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !rep.OK() {
		t.Errorf("removing an unreferenced ticket broke the store: %v", rep.Errors)
	}
}

// TestRemoveRefusesATicketAnotherDependsOn covers the first row of the 9.1
// table. check --fix declines this repair, so remove has to decline the write.
func TestRemoveRefusesATicketAnotherDependsOn(t *testing.T) {
	s := newTestStore(t)
	target := mustCreate(t, s, "The dependency")
	dependent := mustCreate(t, s, "The ticket waiting on it")
	mustApply(t, s, dependent.ID, AddDependency{ID: target.ID})

	e := refuses(t, s, target.ID, RemoveOptions{}, CodeTicketReferenced)

	// Naming the referrer is the whole of what this buys over rm, since finding
	// it by hand means grepping the store for an ID.
	if !strings.Contains(e.Message, dependent.ID) {
		t.Errorf("the refusal does not name the ticket pointing at it:\n%s", e.Message)
	}
	if !strings.Contains(e.Message, dependent.Title) {
		t.Errorf("the refusal names an ID with no title:\n%s", e.Message)
	}
	if e.Details["referencedBy"] != dependent.ID {
		t.Errorf("referencedBy = %q, want %q", e.Details["referencedBy"], dependent.ID)
	}
	if !strings.Contains(e.Message, "unlink") {
		t.Errorf("the refusal does not name the repair:\n%s", e.Message)
	}
}

// TestRemoveRefusesAnEpicItsChildrenName covers the parent half of the same
// row. Deps with Dependents walks dependencies alone, so this is the case a
// scan built on it would miss.
func TestRemoveRefusesAnEpicItsChildrenName(t *testing.T) {
	s := newTestStore(t)
	epic := mustCreate(t, s, "The epic")
	child := mustCreate(t, s, "A child of the epic")
	parent := epic.ID
	mustApply(t, s, child.ID, SetParent{Parent: &parent})

	e := refuses(t, s, epic.ID, RemoveOptions{}, CodeTicketReferenced)

	if !strings.Contains(e.Message, child.ID) {
		t.Errorf("the refusal does not name the child:\n%s", e.Message)
	}
	// unlink drops a dependency and does nothing to a parent, so naming it here
	// would send a reader to a command that reports success and changes nothing.
	if strings.Contains(e.Message, "unlink") {
		t.Errorf("the refusal offers unlink for a parent reference:\n%s", e.Message)
	}
	if !strings.Contains(e.Message, "parent") {
		t.Errorf("the refusal does not say the reference is a parent:\n%s", e.Message)
	}
}

// TestRemoveNamesEachReferrerOnce covers a ticket that names the target twice,
// as its parent and as a dependency. That is one referrer to repair, not two.
func TestRemoveNamesEachReferrerOnce(t *testing.T) {
	s := newTestStore(t)
	target := mustCreate(t, s, "Named twice by one ticket")
	other := mustCreate(t, s, "The ticket naming it twice")
	id := target.ID
	mustApply(t, s, other.ID, SetParent{Parent: &id})
	mustApply(t, s, other.ID, AddDependency{ID: target.ID})

	e := refuses(t, s, target.ID, RemoveOptions{}, CodeTicketReferenced)

	if got := e.Details["referencedBy"]; got != other.ID {
		t.Errorf("referencedBy = %q, want the referrer once: %q", got, other.ID)
	}
	if n := strings.Count(e.Message, other.ID); n != 1 {
		t.Errorf("the message names the referrer %d times, want 1:\n%s", n, e.Message)
	}
	// Both fields point here, so both repairs have to be named.
	for _, want := range []string{"unlink", "parent"} {
		if !strings.Contains(e.Message, want) {
			t.Errorf("the refusal does not mention %q:\n%s", want, e.Message)
		}
	}
}

// TestRemoveRefusesWorkSomebodyDid covers the second row of the 9.1 table.
// Each of these is somebody having touched the ticket after filing it, and
// archive is the operation for work that happened.
func TestRemoveRefusesWorkSomebodyDid(t *testing.T) {
	cases := []struct {
		name string
		work []Mutation
	}{
		{"a note", []Mutation{AppendNote{Text: "what I found"}}},
		{"a comment", []Mutation{AppendComment{Text: "a question"}}},
		{"a summary", []Mutation{SetSummary{Text: "what shipped"}}},
		{"a claim", []Mutation{SetStatus{Status: StatusReady}, ClaimTicket{}}},
		{"an archive record", []Mutation{ArchiveTicket{Reason: "superseded"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestStore(t)
			tk := mustCreate(t, s, "A ticket somebody worked")
			for _, m := range c.work {
				mustApply(t, s, tk.ID, m)
			}

			e := refuses(t, s, tk.ID, RemoveOptions{}, CodeTicketTouched)

			// The refusal has to say what it found, or a person cannot tell
			// which of five conditions stopped them.
			if !strings.Contains(e.Message, "archive") && !strings.Contains(e.Message, "--force") {
				t.Errorf("the refusal names neither way forward:\n%s", e.Message)
			}
		})
	}
}

// TestRemoveTakesAFiledTicketWithItsPlan is the disagreement 9.1 used to carry.
// Its prose said the body must hold a description and nothing else, but create
// seeds a plan, acceptance criteria, and a definition of done at filing time,
// per 12.1. Refusing on those would refuse exactly the mistakes filed best.
func TestRemoveTakesAFiledTicketWithItsPlan(t *testing.T) {
	s := newTestStore(t)
	res, err := s.Create(context.Background(), CreateOptions{
		Title:              "Filed carefully against the wrong repository",
		Description:        "A description.",
		ImplementationPlan: "Step one, step two.",
		AcceptanceCriteria: []string{"the first thing", "the second thing"},
		DefinitionOfDone:   []string{"tests pass"},
		Actor:              testActor,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	path := res.Ticket.Path

	mustRemove(t, s, res.Ticket.ID, RemoveOptions{})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a ticket filed with a plan was treated as worked: %v", err)
	}
}

// TestRemoveForceReportsWhatItBroke holds the override to its bargain. It
// overrides rather than refusing absolutely because a refusal a person routes
// around with rm has taught them nothing, so the report is what it buys back.
func TestRemoveForceReportsWhatItBroke(t *testing.T) {
	s := newTestStore(t)
	target := mustCreate(t, s, "The dependency")
	dependent := mustCreate(t, s, "The ticket waiting on it")
	child := mustCreate(t, s, "A child of it")
	id := target.ID
	mustApply(t, s, dependent.ID, AddDependency{ID: target.ID})
	mustApply(t, s, child.ID, SetParent{Parent: &id})
	path := target.Path

	res := mustRemove(t, s, target.ID, RemoveOptions{Force: true})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("--force did not remove the file: %v", err)
	}
	if len(res.Dangling) != 2 {
		t.Fatalf("dangling = %v, want both references", res.Dangling)
	}
	byField := map[string]Dangling{}
	for _, d := range res.Dangling {
		byField[d.Field] = d
	}
	if got := byField["dependencies"]; got.Ticket != dependent.ID || got.Title != dependent.Title {
		t.Errorf("the dependency referrer is %+v, want %s (%s)", got, dependent.ID, dependent.Title)
	}
	if got := byField["parent"]; got.Ticket != child.ID || got.Title != child.Title {
		t.Errorf("the parent referrer is %+v, want %s (%s)", got, child.ID, child.Title)
	}

	// The report is only worth having if it matches what check now finds.
	rep, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(rep.Errors) != 2 {
		t.Errorf("check found %d errors, want the two dangling references: %v", len(rep.Errors), rep.Errors)
	}
}

// TestRemoveForceTakesWorkedTickets covers the other half of the override.
func TestRemoveForceTakesWorkedTickets(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "A ticket somebody worked")
	mustApply(t, s, tk.ID, AppendNote{Text: "what I found"})
	path := tk.Path

	res := mustRemove(t, s, tk.ID, RemoveOptions{Force: true})

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("--force did not remove the file: %v", err)
	}
	if len(res.Dangling) != 0 {
		t.Errorf("nothing pointed at it, so nothing dangles: %v", res.Dangling)
	}
}

// TestRemoveHonoursTheRevisionPrecondition holds remove to the guarantee every
// other write gives. A flag that promises to refuse a stale write is worth
// nothing if the one destructive command skips it.
func TestRemoveHonoursTheRevisionPrecondition(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Changed under the reader")
	stale := tk.Revision
	mustApply(t, s, tk.ID, SetDescription{Text: "somebody else edited this"})

	e := refuses(t, s, tk.ID, RemoveOptions{IfRevision: stale}, CodeStaleRevision)

	if e.Details["expected"] != stale {
		t.Errorf("expected = %q, want the stale revision %q", e.Details["expected"], stale)
	}
	if e.Details["actual"] == stale {
		t.Errorf("actual is the stale revision, so nothing was compared")
	}
	// A lost race is the failure an agent hits most, and it is worth naming the
	// ticket it lost.
	if e.Title != tk.Title {
		t.Errorf("the staleness does not name the ticket: title = %q", e.Title)
	}

	// The current revision goes through.
	current, err := s.Get(context.Background(), tk.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	mustRemove(t, s, tk.ID, RemoveOptions{IfRevision: current.Revision})
}

// TestRemoveResolvesAPrefix holds remove to 5.5 like every other command that
// takes an ID. A destructive command that demands the full 26 characters would
// be the one place a person pastes an ID by hand.
func TestRemoveResolvesAPrefix(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Removed by prefix")

	res := mustRemove(t, s, tk.ID[:12], RemoveOptions{})

	if res.Ticket.ID != tk.ID {
		t.Errorf("resolved to %s, want %s", res.Ticket.ID, tk.ID)
	}
}

// TestRemoveRejectsAnUnknownTicket covers the ordinary miss.
func TestRemoveRejectsAnUnknownTicket(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "The only ticket here")

	_, err := s.Remove(context.Background(), "TKT-01K400GBC0WHK14CSBHQ8WSEPZ", RemoveOptions{})
	if got := CodeOf(err); got != CodeTicketNotFound {
		t.Errorf("code = %q, want %q (%v)", got, CodeTicketNotFound, err)
	}
}
