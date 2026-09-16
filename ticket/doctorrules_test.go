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

// TestLabelOrderAsksWhereverTheOrderIsAChoice is the threshold, changed from
// "a label is hidden" to "a choice exists" on 2026-09-16.
//
// The first label is the primary one, and git-ticket-canvas states the same
// convention independently. A ticket with two labels has a real decision about
// which leads even though both are on screen, so a rule tied to hiding was
// asking about visibility when the question is about primacy.
func TestLabelOrderAsksWhereverTheOrderIsAChoice(t *testing.T) {
	s := newTestStore(t)

	one := mustCreate(t, s, "One label, no ordering to make")
	mustApply(t, s, one.ID, AddLabel{Label: "alpha"})

	two := mustCreate(t, s, "Two labels, both on screen, still a choice")
	mustApply(t, s, two.ID, AddLabel{Label: "alpha"})
	mustApply(t, s, two.ID, AddLabel{Label: "beta"})

	r := shippedReport(t, s)
	if f := findingFor(r, RuleLabelOrder, one.ID); f != nil {
		t.Errorf("label_order fired on a single label, which is not an ordering: %q", f.Message)
	}
	f := findingFor(r, RuleLabelOrder, two.ID)
	if f == nil {
		t.Fatal("label_order did not fire on two labels, where the choice is real")
	}
	if f.Level != LevelSoft {
		t.Errorf("level = %q, want soft", f.Level)
	}
	if !strings.Contains(f.Message, "alpha") {
		t.Errorf("the finding does not name the label that leads: %q", f.Message)
	}
	if !strings.HasSuffix(strings.TrimSpace(f.Message), "?") {
		t.Errorf("a soft finding did not read as a question: %q", f.Message)
	}
}

// TestLabelOrderNamesWhatACardHides keeps the visible parameter meaningful. It
// no longer decides whether the rule fires, only what the finding can say, so a
// ticket with more labels than a card shows gets them named.
func TestLabelOrderNamesWhatACardHides(t *testing.T) {
	s := newTestStore(t)
	three := mustCreate(t, s, "Three labels")
	for _, l := range []string{"alpha", "beta", "gamma"} {
		mustApply(t, s, three.ID, AddLabel{Label: l})
	}

	f := findingFor(shippedReport(t, s), RuleLabelOrder, three.ID)
	if f == nil {
		t.Fatal("label_order did not fire on three labels")
	}
	if !strings.Contains(f.Message, "gamma") {
		t.Errorf("the finding does not name the hidden label: %q", f.Message)
	}

	// Compact cards show three, so nothing is hidden. The rule still asks,
	// because the order is still a choice, and the message stops claiming
	// anything is out of sight.
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        visible: 3\n")

	f = findingFor(shippedReport(t, s), RuleLabelOrder, three.ID)
	if f == nil {
		t.Fatal("label_order stopped asking when the store widened the card")
	}
	if strings.Contains(f.Message, "hiding") {
		t.Errorf("the finding still claims a label is hidden on a card that shows three: %q", f.Message)
	}
}

// TestLabelOrderSurvivesAnUnreadableThreshold keeps one bad parameter from
// costing the whole report. A rule is advice, and refusing to give any because
// a number was misspelled spends the run on a typo.
func TestLabelOrderSurvivesAnUnreadableThreshold(t *testing.T) {
	s := newTestStore(t)
	three := mustCreate(t, s, "Three labels")
	for _, l := range []string{"alpha", "beta", "gamma"} {
		mustApply(t, s, three.ID, AddLabel{Label: l})
	}
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        visible: sometimes\n")

	if findingFor(shippedReport(t, s), RuleLabelOrder, three.ID) == nil {
		t.Error("an unreadable threshold silenced the rule rather than falling back")
	}
}

// TestShippedRulesNeverFailARunOnTheirOwn holds the pair to the levels they
// claim: the soft one may reach the informational grade and may not pass it.
func TestShippedRulesNeverFailARunOnTheirOwn(t *testing.T) {
	s := newTestStore(t)
	three := mustCreate(t, s, "Three labels, all of them")
	for _, l := range []string{"alpha", "beta", "gamma"} {
		mustApply(t, s, three.ID, AddLabel{Label: l})
	}

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

// TestLabelOrderMinimumIsAParameter is the escape hatch for a store whose
// convention already settles which dimension leads.
//
// Measured when the default moved from three labels to two: terva, which
// documents area/ before scope/ and carries two labels on nearly every ticket,
// went from 0 soft findings to 48. Every one of those asks a question terva
// answered once, in writing. Raising min is the narrow answer; turning the rule
// off is the blunt one, and a store should not have to reach for the blunt one.
func TestLabelOrderMinimumIsAParameter(t *testing.T) {
	s := newTestStore(t)
	two := mustCreate(t, s, "Two labels, order settled by convention")
	mustApply(t, s, two.ID, AddLabel{Label: "area/core"})
	mustApply(t, s, two.ID, AddLabel{Label: "scope/contained"})

	if findingFor(shippedReport(t, s), RuleLabelOrder, two.ID) == nil {
		t.Fatal("label_order did not fire at the default of two")
	}

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        min: 3\n")
	if f := findingFor(shippedReport(t, s), RuleLabelOrder, two.ID); f != nil {
		t.Errorf("a store that raised min still got asked: %q", f.Message)
	}
}

// TestLabelOrderMinimumRefusesToGoBelowTwo keeps a store from configuring noise
// it can learn nothing from: below two labels there is no ordering to ask about.
func TestLabelOrderMinimumRefusesToGoBelowTwo(t *testing.T) {
	s := newTestStore(t)
	one := mustCreate(t, s, "A single label")
	mustApply(t, s, one.ID, AddLabel{Label: "alpha"})

	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        min: 1\n")
	if f := findingFor(shippedReport(t, s), RuleLabelOrder, one.ID); f != nil {
		t.Errorf("min below two produced a finding with nothing to ask: %q", f.Message)
	}
}
