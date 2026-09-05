package ticket

import (
	"context"
	"strings"
	"testing"
	"time"
)

// These tests hold Create to plan 6.2.1: a ticket may arrive done or
// archived, backdated, and nothing else.

func TestCreateArrivesDone(t *testing.T) {
	s := newTestStore(t)
	res, err := s.Create(context.Background(), CreateOptions{
		Title:  "Shipped in the old system",
		Status: StatusDone,
		Actor:  testActor,
	})
	if err != nil {
		t.Fatalf("create --status done: %v", err)
	}
	tk := res.Ticket
	if tk.Status != StatusDone {
		t.Fatalf("status = %q, want done", tk.Status)
	}
	if !strings.Contains(tk.Path, "/done/") {
		t.Fatalf("path = %q, want it under done/", tk.Path)
	}

	// The whole point of done over archived: it satisfies a dependency.
	dep, err := s.Create(context.Background(), CreateOptions{
		Title:        "Builds on the backport",
		Dependencies: []string{tk.ID},
		Actor:        testActor,
	})
	if err != nil {
		t.Fatalf("dependent create: %v", err)
	}
	if _, err := s.Apply(context.Background(), dep.Ticket.ID, SetStatus{Status: StatusReady}, ApplyOptions{Actor: testActor}); err != nil {
		t.Fatalf("promote dependent: %v", err)
	}
	ready, err := s.Readiness(context.Background())
	if err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if r := ready[dep.Ticket.ID]; !r.Ready {
		t.Fatalf("dependent readiness = %+v, want ready: a created-done ticket must satisfy its dependents", r)
	}
}

func TestCreateArrivesArchived(t *testing.T) {
	s := newTestStore(t)
	res, err := s.Create(context.Background(), CreateOptions{
		Title:  "Abandoned in the old system",
		Status: StatusArchived,
		Reason: "superseded before migration",
		Actor:  testActor,
	})
	if err != nil {
		t.Fatalf("create --status archived: %v", err)
	}
	tk := res.Ticket
	if tk.Status != StatusArchived || !strings.Contains(tk.Path, "/archive/") {
		t.Fatalf("status %q at %q, want archived under archive/", tk.Status, tk.Path)
	}
	if tk.Archive == nil || tk.Archive.FromStatus == nil || *tk.Archive.FromStatus != StatusDraft {
		t.Fatalf("archive block = %+v, want from_status draft", tk.Archive)
	}
	if tk.Archive.Reason == nil || *tk.Archive.Reason != "superseded before migration" {
		t.Fatalf("archive reason = %+v", tk.Archive.Reason)
	}
	if !strings.Contains(tk.Body.Notes, "superseded before migration") {
		t.Fatalf("the reason did not land in Notes:\n%s", tk.Body.Notes)
	}

	// from_status draft is what keeps it from satisfying anybody, per 6.3.
	dep, err := s.Create(context.Background(), CreateOptions{
		Title:        "Waits on the abandoned work",
		Dependencies: []string{tk.ID},
		Actor:        testActor,
	})
	if err != nil {
		t.Fatalf("dependent create: %v", err)
	}
	if _, err := s.Apply(context.Background(), dep.Ticket.ID, SetStatus{Status: StatusReady}, ApplyOptions{Actor: testActor}); err != nil {
		t.Fatalf("promote dependent: %v", err)
	}
	ready, err := s.Readiness(context.Background())
	if err != nil {
		t.Fatalf("readiness: %v", err)
	}
	if r := ready[dep.Ticket.ID]; r.Ready || !r.Blocked {
		t.Fatalf("dependent readiness = %+v, want blocked: an archived-from-draft backport satisfies nothing", r)
	}
}

func TestCreateRefusesOtherStatuses(t *testing.T) {
	s := newTestStore(t)
	for _, status := range []string{StatusReady, StatusInProgress, StatusBlocked, StatusReview, "nonsense"} {
		_, err := s.Create(context.Background(), CreateOptions{Title: "No shortcut", Status: status, Actor: testActor})
		if CodeOf(err) != CodeInvalidField {
			t.Errorf("create --status %s = %v, want %s", status, err, CodeInvalidField)
		}
	}
}

func TestCreateReasonBelongsToArchivedOnly(t *testing.T) {
	s := newTestStore(t)
	for _, status := range []string{"", StatusDone} {
		_, err := s.Create(context.Background(), CreateOptions{Title: "No place for it", Status: status, Reason: "because", Actor: testActor})
		if CodeOf(err) != CodeInvalidField {
			t.Errorf("reason with status %q = %v, want %s", status, err, CodeInvalidField)
		}
	}
}

// TestCreateBackdates holds --created to its purpose: the ULID takes its
// time part from the instant, so backported history sorts chronologically,
// and created_at agrees.
func TestCreateBackdates(t *testing.T) {
	s := newTestStore(t)
	past := time.Date(2020, 3, 14, 9, 26, 53, 0, time.UTC)
	old, err := s.Create(context.Background(), CreateOptions{
		Title:   "Ancient history",
		Status:  StatusDone,
		Created: past,
		Actor:   testActor,
	})
	if err != nil {
		t.Fatalf("backdated create: %v", err)
	}
	if got := old.Ticket.CreatedAt.String(); !strings.HasPrefix(got, "2020-03-14") {
		t.Fatalf("created_at = %q, want the backdated day", got)
	}
	if got := old.Ticket.UpdatedAt.String(); !strings.HasPrefix(got, "2020-03-14") {
		t.Fatalf("updated_at = %q, want the backdated day", got)
	}
	recent := mustCreate(t, s, "Filed at the reference instant")
	if !(old.Ticket.ID < recent.ID) {
		t.Fatalf("backdated ID %s does not sort before %s", old.Ticket.ID, recent.ID)
	}
}

func TestCreateRefusesTheFuture(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Create(context.Background(), CreateOptions{
		Title:   "From tomorrow",
		Created: fixedClock()().Add(time.Hour),
		Actor:   testActor,
	})
	if CodeOf(err) != CodeInvalidField {
		t.Fatalf("future --created = %v, want %s", err, CodeInvalidField)
	}
}

// TestArrivedStoreIsClean: a store holding a created-done and a
// created-archived ticket raises no finding, because the files sit where
// their status says and the archive block is complete.
func TestArrivedStoreIsClean(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create(context.Background(), CreateOptions{Title: "Arrived done", Status: StatusDone, Actor: testActor}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), CreateOptions{Title: "Arrived archived", Status: StatusArchived, Actor: testActor}); err != nil {
		t.Fatal(err)
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(report.Errors) != 0 || len(report.Warnings) != 0 {
		t.Fatalf("findings = %+v / %+v, want none", report.Errors, report.Warnings)
	}
}
