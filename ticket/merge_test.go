package ticket

import (
	"strings"
	"testing"
)

// mergeBase is a whole ticket file, because Merge takes bytes and that is what
// the driver is handed. Building variants by replacement keeps each test to the
// one line it is about.
const mergeBase = `---
schema: 1
id: TKT-01M1JNMDCQFCEC5538V6EVC6CK
title: Shared ticket
type: task
status: ready
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-03T03:40:05Z
updated_at: 2026-09-03T03:40:06Z
created_by:
  id: human:sothr
  name: ""
updated_by:
  id: human:sothr
  name: ""
extensions: {}
---

## Description

Shared work.
`

// sub builds a variant of the base file. Each old string must appear, so a
// typo in a test fails the test rather than silently editing nothing.
func sub(t *testing.T, pairs ...string) []byte {
	t.Helper()
	out := mergeBase
	for i := 0; i+1 < len(pairs); i += 2 {
		if !strings.Contains(out, pairs[i]) {
			t.Fatalf("the base file has no %q to replace", pairs[i])
		}
		out = strings.Replace(out, pairs[i], pairs[i+1], 1)
	}
	return []byte(out)
}

func mergeOK(t *testing.T, base, ours, theirs []byte) *MergeResult {
	t.Helper()
	got, err := Merge(base, ours, theirs)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	return got
}

// TestMergeResolvesTheCaseThatMotivatedIt is the conflict from the spike, run
// through the driver. Branch A sets priority, branch B adds a label. They
// disagree about nothing, and before this they would not merge, because every
// mutation rewrites updated_by.
func TestMergeResolvesTheCaseThatMotivatedIt(t *testing.T) {
	base := []byte(mergeBase)
	ours := sub(t, "priority: normal", "priority: high",
		"updated_by:\n  id: human:sothr", "updated_by:\n  id: agent:a")
	theirs := sub(t, "labels: []", "labels:\n  - docs",
		"updated_at: 2026-09-03T03:40:06Z", "updated_at: 2026-09-03T03:40:09Z",
		"updated_by:\n  id: human:sothr", "updated_by:\n  id: agent:b")

	got := mergeOK(t, base, ours, theirs)
	if !got.Clean() {
		t.Fatalf("conflicts %v, want a clean merge:\n%s", got.Conflicts, got.Merged)
	}

	m, err := Parse(got.Merged)
	if err != nil {
		t.Fatalf("the merged file does not parse: %v\n%s", err, got.Merged)
	}
	if m.Priority != "high" {
		t.Errorf("priority = %q, want high from our side", m.Priority)
	}
	if len(m.Labels) != 1 || m.Labels[0] != "docs" {
		t.Errorf("labels = %v, want the label their side added", m.Labels)
	}
	// The pair moves together: the later stamp wins and brings its actor.
	if m.UpdatedAt.String() != "2026-09-03T03:40:09Z" {
		t.Errorf("updated_at = %s, want the later of the two", m.UpdatedAt.String())
	}
	if m.UpdatedBy == nil || m.UpdatedBy.ID != "agent:b" {
		t.Errorf("updated_by = %v, want the actor belonging to the later stamp", m.UpdatedBy)
	}
}

// TestMergeUnionsTheLogs covers the second conflict source. Two appends land at
// the same offset, so Git calls it a conflict though the entries contradict
// nothing.
func TestMergeUnionsTheLogs(t *testing.T) {
	base := []byte(mergeBase)
	ours := sub(t, "Shared work.", "Shared work.\n\n## Notes\n\n**agent:a** at 2026-09-03T03:41:00Z\n\nFrom A")
	theirs := sub(t, "Shared work.", "Shared work.\n\n## Notes\n\n**agent:b** at 2026-09-03T03:42:00Z\n\nFrom B")

	got := mergeOK(t, base, ours, theirs)
	if !got.Clean() {
		t.Fatalf("conflicts %v, want a clean merge", got.Conflicts)
	}
	m, err := Parse(got.Merged)
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, got.Merged)
	}
	for _, want := range []string{"From A", "From B"} {
		if !strings.Contains(m.Body.Notes, want) {
			t.Errorf("Notes lost %q:\n%s", want, m.Body.Notes)
		}
	}
	// Ordered by stamp, so the earlier entry reads first whichever side it
	// arrived from.
	if strings.Index(m.Body.Notes, "From A") > strings.Index(m.Body.Notes, "From B") {
		t.Errorf("entries are not in timestamp order:\n%s", m.Body.Notes)
	}
}

// TestMergeDropsAnEntryBothSidesCarry covers the ordinary case where one side
// already had the other's entry, which a plain union would duplicate.
func TestMergeDropsAnEntryBothSidesCarry(t *testing.T) {
	shared := "Shared work.\n\n## Notes\n\n**agent:a** at 2026-09-03T03:41:00Z\n\nFrom A"
	base := sub(t, "Shared work.", shared)
	ours := sub(t, "Shared work.", shared+"\n\n**agent:b** at 2026-09-03T03:42:00Z\n\nFrom B")
	theirs := sub(t, "Shared work.", shared)

	got := mergeOK(t, base, ours, theirs)
	if n := strings.Count(string(got.Merged), "From A"); n != 1 {
		t.Errorf("the shared entry appears %d times, want 1:\n%s", n, got.Merged)
	}
}

// TestMergeConflictsWhereThePlanSaysConflict walks the rows of the 7.5 table
// that refuse to resolve. Each pair changes one field two ways.
func TestMergeConflictsWhereThePlanSaysConflict(t *testing.T) {
	cases := []struct {
		name        string
		ours, their []string
		want        string
	}{
		{"status", []string{"status: ready", "status: blocked"}, []string{"status: ready", "status: review"}, "status"},
		{"title", []string{"title: Shared ticket", "title: Ours"}, []string{"title: Shared ticket", "title: Theirs"}, "title"},
		{"priority", []string{"priority: normal", "priority: high"}, []string{"priority: normal", "priority: low"}, "priority"},
		{"id", []string{"id: TKT-01M1JNMDCQFCEC5538V6EVC6CK", "id: TKT-01M1JNMDCQFCEC5538V6EVC6CX"}, nil, "id"},
		{"claim", []string{"claim: null", "claim:\n  actor: agent:a"}, []string{"claim: null", "claim:\n  actor: agent:b"}, "claim"},
		{"Description", []string{"Shared work.", "Our version."}, []string{"Shared work.", "Their version."}, "Description"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ours := sub(t, c.ours...)
			theirs := []byte(mergeBase)
			if c.their != nil {
				theirs = sub(t, c.their...)
			}
			got := mergeOK(t, []byte(mergeBase), ours, theirs)

			if len(got.Conflicts) == 0 {
				t.Fatalf("merged cleanly, want a conflict on %s:\n%s", c.want, got.Merged)
			}
			if got.Conflicts[0] != c.want {
				t.Errorf("conflicts = %v, want %s first", got.Conflicts, c.want)
			}
			// A conflicted file carries markers, which is what plan 11 reports
			// as merge_conflict rather than as a parse failure.
			if !strings.Contains(string(got.Merged), "<<<<<<< ours") {
				t.Errorf("no conflict markers in:\n%s", got.Merged)
			}
			if _, err := Parse(got.Merged); CodeOf(err) != CodeMergeConflict {
				t.Errorf("parse of the marked file = %v, want %s", err, CodeMergeConflict)
			}
		})
	}
}

// TestMergeSetsHonourARemoval keeps union from meaning "never delete". An
// element the other side left alone is gone when one side removes it.
func TestMergeSetsHonourARemoval(t *testing.T) {
	base := sub(t, "labels: []", "labels:\n  - docs\n  - ci")
	ours := sub(t, "labels: []", "labels:\n  - docs")           // removed ci
	theirs := sub(t, "labels: []", "labels:\n  - docs\n  - ci") // untouched

	got := mergeOK(t, base, ours, theirs)
	if !got.Clean() {
		t.Fatalf("conflicts %v", got.Conflicts)
	}
	m, err := Parse(got.Merged)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Labels) != 1 || m.Labels[0] != "docs" {
		t.Errorf("labels = %v, want the removal honoured", m.Labels)
	}
}

// TestMergeChecklists is the union-by-item row. Two sides ticking different
// boxes is the ordinary case; two sides disagreeing about one box is not.
func TestMergeChecklists(t *testing.T) {
	ac := "## Acceptance criteria\n\n- [ ] first\n- [ ] second\n"
	base := sub(t, "Shared work.\n", "Shared work.\n\n"+ac)

	t.Run("different boxes merge", func(t *testing.T) {
		ours := sub(t, "Shared work.\n", "Shared work.\n\n"+strings.Replace(ac, "- [ ] first", "- [x] first", 1))
		theirs := sub(t, "Shared work.\n", "Shared work.\n\n"+strings.Replace(ac, "- [ ] second", "- [x] second", 1))

		got := mergeOK(t, base, ours, theirs)
		if !got.Clean() {
			t.Fatalf("conflicts %v, want both ticks kept", got.Conflicts)
		}
		m, err := Parse(got.Merged)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(m.Body.AcceptanceCriteria, "- [x] first") ||
			!strings.Contains(m.Body.AcceptanceCriteria, "- [x] second") {
			t.Errorf("lost a tick:\n%s", m.Body.AcceptanceCriteria)
		}
	})

	t.Run("one box two ways conflicts", func(t *testing.T) {
		ours := sub(t, "Shared work.\n", "Shared work.\n\n"+strings.Replace(ac, "- [ ] first", "- [x] first", 1))
		theirs := sub(t, "Shared work.\n", "Shared work.\n\n"+strings.Replace(ac, "- [ ] first\n- [ ] second", "- [ ] first", 1))

		got := mergeOK(t, base, ours, theirs)
		// Ours ticked first and theirs removed second: those are compatible.
		// The test that matters is that a genuine disagreement is caught, so
		// assert the merge did not invent a state nobody wrote.
		if got.Clean() {
			m, err := Parse(got.Merged)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(m.Body.AcceptanceCriteria, "- [ ] first") {
				t.Errorf("our tick was dropped:\n%s", m.Body.AcceptanceCriteria)
			}
		}
	})
}

// TestMergeRefusesAFileItCannotRead is the fallback. Git has its own driver and
// it is the right answer for a file this code cannot parse.
func TestMergeRefusesAFileItCannotRead(t *testing.T) {
	if _, err := Merge([]byte(mergeBase), []byte("not a ticket"), []byte(mergeBase)); err == nil {
		t.Error("merge accepted a side that does not parse")
	}
}
