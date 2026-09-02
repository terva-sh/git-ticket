package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var testActor = Actor{ID: "human:sothr", Name: "Drew Short"}

// newTestStore returns an empty store with the clock pinned to the reference
// instant.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Init(t.TempDir(), InitOptions{Actor: testActor, Now: fixedClock()})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	return s
}

func mustCreate(t *testing.T, s *Store, title string) *Ticket {
	t.Helper()
	res, err := s.Create(context.Background(), CreateOptions{
		Title:       title,
		Description: "Created by a test.",
		Actor:       testActor,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return res.Ticket
}

func mustApply(t *testing.T, s *Store, ref string, m Mutation) *Result {
	t.Helper()
	res, err := s.Apply(context.Background(), ref, m, ApplyOptions{Actor: testActor})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	return res
}

func TestCreateWritesAReadableTicket(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Add token refresh handling")

	want := filepath.Join(s.TicketsDir(), tk.ID+".md")
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("the ticket is not where the ID says it is: %v", err)
	}
	if tk.Status != StatusDraft {
		t.Errorf("status = %s, want %s", tk.Status, StatusDraft)
	}
	if tk.Revision != Revision(data) {
		t.Errorf("the returned revision is not the revision of the bytes on disk")
	}

	// A new ticket has to survive its own round trip like any other.
	reparsed, err := Parse(data)
	if err != nil {
		t.Fatalf("a ticket this tool wrote does not parse: %v", err)
	}
	if string(Render(reparsed)) != string(data) {
		t.Error("a ticket this tool wrote does not round-trip")
	}

	// Four characters of the ULID is enough to find it again.
	got, err := s.Get(context.Background(), tk.ID[len(IDPrefix):len(IDPrefix)+4])
	if err != nil {
		t.Fatalf("get by prefix: %v", err)
	}
	if got.ID != tk.ID {
		t.Errorf("got %s, want %s", got.ID, tk.ID)
	}
}

// TestFieldLevelUpdateTouchesOnlyItsOwnLines is the acceptance criterion in
// plan section 14: a field-level update leaves unrelated fields byte-identical,
// so a diff shows the change and little else.
func TestFieldLevelUpdateTouchesOnlyItsOwnLines(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Cache the provider model list")
	before, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}

	// A later instant, so updated_at genuinely moves.
	s.now = func() time.Time { return referenceInstant.Add(time.Hour) }
	res := mustApply(t, s, tk.ID, SetPriority{Priority: "high"})
	after, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}

	changed := changedLines(string(before), string(after))
	want := map[string]bool{"priority": true, "updated_at": true, "updated_by": true}
	for _, line := range changed {
		key := strings.SplitN(strings.TrimSpace(line), ":", 2)[0]
		if !want[key] {
			t.Errorf("a priority change also rewrote %q", line)
		}
	}
	if len(changed) == 0 {
		t.Error("nothing changed at all")
	}
}

// changedLines returns the lines of b that are not at the same position in a.
func changedLines(a, b string) []string {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	var out []string
	for i := range bl {
		if i >= len(al) || al[i] != bl[i] {
			out = append(out, bl[i])
		}
	}
	return out
}

func TestStaleRevisionIsRefused(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Retry a failed provider request once")
	stale := tk.Revision

	// One writer lands.
	if _, err := s.Apply(context.Background(), tk.ID, SetTitle{Title: "First writer"},
		ApplyOptions{Actor: testActor, IfRevision: stale}); err != nil {
		t.Fatalf("the first write should succeed: %v", err)
	}

	// The second still holds the revision it read before that write.
	_, err := s.Apply(context.Background(), tk.ID, SetTitle{Title: "Second writer"},
		ApplyOptions{Actor: testActor, IfRevision: stale})
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeStaleRevision {
		t.Fatalf("err = %v, want %s", err, CodeStaleRevision)
	}
	if e.Details["expected"] != stale {
		t.Errorf("expected revision = %q, want %q", e.Details["expected"], stale)
	}
	if e.Details["actual"] == "" || e.Details["actual"] == stale {
		t.Errorf("actual revision = %q, which tells the caller nothing", e.Details["actual"])
	}

	// The refusal must not have written anything.
	got, err := s.Get(context.Background(), tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "First writer" {
		t.Errorf("title = %q, want the first writer's", got.Title)
	}
}

// TestConcurrentWritersOneWins is the Phase 1 exit criterion: two writers
// racing the same ticket produce one success and one stale_revision.
func TestConcurrentWritersOneWins(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Rotate the signing key without downtime")
	revision := tk.Revision

	const writers = 4
	var wg sync.WaitGroup
	results := make([]error, writers)
	start := make(chan struct{})

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// A separate Store per writer, so each takes its own file
			// descriptor on the lock rather than sharing one.
			w, err := OpenWith(s.Path(), OpenOptions{Now: fixedClock(), NoRoot: true})
			if err != nil {
				results[i] = err
				return
			}
			<-start
			_, results[i] = w.Apply(context.Background(), tk.ID,
				SetTitle{Title: "writer " + string(rune('A'+i))},
				ApplyOptions{Actor: testActor, IfRevision: revision})
		}(i)
	}
	close(start)
	wg.Wait()

	var won, stale int
	for i, err := range results {
		switch {
		case err == nil:
			won++
		case CodeOf(err) == CodeStaleRevision:
			stale++
		default:
			t.Errorf("writer %d failed with %v, want nil or %s", i, err, CodeStaleRevision)
		}
	}
	if won != 1 {
		t.Errorf("%d writers won, want exactly 1", won)
	}
	if stale != writers-1 {
		t.Errorf("%d writers were told the revision was stale, want %d", stale, writers-1)
	}
}

func TestLockTimeoutWhenHeld(t *testing.T) {
	s := newTestStore(t)

	// A short timeout, so the test does not wait ten seconds to prove the
	// point.
	cfg := s.Config()
	cfg.Lock.Timeout = Duration(20 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(s.Path(), "config.yml"), RenderConfig(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	waiter, err := OpenWith(s.Path(), OpenOptions{Now: fixedClock(), NoRoot: true})
	if err != nil {
		t.Fatal(err)
	}

	held, err := s.lock()
	if err != nil {
		t.Fatalf("taking the lock: %v", err)
	}
	defer held.release()

	_, err = waiter.Create(context.Background(), CreateOptions{Title: "Blocked on the lock", Actor: testActor})
	if CodeOf(err) != CodeLockTimeout {
		t.Errorf("err = %v, want %s", err, CodeLockTimeout)
	}
}

func TestStatusTransitions(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Ship the token refresh work")

	// draft cannot jump to done, and the refusal names where it may go.
	_, err := s.Apply(context.Background(), tk.ID, SetStatus{Status: StatusDone}, ApplyOptions{Actor: testActor})
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeInvalidTransition {
		t.Fatalf("err = %v, want %s", err, CodeInvalidTransition)
	}
	if !strings.Contains(e.Details["permitted"], StatusReady) {
		t.Errorf("permitted = %q, which does not name ready", e.Details["permitted"])
	}

	// archived is not reachable here, because archiving also moves the file.
	_, err = s.Apply(context.Background(), tk.ID, SetStatus{Status: StatusArchived}, ApplyOptions{Actor: testActor})
	if CodeOf(err) != CodeInvalidTransition {
		t.Errorf("status archived = %v, want %s", err, CodeInvalidTransition)
	}

	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})

	// blocked needs a reason.
	_, err = s.Apply(context.Background(), tk.ID, SetStatus{Status: StatusBlocked}, ApplyOptions{Actor: testActor})
	if CodeOf(err) != CodeInvalidField {
		t.Errorf("blocked with no reason = %v, want %s", err, CodeInvalidField)
	}

	res := mustApply(t, s, tk.ID, SetStatus{Status: StatusBlocked, Reason: "the vendor sandbox is down"})
	if res.Ticket.StatusReason == nil || *res.Ticket.StatusReason != "the vendor sandbox is down" {
		t.Errorf("status_reason = %v, want the reason given", res.Ticket.StatusReason)
	}
	if !strings.Contains(res.Ticket.Body.Notes, "the vendor sandbox is down") {
		t.Error("the reason did not reach Notes, so the history is lost on the next transition")
	}

	// Leaving blocked clears the field but not the note.
	res = mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})
	if res.Ticket.StatusReason != nil {
		t.Errorf("status_reason = %v, want it cleared", *res.Ticket.StatusReason)
	}
	if !strings.Contains(res.Ticket.Body.Notes, "the vendor sandbox is down") {
		t.Error("Notes lost the history")
	}
}

func TestArchiveMovesTheFileAndRecordsFromStatus(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Pin the provider SDK to a known version")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, tk.ID, SetStatus{Status: StatusInProgress})
	mustApply(t, s, tk.ID, SetStatus{Status: StatusDone})

	livePath := filepath.Join(s.TicketsDir(), tk.ID+".md")
	res := mustApply(t, s, tk.ID, ArchiveTicket{Reason: "shipped in v1.2"})

	if want := filepath.Join(s.ArchiveDir(), tk.ID+".md"); res.Ticket.Path != want {
		t.Errorf("archived to %s, want %s", res.Ticket.Path, want)
	}
	if _, err := os.Stat(livePath); !os.IsNotExist(err) {
		t.Error("the live file is still there after archiving")
	}
	if len(res.PathsChanged) != 2 {
		t.Errorf("PathsChanged = %v, want both the new and the old path", res.PathsChanged)
	}
	if res.Ticket.Archive == nil || res.Ticket.Archive.FromStatus == nil || *res.Ticket.Archive.FromStatus != StatusDone {
		t.Fatalf("from_status = %+v, want done", res.Ticket.Archive)
	}
	if res.Ticket.Archive.Reason == nil || *res.Ticket.Archive.Reason != "shipped in v1.2" {
		t.Errorf("archive reason = %v, want the reason given", res.Ticket.Archive.Reason)
	}
	if !strings.Contains(res.Ticket.Body.Notes, "shipped in v1.2") {
		t.Error("the archive reason did not reach Notes")
	}
	// Archived out of done, so it still satisfies a dependency.
	if !res.Ticket.SatisfiesDependency() {
		t.Error("a ticket archived out of done should still satisfy a dependency")
	}

	// And back again.
	res = mustApply(t, s, tk.ID, UnarchiveTicket{})
	if res.Ticket.Status != StatusReady {
		t.Errorf("unarchived to %s, want ready", res.Ticket.Status)
	}
	if res.Ticket.Archive != nil {
		t.Error("the archive block outlived the archive")
	}
	// Plan 6.3, and the whole point of writing the reason twice. Unarchiving
	// deletes the block, so the Notes entry is the only thing left that says
	// why the ticket was ever closed out.
	if !strings.Contains(res.Ticket.Body.Notes, "shipped in v1.2") {
		t.Error("unarchiving lost the archive reason")
	}
	if _, err := os.Stat(filepath.Join(s.TicketsDir(), tk.ID+".md")); err != nil {
		t.Errorf("the file did not come back to tickets/: %v", err)
	}
}

func TestClaimConflictAndForce(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Delete the vendored copy of the old client")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})

	agent := Actor{ID: "agent:terva/session-123", Name: "Mieli"}
	if _, err := s.Apply(context.Background(), tk.ID, ClaimTicket{Branch: "feat/drop-vendor"},
		ApplyOptions{Actor: agent}); err != nil {
		t.Fatalf("claim: %v", err)
	}

	// A second actor cannot take a live claim.
	_, err := s.Apply(context.Background(), tk.ID, ClaimTicket{}, ApplyOptions{Actor: testActor})
	if CodeOf(err) != CodeClaimConflict {
		t.Fatalf("err = %v, want %s", err, CodeClaimConflict)
	}

	// Force takes it and leaves a trace.
	res, err := s.Apply(context.Background(), tk.ID, ClaimTicket{Force: true}, ApplyOptions{Actor: testActor})
	if err != nil {
		t.Fatalf("forced claim: %v", err)
	}
	if res.Ticket.Claim.Actor != testActor.ID {
		t.Errorf("claim actor = %s, want %s", res.Ticket.Claim.Actor, testActor.ID)
	}
	if !strings.Contains(res.Ticket.Body.Notes, agent.ID) {
		t.Error("taking a claim by force left no trace in Notes")
	}

	// A draft cannot be claimed at all.
	draft := mustCreate(t, s, "Sketch the offline token story")
	if _, err := s.Apply(context.Background(), draft.ID, ClaimTicket{}, ApplyOptions{Actor: testActor}); err == nil {
		t.Error("a draft should not be claimable")
	}
}

// TestReclaimRenewsRatherThanReplaces covers the renewal rules in plan 6.4.
// Re-claiming a ticket you already hold keeps claimed_at, and keeps an expiry
// that nothing else supplied, because the routine gesture for staying alive on
// a long task must not widen a bounded claim into an unbounded one.
func TestReclaimRenewsRatherThanReplaces(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Rebuild the index from the write-ahead log")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})

	first := mustApply(t, s, tk.ID, ClaimTicket{Branch: "feat/index", ExpiresIn: time.Hour})
	claimedAt, expiresAt := first.Ticket.Claim.ClaimedAt, first.Ticket.Claim.ExpiresAt
	if expiresAt == nil || claimedAt == nil {
		t.Fatal("the first claim recorded no expiry or no claimed_at")
	}

	// Half an hour of work later the claim is still live. The clock has to move
	// here, or a preserved claimed_at and a freshly computed one would be the
	// same instant and the assertion below could never fail.
	s.now = func() time.Time { return referenceInstant.Add(30 * time.Minute) }

	// A renewal that supplies no duration keeps both instants.
	renewed := mustApply(t, s, tk.ID, ClaimTicket{Branch: "feat/index"})
	if got := renewed.Ticket.Claim.ExpiresAt; got == nil || !got.Time.Equal(expiresAt.Time) {
		t.Errorf("expires_at = %v, want it kept at %v", got, expiresAt)
	}
	if got := renewed.Ticket.Claim.ClaimedAt; got == nil || !got.Time.Equal(claimedAt.Time) {
		t.Errorf("claimed_at = %v, want it kept at %v", got, claimedAt)
	}

	// An explicit duration re-anchors the bound from now.
	longer := mustApply(t, s, tk.ID, ClaimTicket{ExpiresIn: 3 * time.Hour})
	want := s.Now().Add(3 * time.Hour)
	if got := longer.Ticket.Claim.ExpiresAt; got == nil || !got.Time.Equal(want) {
		t.Errorf("expires_at = %v, want %v", got, want)
	}
}

// TestReclaimAfterALapseIsAFreshAcquisition covers the other half of 6.4. A
// lapsed claim already grants no exclusivity, so re-acquiring it takes whatever
// the expiry sources give, and here they give nothing. claimed_at still
// survives, because it records when this actor started rather than how long the
// bound had left.
func TestReclaimAfterALapseIsAFreshAcquisition(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Drain the retry queue before the cutover")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusReady})

	first := mustApply(t, s, tk.ID, ClaimTicket{ExpiresIn: time.Hour})
	claimedAt := first.Ticket.Claim.ClaimedAt

	// Walk the clock past the expiry.
	s.now = func() time.Time { return referenceInstant.Add(2 * time.Hour) }

	lapsed := mustApply(t, s, tk.ID, ClaimTicket{})
	if got := lapsed.Ticket.Claim.ExpiresAt; got != nil {
		t.Errorf("expires_at = %v, want nil once the claim has lapsed", got)
	}
	if got := lapsed.Ticket.Claim.ClaimedAt; got == nil || !got.Time.Equal(claimedAt.Time) {
		t.Errorf("claimed_at = %v, want it kept at %v", got, claimedAt)
	}
}

// TestUnreadableTicketIsNotAbsent covers plan section 8. A ticket whose file is
// present but does not parse answers with the parse failure rather than with
// ticket_not_found, because calling a file absent when it is sitting on disk
// sends a reader looking in the wrong place.
func TestUnreadableTicketIsNotAbsent(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Rotate the signing key before it expires")
	other := mustCreate(t, s, "Leave this one readable")
	path := filepath.Join(s.TicketsDir(), tk.ID+".md")

	breakFile := func(old, new string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	breakFile("title:", "title: [unclosed")

	if _, err := s.Get(context.Background(), tk.ID); CodeOf(err) != CodeParseError {
		t.Errorf("Get on a broken ticket = %v, want %s", err, CodeParseError)
	}
	// A prefix reaches it too, so the fix is not limited to a full ID.
	if _, err := s.Get(context.Background(), tk.ID[:24]); CodeOf(err) != CodeParseError {
		t.Errorf("Get by prefix = %v, want %s", err, CodeParseError)
	}
	// Both tickets were created against the fixed clock, so they share their
	// whole ULID timestamp and a short prefix matches both. That has to report
	// the ambiguity. Dropping the broken file from resolution instead would
	// hand back the readable neighbour, and a wrong ticket is worse than none.
	// git ticket list prints prefixes of exactly this length.
	if _, err := s.Get(context.Background(), tk.ID[:13]); CodeOf(err) != CodeAmbiguousID {
		t.Errorf("Get by a shared prefix = %v, want %s", err, CodeAmbiguousID)
	}
	// A mutation says the same thing rather than claiming the ticket vanished.
	_, err := s.Apply(context.Background(), tk.ID, SetStatus{Status: StatusReady}, ApplyOptions{Actor: testActor})
	if CodeOf(err) != CodeParseError {
		t.Errorf("Apply on a broken ticket = %v, want %s", err, CodeParseError)
	}

	// A schema this reader does not know reports as itself. The parse error
	// carries no ID here, because the schema is refused before the rest of the
	// frontmatter is decoded, so this is what exercises the filename fallback.
	breakFile("title: [unclosed", "title:")
	breakFile("schema: 1", "schema: 2")
	if _, err := s.Get(context.Background(), tk.ID); CodeOf(err) != CodeSchemaUnsupported {
		t.Errorf("Get on a schema-2 ticket = %v, want %s", err, CodeSchemaUnsupported)
	}

	// A ticket that genuinely is not there still says so.
	if _, err := s.Get(context.Background(), "TKT-01K3ZZZZZZZZZZZZZZZZZZZZZZ"); CodeOf(err) != CodeTicketNotFound {
		t.Errorf("Get on an absent ticket = %v, want %s", err, CodeTicketNotFound)
	}
	// And the readable ticket beside it is untouched.
	if _, err := s.Get(context.Background(), other.ID); err != nil {
		t.Errorf("the readable ticket broke too: %v", err)
	}
}

// TestCreateWritesTheStoreSchema covers the rule in plan 12.5. A new ticket is
// written at the store's declared schema rather than at this binary's maximum,
// so a newer binary cannot drift an older store upward with no migration run.
//
// The interesting half cannot be tested in tree while SchemaVersion is 1,
// because the store's level and the binary's maximum are then the same number.
// It was verified by building a copy with SchemaVersion = 2 and creating a
// ticket in a schema-1 store. With the rule the file says schema 1 and a
// schema-1 reader sees every ticket. Without it the file says schema 2, and
// that reader drops to two rows of three and one error, losing the newest
// ticket. What is testable here is that Create consults the declaration at all,
// and the fallback when a store declares nothing.
func TestCreateWritesTheStoreSchema(t *testing.T) {
	s := newTestStore(t)

	tk := mustCreate(t, s, "Written at the level the store declares")
	if tk.Schema != s.config.Schema {
		t.Errorf("schema = %d, want the store's declared %d", tk.Schema, s.config.Schema)
	}

	// A store that declares nothing gets this reader's version. Without the
	// fallback the ticket would carry schema 0, which renders as a level no
	// reader claims and parses back as something else again.
	s.config.Schema = 0
	zero := mustCreate(t, s, "Written where the store declares nothing")
	if zero.Schema != SchemaVersion {
		t.Errorf("schema = %d with nothing declared, want %d", zero.Schema, SchemaVersion)
	}
}

func TestChecklistOperations(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Measure cold start before optimizing it")

	mustApply(t, s, tk.ID, AddChecklistItem{Section: AcceptanceCriteria, Text: "Measured on an empty cache"})
	mustApply(t, s, tk.ID, AddChecklistItem{Section: AcceptanceCriteria, Text: "Written down in the runbook"})
	res := mustApply(t, s, tk.ID, SetChecklistItem{Section: AcceptanceCriteria, Index: 2, Checked: true})

	items := Checklist(res.Ticket.Body.AcceptanceCriteria)
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Checked || !items[1].Checked {
		t.Errorf("items = %+v, want only the second checked", items)
	}
	if !strings.Contains(res.Ticket.Body.AcceptanceCriteria, "- [x] Written down in the runbook") {
		t.Errorf("rendered checklist:\n%s", res.Ticket.Body.AcceptanceCriteria)
	}

	// An index past the end says how many there are rather than failing mutely.
	_, err := s.Apply(context.Background(), tk.ID,
		SetChecklistItem{Section: AcceptanceCriteria, Index: 9, Checked: true}, ApplyOptions{Actor: testActor})
	if CodeOf(err) != CodeInvalidField {
		t.Errorf("err = %v, want %s", err, CodeInvalidField)
	}
}

// TestImplementationPlanIsWritable covers the gap that plan 5.2 defined the
// section and nothing could fill it: it parsed, rendered, and was searched,
// while no mutation reached it.
func TestImplementationPlanIsWritable(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Rotate the signing key without downtime")

	if tk.Body.ImplementationPlan != "" {
		t.Fatalf("a new ticket starts with a plan: %q", tk.Body.ImplementationPlan)
	}

	res := mustApply(t, s, tk.ID, SetImplementationPlan{Text: "1. Read the verifier\n2. Teach it both keys"})
	if res.Ticket.Body.ImplementationPlan != "1. Read the verifier\n2. Teach it both keys" {
		t.Errorf("plan = %q", res.Ticket.Body.ImplementationPlan)
	}

	// The section reaches the file, not just the value in memory.
	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.Contains(string(data), "## Implementation plan\n\n1. Read the verifier") {
		t.Errorf("the plan did not render into the file:\n%s", data)
	}

	// It replaces rather than appends, which is the SetSummary rule: a plan is
	// one statement of how the work will go, and the log is Notes.
	res = mustApply(t, s, tk.ID, SetImplementationPlan{Text: "1. Actually, roll the key first"})
	if got := res.Ticket.Body.ImplementationPlan; got != "1. Actually, roll the key first" {
		t.Errorf("a second write should replace the first, got %q", got)
	}
}

// TestCreateTrimsBodySectionsItSeeds pins the reason Create trims. The renderer
// writes section text verbatim and the parser strips the blank lines around it,
// so text padded on the way in renders bytes that do not survive the round trip
// plan 5.3 requires. Every Set* mutation on a body section already trimmed;
// Create did not, and create --description could reach it.
func TestCreateTrimsBodySectionsItSeeds(t *testing.T) {
	s := newTestStore(t)
	res, err := s.Create(context.Background(), CreateOptions{
		Title:              "Seeded with padded prose",
		Description:        "\n\nThe description.\n\n",
		ImplementationPlan: "\n1. The plan.\n\n",
		Actor:              testActor,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if res.Ticket.Body.Description != "The description." {
		t.Errorf("description = %q", res.Ticket.Body.Description)
	}
	if res.Ticket.Body.ImplementationPlan != "1. The plan." {
		t.Errorf("plan = %q", res.Ticket.Body.ImplementationPlan)
	}

	// The bytes on disk are the ones a second render would produce.
	data, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	again, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := string(Render(again)); got != string(data) {
		t.Errorf("a seeded ticket does not round trip\n%s", diffLines(string(data), got))
	}
}

func TestDependencyMutationsRefuseTheImpossible(t *testing.T) {
	s := newTestStore(t)
	a := mustCreate(t, s, "Cycle member A")
	b := mustCreate(t, s, "Cycle member B")

	if _, err := s.Apply(context.Background(), a.ID, AddDependency{ID: a.ID}, ApplyOptions{Actor: testActor}); CodeOf(err) != CodeDependencyCycle {
		t.Errorf("self-dependency = %v, want %s", err, CodeDependencyCycle)
	}
	if _, err := s.Apply(context.Background(), a.ID, AddDependency{ID: "TKT-01K3ZZZZZZZZZZZZZZZZZZZZZZ"}, ApplyOptions{Actor: testActor}); CodeOf(err) != CodeDependencyMissing {
		t.Errorf("missing dependency = %v, want %s", err, CodeDependencyMissing)
	}

	res := mustApply(t, s, a.ID, AddDependency{ID: b.ID})
	if len(res.Ticket.Dependencies) != 1 || res.Ticket.Dependencies[0] != b.ID {
		t.Errorf("dependencies = %v, want just %s", res.Ticket.Dependencies, b.ID)
	}
	// Adding it twice is not two dependencies.
	res = mustApply(t, s, a.ID, AddDependency{ID: b.ID})
	if len(res.Ticket.Dependencies) != 1 {
		t.Errorf("dependencies = %v after a repeat add", res.Ticket.Dependencies)
	}
}

// TestApplyKeepsUnknownFields is plan 5.4 through the write path: a field a
// newer reader added survives a mutation by this one.
func TestApplyKeepsUnknownFields(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Written by an older reader")

	// Splice in a field this version does not define, the way a newer writer
	// would.
	data, err := os.ReadFile(tk.Path)
	if err != nil {
		t.Fatal(err)
	}
	patched := strings.Replace(string(data), "extensions: {}\n", "extensions: {}\nseverity: high\n", 1)
	if err := os.WriteFile(tk.Path, []byte(patched), 0o644); err != nil {
		t.Fatal(err)
	}

	res := mustApply(t, s, tk.ID, SetPriority{Priority: "urgent"})
	after, err := os.ReadFile(res.Ticket.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "severity: high") {
		t.Errorf("the unknown field did not survive the write:\n%s", after)
	}
}

// TestCheckPassesOnAStoreThisToolWrote closes the loop: everything the
// mutations produce is something check accepts.
func TestCheckPassesOnAStoreThisToolWrote(t *testing.T) {
	s := newTestStore(t)
	a := mustCreate(t, s, "First")
	b := mustCreate(t, s, "Second")
	mustApply(t, s, a.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, a.ID, AddDependency{ID: b.ID})
	mustApply(t, s, b.ID, SetStatus{Status: StatusReady})
	mustApply(t, s, b.ID, SetStatus{Status: StatusInProgress})
	mustApply(t, s, b.ID, ClaimTicket{})
	mustApply(t, s, b.ID, SetStatus{Status: StatusDone})
	mustApply(t, s, b.ID, ArchiveTicket{Reason: "done and filed"})

	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		for _, f := range report.Errors {
			t.Errorf("check error: %s %s %s", f.Code, f.File, f.Message)
		}
	}
	for _, f := range report.Warnings {
		t.Errorf("check warning: %s %s %s", f.Code, f.File, f.Message)
	}
}
