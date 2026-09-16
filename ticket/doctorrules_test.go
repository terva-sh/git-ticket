package ticket

import (
	"context"
	"strings"
	"testing"
	"time"
)

func shippedReport(t *testing.T, s *Store) *DoctorReport {
	t.Helper()
	r, err := s.Doctor(context.Background(), DefaultRules(), time.Now().UTC())
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	return r
}

func findingFor(r *DoctorReport, rule, id string) *DoctorFinding {
	for i := range r.Findings {
		if r.Findings[i].Rule == rule && r.Findings[i].Ticket == id {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestLabelMissingFiresOnAnUnlabelledTicket(t *testing.T) {
	s := newTestStore(t)
	bare := mustCreate(t, s, "No labels here")

	f := findingFor(shippedReport(t, s), RuleLabelMissing, bare.ID)
	if f == nil {
		t.Fatalf("label_missing did not fire on an unlabelled ticket")
	}
	if f.Level != LevelHard {
		t.Errorf("level = %q, want hard: whether a ticket has a label is settleable", f.Level)
	}
	if !strings.Contains(f.Remedy, "--add-label") {
		t.Errorf("the remedy does not say how to add one: %q", f.Remedy)
	}
	if f.File == "" {
		t.Error("the finding does not name the file")
	}
}

// TestLabelMissingReportsPresenceOnly is the criterion that keeps the two
// commands from arguing. A label outside the allowlist is check's label_unknown,
// and a ticket carrying one is labelled as far as this rule is concerned.
func TestLabelMissingReportsPresenceOnly(t *testing.T) {
	s := newTestStore(t)
	labelled := mustCreate(t, s, "Has a label nobody allowed")
	mustApply(t, s, labelled.ID, AddLabel{Label: "not-in-the-allowlist"})

	if f := findingFor(shippedReport(t, s), RuleLabelMissing, labelled.ID); f != nil {
		t.Errorf("label_missing fired on a ticket that has a label: %q", f.Message)
	}
}

// TestDoctorReadsTheOpenSet is why the report is worth reading. Hygiene is
// about a ticket somebody might pick up, and nobody picks up an archived one.
func TestDoctorReadsTheOpenSet(t *testing.T) {
	s := newTestStore(t)
	open := mustCreate(t, s, "Still open, still unlabelled")
	finished := mustCreate(t, s, "Finished and unlabelled")
	mustApply(t, s, finished.ID, SetStatus{Status: "ready"})
	mustApply(t, s, finished.ID, SetStatus{Status: "in-progress"})
	mustApply(t, s, finished.ID, SetStatus{Status: "done"})

	r := shippedReport(t, s)
	if findingFor(r, RuleLabelMissing, open.ID) == nil {
		t.Error("the open ticket was not reported")
	}
	if f := findingFor(r, RuleLabelMissing, finished.ID); f != nil {
		t.Errorf("a done ticket was reported: %q", f.Message)
	}
}

// conventionalStore builds a store that leads with area/ often enough to have a
// convention the rule will read: five ordered tickets, which is the sample
// floor, all agreeing.
func conventionalStore(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	for _, area := range []string{"cli", "core", "format", "tui", "docs"} {
		tk := mustCreate(t, s, "Conventional "+area)
		mustApply(t, s, tk.ID, AddLabel{Label: "area/" + area})
		mustApply(t, s, tk.ID, AddLabel{Label: "scope/contained"})
	}
	return s
}

// TestLabelOrderReportsTheTicketThatBreaksTheConvention is the rule, rewritten
// in v0.19.1.
//
// It asked wherever two or more labels existed until then, on the argument that
// a choice was available. That is true and useless: a store with a convention
// has made the same choice the same way every time, so asking per ticket
// reports the convention working. It fired on 120 of terva's 121 open tickets
// and on all 131 of ketju's, and a rule that fires on everything cannot guide a
// change.
func TestLabelOrderReportsTheTicketThatBreaksTheConvention(t *testing.T) {
	s := conventionalStore(t)

	odd := mustCreate(t, s, "Leads with the other dimension")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	r := shippedReport(t, s)

	var reported []string
	for _, f := range r.Findings {
		if f.Rule == RuleLabelOrder {
			reported = append(reported, f.Ticket)
		}
	}
	if len(reported) != 1 || reported[0] != odd.ID {
		t.Fatalf("label_order reported %v, want only the ticket that breaks the convention (%s)", reported, odd.ID)
	}

	f := findingFor(r, RuleLabelOrder, odd.ID)
	if f.Level != LevelSoft {
		t.Errorf("level = %q, want soft", f.Level)
	}
	if !strings.Contains(f.Message, `"scope/"`) {
		t.Errorf("the finding does not name the dimension this ticket leads with: %q", f.Message)
	}
	if !strings.Contains(f.Message, `"area/"`) {
		t.Errorf("the finding does not name the store's own convention: %q", f.Message)
	}
	if !strings.HasSuffix(strings.TrimSpace(f.Message), "?") {
		t.Errorf("a soft finding did not read as a question: %q", f.Message)
	}
	if f.Remedy == "" {
		t.Error("the finding says nothing about what would resolve it")
	}
}

// TestLabelOrderSaysNothingWhenTheStoreHasNoConvention is the half that keeps
// the rule honest. A store that has not settled which dimension leads is not
// making a mistake when its tickets disagree with each other, and reporting
// them would be this tool inventing a convention the store never adopted.
func TestLabelOrderSaysNothingWhenTheStoreHasNoConvention(t *testing.T) {
	s := newTestStore(t)
	for i, lead := range []string{"area/a", "scope/b", "area/c", "scope/d", "area/e", "scope/f"} {
		tk := mustCreate(t, s, "Split store "+lead)
		mustApply(t, s, tk.ID, AddLabel{Label: lead})
		mustApply(t, s, tk.ID, AddLabel{Label: "other/tail"})
		_ = i
	}

	for _, f := range shippedReport(t, s).Findings {
		if f.Rule == RuleLabelOrder {
			t.Errorf("label_order invented a convention in a store split evenly: %q", f.Message)
		}
	}
}

// TestLabelOrderNeedsEnoughTicketsToInferFrom stops three tickets agreeing from
// counting as a practice. A store below the floor has not labelled enough work
// to have established anything, and saying nothing is the honest answer.
// The share is held well clear of the confidence threshold throughout, at five
// sixths, so what is being measured here is the floor and not the other test's
// subject.
func TestLabelOrderNeedsEnoughTicketsToInferFrom(t *testing.T) {
	s := conventionalStore(t)
	odd := mustCreate(t, s, "The only dissenter")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	if findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID) == nil {
		t.Fatal("six ordered tickets were not enough to read a convention from")
	}

	// The floor is a parameter, so a store that wants more evidence before
	// being told anything can ask for it.
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        sample: 7\n")
	if f := findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID); f != nil {
		t.Errorf("a store that raised sample above its ticket count still got told: %q", f.Message)
	}
}

// TestLabelOrderIgnoresAStoreTooSmallToHaveAPractice is the default floor. Two
// tickets agreeing is not a convention, and inferring one from them would let
// the second ticket in a store dictate the shape of every later finding.
func TestLabelOrderIgnoresAStoreTooSmallToHaveAPractice(t *testing.T) {
	s := newTestStore(t)
	for _, area := range []string{"cli", "core"} {
		tk := mustCreate(t, s, "Small store "+area)
		mustApply(t, s, tk.ID, AddLabel{Label: "area/" + area})
		mustApply(t, s, tk.ID, AddLabel{Label: "scope/contained"})
	}
	odd := mustCreate(t, s, "The only dissenter")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	if f := findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID); f != nil {
		t.Errorf("three ordered tickets were treated as a convention: %q", f.Message)
	}
}

// TestLabelOrderConfidenceIsAParameter is the other threshold. A store with a
// weak majority is told nothing by default and can ask to be told.
func TestLabelOrderConfidenceIsAParameter(t *testing.T) {
	s := newTestStore(t)
	// Four of six lead with area/, which is 67% and under the default.
	for _, lead := range []string{"area/a", "area/b", "area/c", "area/d", "scope/e", "scope/f"} {
		tk := mustCreate(t, s, "Weak majority "+lead)
		mustApply(t, s, tk.ID, AddLabel{Label: lead})
		mustApply(t, s, tk.ID, AddLabel{Label: "other/tail"})
	}

	for _, f := range shippedReport(t, s).Findings {
		if f.Rule == RuleLabelOrder {
			t.Errorf("a two-thirds majority was treated as a convention: %q", f.Message)
		}
	}

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        confidence: 0.6\n")
	var fired int
	for _, f := range shippedReport(t, s).Findings {
		if f.Rule == RuleLabelOrder {
			fired++
		}
	}
	if fired != 2 {
		t.Errorf("label_order reported %d tickets, want the 2 that lead with the minority dimension", fired)
	}
}

// TestLabelOrderTreatsUndimensionedLabelsAsAConvention keeps the rule usable in
// a store that does not prefix at all. Leading with bare labels is as much a
// practice as leading with area/, and the one ticket that departs from it is
// worth the same question. git-ticket-canvas is the real store in this shape.
func TestLabelOrderTreatsUndimensionedLabelsAsAConvention(t *testing.T) {
	s := newTestStore(t)
	for _, l := range []string{"bug", "chore", "spike", "docs", "infra"} {
		tk := mustCreate(t, s, "Bare "+l)
		mustApply(t, s, tk.ID, AddLabel{Label: l})
		mustApply(t, s, tk.ID, AddLabel{Label: "tail"})
	}
	odd := mustCreate(t, s, "Suddenly dimensioned")
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})
	mustApply(t, s, odd.ID, AddLabel{Label: "tail"})

	f := findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID)
	if f == nil {
		t.Fatal("label_order said nothing in a store whose convention is to carry no dimension")
	}
	if !strings.Contains(f.Message, "no dimension") {
		t.Errorf("the finding does not describe the store's convention readably: %q", f.Message)
	}
}

// TestLabelOrderNamesWhatACardHides keeps the visible parameter meaningful. It
// does not decide whether the rule fires, only what the finding can say, so a
// reported ticket with more labels than a card shows gets them named.
func TestLabelOrderNamesWhatACardHides(t *testing.T) {
	s := conventionalStore(t)
	odd := mustCreate(t, s, "Three labels, wrong one leading")
	for _, l := range []string{"scope/crossing", "area/cli", "gamma"} {
		mustApply(t, s, odd.ID, AddLabel{Label: l})
	}

	f := findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID)
	if f == nil {
		t.Fatal("label_order did not fire on the ticket that breaks the convention")
	}
	if !strings.Contains(f.Message, "gamma") {
		t.Errorf("the finding does not name the hidden label: %q", f.Message)
	}

	// Compact cards show three, so nothing is hidden. The rule still reports,
	// because the convention is still broken, and the message stops claiming
	// anything is out of sight.
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        visible: 3\n")

	f = findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID)
	if f == nil {
		t.Fatal("label_order stopped reporting when the store widened the card")
	}
	if strings.Contains(f.Message, "hiding") {
		t.Errorf("the finding still claims a label is hidden on a card that shows three: %q", f.Message)
	}
}

// TestLabelOrderSurvivesAnUnreadableThreshold keeps one bad parameter from
// costing the whole report. A rule is advice, and refusing to give any because
// a number was misspelled spends the run on a typo.
func TestLabelOrderSurvivesAnUnreadableThreshold(t *testing.T) {
	s := conventionalStore(t)
	odd := mustCreate(t, s, "Breaks the convention")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        visible: sometimes\n        confidence: mostly\n        sample: a few\n")

	if findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID) == nil {
		t.Error("unreadable parameters silenced the rule rather than falling back")
	}
}

// TestShippedRulesNeverFailARunOnTheirOwn holds the pair to the levels they
// claim: the soft one may reach the informational grade and may not pass it.
func TestShippedRulesNeverFailARunOnTheirOwn(t *testing.T) {
	s := conventionalStore(t)
	odd := mustCreate(t, s, "Breaks the convention")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	r := shippedReport(t, s)
	for _, f := range r.Findings {
		if f.Rule != RuleLabelOrder {
			t.Fatalf("expected only the soft finding, got %s on %s", f.Rule, f.Ticket)
		}
	}
	if g := r.Grade(); g != GradeSoft {
		t.Errorf("grade = %v, want soft: a labelled ticket must not produce a hard finding", g)
	}
}

// TestAConventionalStoreIsQuiet is the point of the rewrite. Every ticket
// following the store's own practice is the case doctor should have nothing to
// say about, and it is what the old rule got wrong on every store that had one.
func TestAConventionalStoreIsQuiet(t *testing.T) {
	r := shippedReport(t, conventionalStore(t))
	if len(r.Findings) != 0 {
		t.Errorf("a store where every ticket follows its own convention got %d findings: %+v",
			len(r.Findings), r.Findings)
	}
	if g := r.Grade(); g != GradeClean {
		t.Errorf("grade = %v, want clean", g)
	}
}

// TestTheTwoRulesDoNotFireOnTheSameTicketTwice is the pair read together. A
// ticket with no labels cannot have an ordering problem, and the rules should
// not both be talking about it.
func TestTheTwoRulesDoNotFireOnTheSameTicketTwice(t *testing.T) {
	s := newTestStore(t)
	bare := mustCreate(t, s, "Nothing at all")

	r := shippedReport(t, s)
	if findingFor(r, RuleLabelOrder, bare.ID) != nil {
		t.Error("label_order fired on a ticket with no labels")
	}
	if findingFor(r, RuleLabelMissing, bare.ID) == nil {
		t.Error("label_missing did not fire on a ticket with no labels")
	}
}

// TestLabelOrderMinimumIsAParameter keeps min meaningful under the new rule. It
// governs which tickets count as ordered at all, so raising it removes them
// from the evidence and from the report together.
func TestLabelOrderMinimumIsAParameter(t *testing.T) {
	s := conventionalStore(t)
	odd := mustCreate(t, s, "Two labels, minority dimension leading")
	mustApply(t, s, odd.ID, AddLabel{Label: "scope/crossing"})
	mustApply(t, s, odd.ID, AddLabel{Label: "area/cli"})

	if findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID) == nil {
		t.Fatal("label_order did not report the ticket that breaks the convention")
	}

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        min: 3\n")
	if f := findingFor(shippedReport(t, s), RuleLabelOrder, odd.ID); f != nil {
		t.Errorf("a store that raised min still got asked about a two-label ticket: %q", f.Message)
	}
}

// TestLabelOrderMinimumRefusesToGoBelowTwo keeps a store from configuring noise
// it can learn nothing from: below two labels there is no ordering at all.
func TestLabelOrderMinimumRefusesToGoBelowTwo(t *testing.T) {
	s := conventionalStore(t)
	one := mustCreate(t, s, "A single label")
	mustApply(t, s, one.ID, AddLabel{Label: "scope/lonely"})

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        min: 1\n")
	if f := findingFor(shippedReport(t, s), RuleLabelOrder, one.ID); f != nil {
		t.Errorf("min below two produced a finding with nothing to ask: %q", f.Message)
	}
}
