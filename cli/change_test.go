package cli

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// TestEveryChangeKindHasALine keeps the fallback in changeLine unreachable.
//
// The change vocabulary exists so a loss is named rather than silent, so a kind
// the library reports and this binary has no wording for would defeat the whole
// point. The fallback still prints something, because printing the kind badly
// beats dropping it, but a kind added to ticket.ChangeKinds and not to
// changeLine should fail here rather than reach a reader.
func TestEveryChangeKindHasALine(t *testing.T) {
	for _, k := range ticket.ChangeKinds() {
		// A value and a count on every kind, so the ones that ignore them are
		// exercised too. What is asserted is the wording, not the plumbing.
		got := changeLine(ticket.Change{Kind: k, Value: "probe", Count: 3})
		if got == "" {
			t.Errorf("%s renders as nothing", k)
			continue
		}
		if strings.Contains(got, "does not have wording for") {
			t.Errorf("%s reaches the fallback: %s", k, got)
		}
		if !strings.Contains(got, ": ") {
			t.Errorf("%s does not read as \"subject: what happens\": %s", k, got)
		}
	}
}

// TestChangeLineHasNoTense is the property that lets one wording serve both the
// preview, which has filed nothing, and the write, which has.
//
// A line that said "dropped" would be wrong in the preview and a line that said
// "will drop" would be wrong in the report, and keeping two sets in step is the
// work this avoids.
func TestChangeLineHasNoTense(t *testing.T) {
	past := []string{"dropped", "was ", "were ", "has been", "have been"}
	future := []string{"will ", "would ", "going to"}
	for _, k := range ticket.ChangeKinds() {
		got := changeLine(ticket.Change{Kind: k, Value: "probe", Count: 3})
		for _, w := range past {
			// "carried" is not in the list. It is a participle doing adjective
			// work in "2 carried, every box unchecked" and in "carried as one
			// note", and both read the same before and after the write.
			if strings.Contains(got, w) {
				t.Errorf("%s reads as past tense (%q): %s", k, w, got)
			}
		}
		for _, w := range future {
			if strings.Contains(got, w) {
				t.Errorf("%s reads as future tense (%q): %s", k, w, got)
			}
		}
	}
}

// TestChangeLineFallbackNamesTheKind covers the branch the test above keeps
// unreachable. It is the behaviour an older binary meeting a newer library
// would show, and it is worth knowing it names the kind rather than printing an
// empty line.
func TestChangeLineFallbackNamesTheKind(t *testing.T) {
	got := changeLine(ticket.Change{Kind: "something_new", Value: "x"})
	if !strings.Contains(got, "something_new") || !strings.Contains(got, "x") {
		t.Errorf("the fallback loses the kind or the value: %s", got)
	}

	bare := changeLine(ticket.Change{Kind: "something_new"})
	if !strings.Contains(bare, "something_new") {
		t.Errorf("the valueless fallback loses the kind: %s", bare)
	}
	if strings.Contains(bare, " : ") {
		t.Errorf("the valueless fallback has a gap where the value would be: %s", bare)
	}
}
