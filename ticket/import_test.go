package ticket

import (
	"context"
	"os"
	"strings"
	"testing"
)

// These are the tests the refactor exists for: the interchange is reachable
// without building argv and reading prose. Every earlier test of this behaviour
// runs the CLI, which is the surface a host cannot use.

// exportOneTicket is a patch carrying a single ticket out of a store, the way
// export writes one.
func exportOneTicket(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the sending ticket: %v", err)
	}
	hunk, _ := AddedFileHunk(".tickets/tickets/incoming.md", data)
	return hunk
}

// TestPlanImportDecidesAndApplyImportFiles covers the pair over one ticket
// carrying both halves of the interchange rule: a checklist, which is the work
// and travels, and a label this store never declared, which does not.
func TestPlanImportDecidesAndApplyImportFiles(t *testing.T) {
	ctx := context.Background()

	src := newTestStore(t)
	sent, err := src.Create(ctx, CreateOptions{
		Title:              "Carry this across",
		AcceptanceCriteria: []string{"first", "second"},
		Actor:              testActor,
	})
	if err != nil {
		t.Fatalf("create in the sending store: %v", err)
	}
	patch := exportOneTicket(t, sent.Ticket.Path)

	dst := newTestStore(t)
	plan, err := dst.PlanImport(ctx, ImportOptions{
		Patch:     patch,
		FromStore: "the-other-project",
		Actor:     testActor,
	})
	if err != nil {
		t.Fatalf("PlanImport: %v", err)
	}
	if len(plan.Tickets) != 1 {
		t.Fatalf("planned %d tickets, want 1", len(plan.Tickets))
	}

	pt := plan.Tickets[0]
	if pt.Incoming.ID != sent.Ticket.ID {
		t.Errorf("plan names %s, want the sender's %s", pt.Incoming.ID, sent.Ticket.ID)
	}
	if len(pt.Create.AcceptanceCriteria) != 2 {
		t.Errorf("plan carries %d criteria, want 2", len(pt.Create.AcceptanceCriteria))
	}
	if !hasChange(pt.Changes, ChangeAcceptanceCriteriaUnchecked) {
		t.Errorf("the plan does not report the checklist as unchecked: %+v", pt.Changes)
	}

	res, err := dst.ApplyImport(ctx, plan)
	if err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}
	if len(res.Filed) != 1 {
		t.Fatalf("filed %d tickets, want 1", len(res.Filed))
	}
	// The map from the ID that arrived to the ID this store minted is the one
	// thing a caller cannot reconstruct, which is why the result carries it.
	if res.Filed[0].FromID != sent.Ticket.ID {
		t.Errorf("FromID = %s, want %s", res.Filed[0].FromID, sent.Ticket.ID)
	}
	if res.Filed[0].ID == sent.Ticket.ID {
		t.Error("the ticket kept the sender's ID; adopting means reminting")
	}

	landed, err := dst.Get(ctx, res.Filed[0].ID)
	if err != nil {
		t.Fatalf("read the filed ticket: %v", err)
	}
	body, err := os.ReadFile(landed.Path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(body), "- [x]"); n != 0 {
		t.Errorf("%d boxes arrived ticked; a tick is evidence about the sender's work", n)
	}
	if n := strings.Count(string(body), "- [ ]"); n != 2 {
		t.Errorf("%d unticked boxes, want 2", n)
	}
	if !strings.Contains(string(body), "origin-ticket:"+sent.Ticket.ID) {
		t.Error("the filed ticket does not record where it came from")
	}
}

// TestPlanImportWritesNothing is the promise the preview makes. A plan that
// quietly filed something would make --adopt a formality rather than consent.
func TestPlanImportWritesNothing(t *testing.T) {
	ctx := context.Background()

	src := newTestStore(t)
	sent, err := src.Create(ctx, CreateOptions{Title: "Stay put", Actor: testActor})
	if err != nil {
		t.Fatal(err)
	}
	patch := exportOneTicket(t, sent.Ticket.Path)

	dst := newTestStore(t)
	before, err := dst.List(ctx, Filter{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dst.PlanImport(ctx, ImportOptions{Patch: patch, Actor: testActor}); err != nil {
		t.Fatalf("PlanImport: %v", err)
	}
	after, err := dst.List(ctx, Filter{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Errorf("planning filed %d tickets", len(after)-len(before))
	}
}

func hasChange(changes []Change, kind ChangeKind) bool {
	for _, c := range changes {
		if c.Kind == kind {
			return true
		}
	}
	return false
}

// TestSameOwnerPlansTheEvidenceAndApplyCarriesIt is the library half of the
// second case on the interchange rule.
//
// The split it holds is the load-bearing one: PlanImport decides which boxes
// travel and ApplyImport ticks exactly those. A preview renders the plan, so an
// ApplyImport that worked the ticks out for itself could tick a box the preview
// never promised, and nobody would find out from the output.
func TestSameOwnerPlansTheEvidenceAndApplyCarriesIt(t *testing.T) {
	ctx := context.Background()

	src := newTestStore(t)
	sent, err := src.Create(ctx, CreateOptions{
		Title:              "Move this between my own stores",
		AcceptanceCriteria: []string{"met", "not met"},
		Actor:              testActor,
	})
	if err != nil {
		t.Fatalf("create in the sending store: %v", err)
	}
	if _, err := src.Apply(ctx, sent.Ticket.ID,
		SetChecklistItem{Section: AcceptanceCriteria, Index: 1, Checked: true},
		ApplyOptions{Actor: testActor}); err != nil {
		t.Fatalf("tick the first criterion: %v", err)
	}
	moved, err := src.Get(ctx, sent.Ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	patch := exportOneTicket(t, moved.Path)

	dst := newTestStore(t)
	plan, err := dst.PlanImport(ctx, ImportOptions{
		Patch:     patch,
		FromStore: "my-other-store",
		SameOwner: true,
		Actor:     testActor,
	})
	if err != nil {
		t.Fatalf("PlanImport: %v", err)
	}
	pt := plan.Tickets[0]

	// The plan names the ticks, one of two, and says so as a carry rather than
	// as the loss the default reports.
	want := []ChecklistTick{{Section: AcceptanceCriteria, Index: 1}}
	if len(pt.Ticks) != len(want) || pt.Ticks[0] != want[0] {
		t.Errorf("plan Ticks = %+v, want %+v", pt.Ticks, want)
	}
	if !hasChange(pt.Changes, ChangeAcceptanceCriteriaCarried) {
		t.Errorf("the plan does not report the ticks as carried: %+v", pt.Changes)
	}
	if hasChange(pt.Changes, ChangeAcceptanceCriteriaUnchecked) {
		t.Errorf("the plan reports the ticks as carried and as unchecked at once: %+v", pt.Changes)
	}

	res, err := dst.ApplyImport(ctx, plan)
	if err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}
	landed, err := dst.Get(ctx, res.Filed[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(landed.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Exactly the owner's own evidence: the one box they ticked and no other.
	if n := strings.Count(string(body), "- [x]"); n != 1 {
		t.Errorf("%d boxes arrived ticked, want the 1 the owner ticked:\n%s", n, body)
	}
	if n := strings.Count(string(body), "- [ ]"); n != 1 {
		t.Errorf("%d boxes arrived unticked, want 1:\n%s", n, body)
	}
}
