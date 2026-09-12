package ticket

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestExportReturnsBytesAndTouchesNoDirectory is the criterion in its own
// words: the artifact comes back as bytes, and where it lands is the caller's
// decision. Nothing in this test names a directory, which is the point.
func TestExportReturnsBytesAndTouchesNoDirectory(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	made, err := s.Create(ctx, CreateOptions{Title: "Send me somewhere", Actor: testActor})
	if err != nil {
		t.Fatal(err)
	}

	when := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	art, err := s.Export(ctx, ExportOptions{
		IDs:  []string{made.Ticket.ID},
		From: "A Sender <a@example.com>",
		Now:  when,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if len(art.Tickets) != 1 || art.Tickets[0].ID != made.Ticket.ID {
		t.Fatalf("Export carried %d tickets, want the one asked for", len(art.Tickets))
	}
	for _, part := range []struct {
		name string
		data []byte
	}{{"cover", art.Cover}, {"patch", art.Patch}} {
		if len(part.data) == 0 {
			t.Errorf("the %s is empty", part.name)
			continue
		}
		if !strings.HasPrefix(string(part.data), MboxFromLine+"\n") {
			t.Errorf("the %s does not open where git's mailsplit looks", part.name)
		}
		if !strings.Contains(string(part.data), "From: A Sender <a@example.com>") {
			t.Errorf("the %s does not carry the sender", part.name)
		}
	}

	// The cover is numbered 0 of 1 and is not itself a patch, which is what
	// leaves room for code composed in from number 2.
	if !strings.Contains(string(art.Cover), "Subject: [PATCH 0/1]") {
		t.Errorf("the cover is not numbered 0/1:\n%s", art.Cover)
	}
	if !strings.Contains(string(art.Patch), "new file mode 100644") {
		t.Errorf("the patch adds no file:\n%s", art.Patch)
	}
}

// TestExportRoundTripsThroughPlanImport is the pair closing on itself: what
// Export writes, PlanImport reads, with no directory and no command in between.
// A host can now do the whole interchange through the library.
func TestExportRoundTripsThroughPlanImport(t *testing.T) {
	ctx := context.Background()

	src := newTestStore(t)
	made, err := src.Create(ctx, CreateOptions{
		Title:              "Cross the gap",
		AcceptanceCriteria: []string{"one"},
		Actor:              testActor,
	})
	if err != nil {
		t.Fatal(err)
	}
	art, err := src.Export(ctx, ExportOptions{
		IDs:  []string{made.Ticket.ID},
		From: "A Sender <a@example.com>",
		Now:  time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	dst := newTestStore(t)
	plan, err := dst.PlanImport(ctx, ImportOptions{Patch: string(art.Patch), Actor: testActor})
	if err != nil {
		t.Fatalf("PlanImport over Export's own bytes: %v", err)
	}
	if len(plan.Tickets) != 1 {
		t.Fatalf("planned %d tickets, want 1", len(plan.Tickets))
	}
	if plan.Tickets[0].Incoming.Title != "Cross the gap" {
		t.Errorf("title = %q", plan.Tickets[0].Incoming.Title)
	}

	res, err := dst.ApplyImport(ctx, plan)
	if err != nil {
		t.Fatalf("ApplyImport: %v", err)
	}
	if res.Filed[0].FromID != made.Ticket.ID {
		t.Errorf("FromID = %s, want %s", res.Filed[0].FromID, made.Ticket.ID)
	}
}

// TestExportNeedsATicket keeps the empty call an error rather than an artifact
// carrying nothing, which would apply cleanly and mean nothing.
func TestExportNeedsATicket(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Export(context.Background(), ExportOptions{}); err == nil {
		t.Fatal("an export of no tickets succeeded")
	}
}
