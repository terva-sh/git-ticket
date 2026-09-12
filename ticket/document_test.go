package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDocument puts an authored ticket somewhere outside the store, which is
// the whole point of 4.3: a document is a file a person or an agent wrote, and
// it does not live in `.tickets/templates/`.
func writeDocument(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ticket.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// wholeTicket is a document of the shape a model produces when asked for a
// ticket: frontmatter, and every section 5.2 names.
const wholeTicket = `---
title: Add token refresh handling
type: bug
priority: high
labels:
  - regression
assignees:
  - human:sothr
milestone: v1.0
---

## Description

The daemon drops the refresh token on a 401 and never retries.

## Acceptance criteria

- [ ] A 401 refreshes once and retries the original request
- [ ] A second 401 gives up and reports the original status

## Definition of done

- [ ] The retry is covered by a test that fails without it

## Implementation plan

1. Catch the 401 in the transport
2. Refresh, then replay the request once
`

// TestReadDocumentSeedsTheWholeTicket is the first criterion at the library
// level: everything a person wrote comes back, sections and checklists intact.
func TestReadDocumentSeedsTheWholeTicket(t *testing.T) {
	doc, err := ReadDocument(writeDocument(t, wholeTicket))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}

	if doc.Title != "Add token refresh handling" {
		t.Errorf("Title = %q", doc.Title)
	}
	if doc.Seed.Type != "bug" || doc.Seed.Priority != "high" {
		t.Errorf("Type = %q, Priority = %q", doc.Seed.Type, doc.Seed.Priority)
	}
	if len(doc.Seed.Labels) != 1 || doc.Seed.Labels[0] != "regression" {
		t.Errorf("Labels = %v", doc.Seed.Labels)
	}
	if len(doc.Seed.Assignees) != 1 || doc.Seed.Assignees[0] != "human:sothr" {
		t.Errorf("Assignees = %v", doc.Seed.Assignees)
	}
	if doc.Seed.Milestone == nil || *doc.Seed.Milestone != "v1.0" {
		t.Errorf("Milestone = %v", doc.Seed.Milestone)
	}
	for _, part := range []struct {
		name, got, want string
	}{
		{"description", doc.Seed.Description, "drops the refresh token"},
		{"acceptance criteria", doc.Seed.AcceptanceCriteria, "refreshes once and retries"},
		{"definition of done", doc.Seed.DefinitionOfDone, "fails without it"},
		{"implementation plan", doc.Seed.ImplementationPlan, "replay the request once"},
	} {
		if !strings.Contains(part.got, part.want) {
			t.Errorf("the %s did not survive: %q", part.name, part.got)
		}
	}
	if len(doc.Ignored) != 0 {
		t.Errorf("Ignored = %v, want none for a document stating no lifecycle", doc.Ignored)
	}
}

// TestReadDocumentIgnoresWhatItDoesNotKnow is 4.2's leniency, which 4.3 keeps
// deliberately. An unknown key is ignored rather than refused, because that is
// what lets a document be made by copying a real ticket or by a model that
// invented a field.
func TestReadDocumentIgnoresWhatItDoesNotKnow(t *testing.T) {
	doc, err := ReadDocument(writeDocument(t, `---
title: Still files
severity: catastrophic
sprint: 14
reviewers:
  - somebody
---

## Description

Filed anyway.
`))
	if err != nil {
		t.Fatalf("an unknown key refused the document: %v", err)
	}
	if doc.Title != "Still files" {
		t.Errorf("Title = %q", doc.Title)
	}
	if !strings.Contains(doc.Seed.Description, "Filed anyway") {
		t.Errorf("Description = %q", doc.Seed.Description)
	}
	if len(doc.Ignored) != 0 {
		t.Errorf("Ignored = %v, want none: an unknown key is not a lifecycle key", doc.Ignored)
	}
}

// TestReadDocumentNamesTheLifecycleKeys is the reporting half of 4.3. The keys
// are not honoured and they are not dropped in silence either.
func TestReadDocumentNamesTheLifecycleKeys(t *testing.T) {
	doc, err := ReadDocument(writeDocument(t, `---
id: TKT-01K3ZZ67Q0PT427VFD1F4WFWSH
title: Copied out of a real store
status: done
created_at: 2024-01-01T00:00:00Z
updated_by: human:someone
---

## Description

Copied, which is the ordinary way to write one of these.
`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	want := []string{"id", "status", "created_at", "updated_by"}
	if strings.Join(doc.Ignored, ",") != strings.Join(want, ",") {
		t.Errorf("Ignored = %v, want %v in 5.1's order", doc.Ignored, want)
	}
}

// TestReadDocumentSaysNothingAboutANullKey keeps the warning worth reading. A
// ticket rendered by this tool carries `status_reason: null` and `claim: null`
// on the ordinary path per 5.3, and a warning that fires on every copied ticket
// would teach a reader to skip the one that matters.
func TestReadDocumentSaysNothingAboutANullKey(t *testing.T) {
	doc, err := ReadDocument(writeDocument(t, `---
title: Copied, with the nulls a render leaves behind
status_reason: null
claim: null
archive: null
---

## Description

Nothing here was stated.
`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	if len(doc.Ignored) != 0 {
		t.Errorf("Ignored = %v, want none: a null states nothing to drop", doc.Ignored)
	}
}

// TestReadDocumentRefusesAPathThatIsNotThere matches 4.2's refusal and its
// reasoning: a ticket that silently lacks what its author wrote is worse than a
// stopped command.
func TestReadDocumentRefusesAPathThatIsNotThere(t *testing.T) {
	_, err := ReadDocument(filepath.Join(t.TempDir(), "nope.md"))
	if err == nil {
		t.Fatal("a missing path was accepted")
	}
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeInvalidField || e.Field != "file" {
		t.Fatalf("err = %v, want invalid_field on file", err)
	}
	if !strings.Contains(err.Error(), "nope.md") {
		t.Errorf("the refusal does not name the path: %v", err)
	}
}

// TestCreateFilesAWholeDocument is the first criterion end to end: one call,
// and the checklists and sections arrive intact.
func TestCreateFilesAWholeDocument(t *testing.T) {
	s := newTestStore(t)
	doc, err := ReadDocument(writeDocument(t, wholeTicket))
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Create(context.Background(), CreateOptions{Document: doc, Actor: testActor})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := res.Ticket

	if got.Title != "Add token refresh handling" {
		t.Errorf("Title = %q, want the document's", got.Title)
	}
	if got.Type != "bug" || got.Priority != "high" {
		t.Errorf("Type = %q, Priority = %q", got.Type, got.Priority)
	}
	// The gate of 6.2.1 holds: a document cannot promote itself out of draft.
	if got.Status != StatusDraft {
		t.Errorf("Status = %q, want draft", got.Status)
	}
	if n := len(Checklist(got.Body.AcceptanceCriteria)); n != 2 {
		t.Errorf("acceptance criteria = %d, want 2", n)
	}
	if n := len(Checklist(got.Body.DefinitionOfDone)); n != 1 {
		t.Errorf("definition of done = %d, want 1", n)
	}
	if !strings.Contains(got.Body.ImplementationPlan, "replay the request once") {
		t.Errorf("the implementation plan did not survive: %q", got.Body.ImplementationPlan)
	}
}

// TestAnExplicitFieldBeatsTheDocument is the third criterion. The rule is the
// one a template already follows, so a document is a starting point and not an
// argument.
func TestAnExplicitFieldBeatsTheDocument(t *testing.T) {
	s := newTestStore(t)
	doc, err := ReadDocument(writeDocument(t, wholeTicket))
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Create(context.Background(), CreateOptions{
		Document: doc,
		Title:    "What the caller typed",
		Type:     "chore",
		Priority: "low",
		Actor:    testActor,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := res.Ticket

	if got.Title != "What the caller typed" {
		t.Errorf("Title = %q, want the flag to win", got.Title)
	}
	if got.Type != "chore" || got.Priority != "low" {
		t.Errorf("Type = %q, Priority = %q, want the flags to win", got.Type, got.Priority)
	}
	// What the caller did not name still comes from the document.
	if !strings.Contains(got.Body.Description, "drops the refresh token") {
		t.Errorf("the document's description was lost: %q", got.Body.Description)
	}
}

// TestADocumentKeepsItsTicks pins behaviour rather than defending it, and plan
// section 15 carries the question with its trigger.
//
// A `- [x]` in the document survives into the filed ticket. That follows from
// 4.3 sharing the loader of 4.2 rather than from a ruling: a template's
// checklist seeds as rendered lines and a document's does the same. `--from`
// and `import --adopt` both untick instead, so the three do not agree, and the
// backport of 6.2.1 is the case that stops the answer being obvious.
//
// If this starts failing because somebody decided the question, that is the
// decision landing and not a regression. Change the test with the ruling.
func TestADocumentKeepsItsTicks(t *testing.T) {
	s := newTestStore(t)
	doc, err := ReadDocument(writeDocument(t, `---
title: Filed with one box already ticked
---

## Acceptance criteria

- [x] The first one, which the author says is done
- [ ] The second one, which is not
`))
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.Create(context.Background(), CreateOptions{Document: doc, Actor: testActor})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	items := Checklist(res.Ticket.Body.AcceptanceCriteria)
	if len(items) != 2 {
		t.Fatalf("acceptance criteria = %d, want 2", len(items))
	}
	if !items[0].Checked || items[1].Checked {
		t.Errorf("ticks = %v, %v, want true, false as the document wrote them", items[0].Checked, items[1].Checked)
	}
}

// TestADocumentWithNoTitleRefuses keeps 4.3's relaxation honest. --title is
// optional with a document because the document supplies it, so a document that
// supplies none leaves the ordinary requirement in force.
func TestADocumentWithNoTitleRefuses(t *testing.T) {
	s := newTestStore(t)
	doc, err := ReadDocument(writeDocument(t, "---\ntype: task\n---\n\n## Description\n\nNo title anywhere.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), CreateOptions{Document: doc, Actor: testActor}); err == nil {
		t.Fatal("a titleless document was filed")
	}
}

// TestADocumentRefusesASecondSeedSource is the rule --from and --template
// already follow: overlapping sources need a precedence nobody would remember.
func TestADocumentRefusesASecondSeedSource(t *testing.T) {
	s := newTestStore(t)
	doc, err := ReadDocument(writeDocument(t, wholeTicket))
	if err != nil {
		t.Fatal(err)
	}
	made, err := s.Create(context.Background(), CreateOptions{Title: "A source", Actor: testActor})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		opts CreateOptions
	}{
		{"with a template", CreateOptions{Document: doc, Template: "bug", Actor: testActor}},
		{"with --from", CreateOptions{Document: doc, From: made.Ticket.ID, Actor: testActor}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Create(context.Background(), tc.opts); err == nil {
				t.Fatal("two seed sources were accepted")
			}
		})
	}
}
