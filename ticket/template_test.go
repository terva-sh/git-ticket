package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemplate puts a template file into a test store. Templates in a
// real store are reviewed files; a test store is built at runtime like
// its tickets are.
func writeTemplate(t *testing.T, s *Store, name, content string) {
	t.Helper()
	dir := filepath.Join(s.Path(), "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const bugTemplate = `---
type: bug
priority: high
labels:
  - regression
assignees:
  - human:sothr
milestone: v1.0
---

## Description

Steps to reproduce:

1. ...

## Acceptance criteria

- [ ] reproduced
- [ ] regression test added

## Definition of done

- [ ] the fix names the cause

## Implementation plan

Bisect first.
`

func TestCreateFromTemplateSeedsEveryField(t *testing.T) {
	s := newTestStore(t)
	writeTemplate(t, s, "bug", bugTemplate)

	res, err := s.Create(context.Background(), CreateOptions{
		Title:    "Clock skew breaks refresh",
		Template: "bug",
		Actor:    testActor,
	})
	if err != nil {
		t.Fatalf("create --template: %v", err)
	}
	tk := res.Ticket
	if tk.Type != "bug" || tk.Priority != "high" {
		t.Fatalf("type/priority = %s/%s, want bug/high", tk.Type, tk.Priority)
	}
	if len(tk.Labels) != 1 || tk.Labels[0] != "regression" {
		t.Fatalf("labels = %v", tk.Labels)
	}
	if len(tk.Assignees) != 1 || tk.Assignees[0] != "human:sothr" {
		t.Fatalf("assignees = %v", tk.Assignees)
	}
	if tk.Milestone == nil || *tk.Milestone != "v1.0" {
		t.Fatalf("milestone = %v", tk.Milestone)
	}
	if !strings.Contains(tk.Body.Description, "Steps to reproduce") {
		t.Fatalf("description = %q", tk.Body.Description)
	}
	if !strings.Contains(tk.Body.AcceptanceCriteria, "- [ ] reproduced") ||
		!strings.Contains(tk.Body.AcceptanceCriteria, "- [ ] regression test added") {
		t.Fatalf("ac = %q", tk.Body.AcceptanceCriteria)
	}
	if !strings.Contains(tk.Body.DefinitionOfDone, "names the cause") {
		t.Fatalf("dod = %q", tk.Body.DefinitionOfDone)
	}
	if !strings.Contains(tk.Body.ImplementationPlan, "Bisect first") {
		t.Fatalf("plan = %q", tk.Body.ImplementationPlan)
	}
	// Nothing lifecycle-shaped seeds: a template create is a draft.
	if tk.Status != StatusDraft {
		t.Fatalf("status = %q, want draft", tk.Status)
	}
}

func TestExplicitOptionsWinOverTheTemplate(t *testing.T) {
	s := newTestStore(t)
	writeTemplate(t, s, "bug", bugTemplate)

	res, err := s.Create(context.Background(), CreateOptions{
		Title:              "Explicit wins",
		Template:           "bug",
		Type:               "chore",
		Priority:           "low",
		Description:        "My own words.",
		AcceptanceCriteria: []string{"only this"},
		Actor:              testActor,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tk := res.Ticket
	if tk.Type != "chore" || tk.Priority != "low" {
		t.Fatalf("type/priority = %s/%s, want the explicit chore/low", tk.Type, tk.Priority)
	}
	if tk.Body.Description != "My own words." {
		t.Fatalf("description = %q, want the explicit one", tk.Body.Description)
	}
	if strings.Contains(tk.Body.AcceptanceCriteria, "reproduced") ||
		!strings.Contains(tk.Body.AcceptanceCriteria, "only this") {
		t.Fatalf("ac = %q, want the explicit items wholesale", tk.Body.AcceptanceCriteria)
	}
	// Fields the options left empty still seed: labels arrive from the
	// template, per the per-field rule.
	if len(tk.Labels) != 1 || tk.Labels[0] != "regression" {
		t.Fatalf("labels = %v, want the template's", tk.Labels)
	}
}

func TestMissingTemplateRefusesTheCreate(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Create(context.Background(), CreateOptions{
		Title:    "No such seed",
		Template: "nonexistent",
		Actor:    testActor,
	})
	if CodeOf(err) != CodeInvalidField {
		t.Fatalf("err = %v, want %s", err, CodeInvalidField)
	}
	if !strings.Contains(err.Error(), "templates") {
		t.Fatalf("refusal does not name the templates directory: %v", err)
	}
}

// TestTemplateIgnoresLifecycleFields: a template made by copying a real
// ticket carries id, status, and timestamps, and the loader ignores
// them rather than refusing, per plan 4.2.
func TestTemplateIgnoresLifecycleFields(t *testing.T) {
	s := newTestStore(t)
	writeTemplate(t, s, "copied", `---
schema: 1
id: TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZZZ
title: The old title
status: done
type: spike
priority: urgent
created_at: 2020-01-01T00:00:00Z
---

## Description

Copied from a real ticket.
`)
	res, err := s.Create(context.Background(), CreateOptions{
		Title:    "Fresh from a copied ticket",
		Template: "copied",
		Actor:    testActor,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tk := res.Ticket
	if tk.Status != StatusDraft {
		t.Fatalf("status = %q; the template's done leaked through", tk.Status)
	}
	if tk.ID == "TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZZZ" {
		t.Fatal("the template's id leaked through")
	}
	if tk.Title != "Fresh from a copied ticket" {
		t.Fatalf("title = %q; the template's title leaked through", tk.Title)
	}
	// The seedable fields still arrive.
	if tk.Type != "spike" || tk.Priority != "urgent" {
		t.Fatalf("type/priority = %s/%s, want the template's spike/urgent", tk.Type, tk.Priority)
	}
}

func TestTemplatesListsSorted(t *testing.T) {
	s := newTestStore(t)
	if names, err := s.Templates(); err != nil || names != nil {
		t.Fatalf("no templates dir: names=%v err=%v, want nil/nil", names, err)
	}
	writeTemplate(t, s, "bug", bugTemplate)
	writeTemplate(t, s, "adr", "---\ntype: spike\n---\n\n## Description\n\nContext:\n")
	names, err := s.Templates()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "adr" || names[1] != "bug" {
		t.Fatalf("names = %v, want [adr bug]", names)
	}
}

// TestTemplateWithBadTypeRefuses: the template's values pass through
// the same validation as typed ones.
func TestTemplateWithBadTypeRefuses(t *testing.T) {
	s := newTestStore(t)
	writeTemplate(t, s, "weird", "---\ntype: saga\n---\n\n## Description\n\nX\n")
	_, err := s.Create(context.Background(), CreateOptions{
		Title:    "Bad seed",
		Template: "weird",
		Actor:    testActor,
	})
	if CodeOf(err) != CodeInvalidField {
		t.Fatalf("err = %v, want %s for the template's unknown type", err, CodeInvalidField)
	}
}
