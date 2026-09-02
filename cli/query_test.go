package cli

import (
	"strings"
	"testing"
)

func bodyOf(t *testing.T, dir, id string) map[string]any {
	t.Helper()
	return showTicket(t, dir, id)["body"].(map[string]any)
}

// TestSearchReadsTitleAndBody is plan section 8: the title, the body sections,
// and the references. The frontmatter beyond the title is deliberately left
// out, so searching for a type or a status does not return everything.
func TestSearchReadsTitleAndBody(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "Rotate the signing key")
	b := makeTicket(t, dir, "Unrelated chore")

	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "signing")); len(ids) != 1 || ids[0] != a {
		t.Errorf("search = %v, want [%s]", ids, a)
	}
	// Case-insensitive by default.
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "SIGNING")); len(ids) != 1 {
		t.Errorf("search should ignore case, got %v", ids)
	}

	// A body section counts, not just the title.
	runCLI(t, dir, nil, "note", b, "mentions the signing key too", "--actor", "human:sothr")
	wantIDs(t, idsOf(t, runCLI(t, dir, nil, "--json", "search", "signing")), a, b)

	// A reference counts too.
	c := makeTicket(t, dir, "Third")
	runCLI(t, dir, nil, "link", c, "--ref", "proposal:telemetry", "--actor", "human:sothr")
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "telemetry")); len(ids) != 1 || ids[0] != c {
		t.Errorf("search should read references, got %v", ids)
	}

	// The status is frontmatter, so it is not searchable. Every ticket here
	// is a draft, and none of them says so in its title or body.
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "draft")); len(ids) != 0 {
		t.Errorf("search matched frontmatter: %v", ids)
	}
}

func TestSearchRegex(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "Rotate the signing key")
	makeTicket(t, dir, "Unrelated chore")

	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "--regex", "^Rotate")); len(ids) != 1 || ids[0] != a {
		t.Errorf("regex search = %v, want [%s]", ids, a)
	}
	// Anchored, so the same text as a substring finds nothing.
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "search", "--regex", "^signing")); len(ids) != 0 {
		t.Errorf("an anchored pattern matched anyway: %v", ids)
	}

	got := runCLI(t, dir, nil, "--json", "search", "--regex", "[")
	if got.code != exitError {
		t.Fatal("a malformed pattern should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}
}

func TestSearchUsage(t *testing.T) {
	dir := newStore(t)
	makeTicket(t, dir, "A")

	// An empty query is refused by the library rather than matching all.
	got := runCLI(t, dir, nil, "--json", "search", "")
	if got.code != exitError {
		t.Fatal("an empty query should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}

	if got := runCLI(t, dir, nil, "--json", "search"); errCode(t, got) != codeUsage {
		t.Error("search with no query should be a usage error")
	}
	if got := runCLI(t, dir, nil, "--json", "search", "one", "two"); errCode(t, got) != codeUsage {
		t.Error("search takes a single query")
	}
}

// TestReadyIsWhatCouldBeStarted covers all three conditions of section 8:
// status ready, no live claim, and every dependency satisfied.
func TestReadyIsWhatCouldBeStarted(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A, ready to go")
	b := makeTicket(t, dir, "B, still a draft")
	c := makeTicket(t, dir, "C, waiting on B")

	// A draft is not ready, whatever else is true of it.
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); len(ids) != 0 {
		t.Errorf("a store of drafts reported %v as ready", ids)
	}

	runCLI(t, dir, nil, "status", a, "ready", "--actor", "human:sothr")
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); len(ids) != 1 || ids[0] != a {
		t.Errorf("ready = %v, want [%s]", ids, a)
	}

	// A live claim takes it out: somebody is already on it.
	runCLI(t, dir, nil, "claim", a, "--actor", "human:sothr")
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); len(ids) != 0 {
		t.Errorf("a claimed ticket is still listed as ready: %v", ids)
	}
	runCLI(t, dir, nil, "release", a, "--actor", "human:sothr")

	// An unsatisfied dependency takes it out. B is a draft, so C waits.
	runCLI(t, dir, nil, "link", c, "--depends-on", b, "--actor", "human:sothr")
	runCLI(t, dir, nil, "status", c, "ready", "--actor", "human:sothr")
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); len(ids) != 1 || ids[0] != a {
		t.Errorf("ready = %v, want only [%s]: C waits on a draft", ids, a)
	}

	// Finishing B satisfies it, per 6.3.
	for _, s := range []string{"ready", "in-progress", "done"} {
		runCLI(t, dir, nil, "status", b, s, "--actor", "human:sothr")
	}
	wantIDs(t, idsOf(t, runCLI(t, dir, nil, "--json", "ready")), a, c)

	if got := runCLI(t, dir, nil, "--json", "ready", "extra"); errCode(t, got) != codeUsage {
		t.Error("ready takes no arguments")
	}
}

// TestNoteAndCommentAppend keeps a log; each entry is stamped and the earlier
// ones stay.
func TestNoteAndCommentAppend(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	runCLI(t, dir, nil, "note", a, "The vendor confirmed the window.", "--actor", "human:sothr")
	runCLI(t, dir, nil, "note", a, "The window moved.", "--actor", "agent:terva/s1")
	notes := bodyOf(t, dir, a)["notes"].(string)
	for _, want := range []string{"The vendor confirmed the window.", "The window moved.",
		"human:sothr", "agent:terva/s1"} {
		if !strings.Contains(notes, want) {
			t.Errorf("Notes is missing %q:\n%s", want, notes)
		}
	}

	runCLI(t, dir, nil, "comment", a, "Second pair of eyes wanted.", "--actor", "human:sothr")
	runCLI(t, dir, nil, "comment", a, "Looks right to me.", "--actor", "agent:terva/s1")
	comments := bodyOf(t, dir, a)["comments"].(string)
	if !strings.Contains(comments, "Second pair of eyes wanted.") ||
		!strings.Contains(comments, "Looks right to me.") {
		t.Errorf("Comments did not keep both entries:\n%s", comments)
	}
	// The two sections stay separate.
	if strings.Contains(comments, "The vendor confirmed") {
		t.Error("a note leaked into Comments")
	}
}

// TestSummaryReplaces is the decision recorded in plan section 9. A summary is
// one statement of where the ticket landed, and Notes is already the log.
func TestSummaryReplaces(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	runCLI(t, dir, nil, "summary", a, "Rotated with no downtime.", "--actor", "human:sothr")
	if got := bodyOf(t, dir, a)["summary"]; got != "Rotated with no downtime." {
		t.Errorf("summary = %v", got)
	}

	runCLI(t, dir, nil, "summary", a, "Superseded statement.", "--actor", "human:sothr")
	summary := bodyOf(t, dir, a)["summary"].(string)
	if summary != "Superseded statement." {
		t.Errorf("summary = %q, want the second one alone", summary)
	}
	if strings.Contains(summary, "Rotated with no downtime.") {
		t.Error("summary appended rather than replaced")
	}
}

// TestPlanReplaces is the same decision as TestSummaryReplaces, applied to the
// section plan 5.2 defined and nothing could write until now. A plan is one
// statement of how the work will go, so rewriting it means the first one to be
// gone rather than read alongside the second.
func TestPlanReplaces(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	runCLI(t, dir, nil, "plan", a, "1. Read the verifier", "--actor", "human:sothr")
	if got := bodyOf(t, dir, a)["implementationPlan"]; got != "1. Read the verifier" {
		t.Errorf("implementationPlan = %v", got)
	}

	runCLI(t, dir, nil, "plan", a, "1. Roll the key first", "--actor", "human:sothr")
	plan := bodyOf(t, dir, a)["implementationPlan"].(string)
	if plan != "1. Roll the key first" {
		t.Errorf("implementationPlan = %q, want the second one alone", plan)
	}
	if strings.Contains(plan, "Read the verifier") {
		t.Error("the plan appended rather than replaced")
	}
}

// TestCreateSeedsAPlan covers the other half of the gap: a ticket filed with
// its plan already written needs no second call.
func TestCreateSeedsAPlan(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--plan", "1. Measure before optimizing"))

	if got := bodyOf(t, dir, id)["implementationPlan"]; got != "1. Measure before optimizing" {
		t.Errorf("implementationPlan = %v", got)
	}
}

func TestTextEntryUsage(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	for _, name := range []string{"note", "comment", "plan", "summary"} {
		// The text is required.
		if got := runCLI(t, dir, nil, "--json", name, a, "--actor", "human:sothr"); errCode(t, got) != codeUsage {
			t.Errorf("%s with no text should be a usage error", name)
		}
		// So is the ID.
		if got := runCLI(t, dir, nil, "--json", name); errCode(t, got) != codeUsage {
			t.Errorf("%s with no arguments should be a usage error", name)
		}
	}

	// Text that starts with a dash goes after a bare --, with the flags
	// before it, since -- ends the flags for everything after.
	got := runCLI(t, dir, nil, "note", a, "--actor", "human:sothr", "--", "--force was needed here")
	if got.code != exitOK {
		t.Fatalf("a note after --: %s", got.stderr)
	}
	if notes := bodyOf(t, dir, a)["notes"].(string); !strings.Contains(notes, "--force was needed here") {
		t.Errorf("the note did not land:\n%s", notes)
	}

	// Text that is only whitespace is empty once trimmed.
	got = runCLI(t, dir, nil, "--json", "note", a, "   ", "--actor", "human:sothr")
	if got.code != exitError {
		t.Error("an empty note should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}
}

// TestFilesMatchesBothReferenceForms covers what the library looks at: a
// file:PATH reference, and a typed reference carrying that path.
func TestFilesMatchesBothReferenceForms(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A, with a file reference")
	b := makeTicket(t, dir, "B, with a typed reference and a path")
	makeTicket(t, dir, "C, referencing nothing")

	runCLI(t, dir, nil, "link", a, "--ref", "file:docs/plan.md", "--actor", "human:sothr")
	runCLI(t, dir, nil, "link", b, "--ref", "proposal:x", "--path", "docs/plan.md", "--actor", "human:sothr")

	wantIDs(t, idsOf(t, runCLI(t, dir, nil, "--json", "files", "docs/plan.md")), a, b)

	// The match is on the whole path, not a prefix of it.
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "files", "docs")); len(ids) != 0 {
		t.Errorf("files matched a partial path: %v", ids)
	}

	if got := runCLI(t, dir, nil, "--json", "files"); errCode(t, got) != codeUsage {
		t.Error("files with no path should be a usage error")
	}
	got := runCLI(t, dir, nil, "--json", "files", "")
	if got.code != exitError {
		t.Error("an empty path should be refused")
	}
}

// TestReadsEmitTicketLists keeps the three new reads inside the contract, and
// checks each says something useful when it finds nothing.
func TestReadsEmitTicketLists(t *testing.T) {
	dir := newStore(t)
	makeTicket(t, dir, "A")

	for _, c := range []struct {
		args  []string
		empty string
	}{
		{[]string{"search", "frobnicate"}, "Nothing matches."},
		{[]string{"ready"}, "Nothing is ready to pick up."},
		{[]string{"files", "nothing/here.md"}, "No ticket recorded a reference to that path."},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json"}, c.args...)...)
		if got.code != exitOK {
			t.Fatalf("%v: %s", c.args, got.stderr)
		}
		envelope := decode(t, got.stdout)
		if envelope["kind"] != "ticket-list" {
			t.Errorf("%v kind = %v, want ticket-list", c.args, envelope["kind"])
		}
		if tickets, ok := envelope["tickets"].([]any); !ok || len(tickets) != 0 {
			t.Errorf("%v tickets = %v, want an empty array", c.args, envelope["tickets"])
		}

		// Human mode says which question found nothing, since a blank
		// screen does not.
		human := runCLI(t, dir, nil, c.args...)
		if !strings.Contains(human.stdout, c.empty) {
			t.Errorf("%v printed %q, want %q", c.args, human.stdout, c.empty)
		}
	}
}

// TestTextEntriesEmitMutationResults keeps every runTextEntry write in the
// contract, and honours the precondition every other write does.
func TestTextEntriesEmitMutationResults(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")
	stale, _ := showTicket(t, dir, a)["revision"].(string)

	for _, name := range []string{"note", "comment", "plan", "summary"} {
		got := runCLI(t, dir, nil, "--json", name, a, "some text", "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("%s: %s", name, got.stderr)
		}
		envelope := decode(t, got.stdout)
		if envelope["kind"] != "mutation-result" {
			t.Errorf("%s kind = %v, want mutation-result", name, envelope["kind"])
		}
		if tk, ok := envelope["ticket"].(map[string]any); !ok || tk["id"] != a {
			t.Errorf("%s ticket = %v", name, envelope["ticket"])
		}
	}

	// The first write above moved the ticket on, so the revision read before
	// them is stale for all of them.
	for _, name := range []string{"note", "comment", "plan", "summary"} {
		got := runCLI(t, dir, nil, "--json", name, a, "more text",
			"--if-revision", stale, "--actor", "human:sothr")
		if got.code != exitError {
			t.Errorf("%s ignored --if-revision", name)
			continue
		}
		if code := errCode(t, got); code != "stale_revision" {
			t.Errorf("%s: code = %v, want stale_revision", name, code)
		}
	}
}

// TestHelpCoversTheAdvisoryCaveat is plan section 8, which says the help text
// has to state that files is advisory rather than derived from Git history.
func TestHelpCoversTheAdvisoryCaveat(t *testing.T) {
	help := runCLI(t, t.TempDir(), nil, "help")
	if help.code != exitOK {
		t.Fatalf("help exited %d", help.code)
	}
	if !strings.Contains(help.stdout, "advisory") {
		t.Errorf("the help text does not say files is advisory:\n%s", help.stdout)
	}
	for _, name := range []string{"search", "ready", "note", "comment", "plan", "summary", "files"} {
		if !strings.Contains(help.stdout, name) {
			t.Errorf("help does not mention %q", name)
		}
	}
}
