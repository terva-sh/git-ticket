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

// TestLabelOrderFiresOnlyWhenALabelIsHidden is the threshold the rule's whole
// justification rests on. With two labels and a card that shows two, order
// changes nothing a reader sees, and a rule that fired there would be asking
// about a choice that does not exist.
func TestLabelOrderFiresOnlyWhenALabelIsHidden(t *testing.T) {
	s := newTestStore(t)
	two := mustCreate(t, s, "Two labels, both visible")
	mustApply(t, s, two.ID, AddLabel{Label: "alpha"})
	mustApply(t, s, two.ID, AddLabel{Label: "beta"})

	three := mustCreate(t, s, "Three labels, one hidden")
	mustApply(t, s, three.ID, AddLabel{Label: "alpha"})
	mustApply(t, s, three.ID, AddLabel{Label: "beta"})
	mustApply(t, s, three.ID, AddLabel{Label: "gamma"})

	r := shippedReport(t, s)
	if f := findingFor(r, RuleLabelOrder, two.ID); f != nil {
		t.Errorf("label_order fired where nothing is hidden: %q", f.Message)
	}
	f := findingFor(r, RuleLabelOrder, three.ID)
	if f == nil {
		t.Fatal("label_order did not fire where a label is hidden")
	}
	if f.Level != LevelSoft {
		t.Errorf("level = %q, want soft", f.Level)
	}
	// It names what is out of sight, which is the part it can see.
	if !strings.Contains(f.Message, "gamma") {
		t.Errorf("the finding does not name the hidden label: %q", f.Message)
	}
	// And asks rather than rules, which is the part it cannot.
	if !strings.HasSuffix(strings.TrimSpace(f.Message), "?") {
		t.Errorf("a soft finding did not read as a question: %q", f.Message)
	}
}

// TestLabelOrderTakesItsThresholdFromParams is the first shipped rule to read a
// parameter, and the reason the framework carries them: the number comes from
// how a board renders, and not every store reads its tickets on the same one.
func TestLabelOrderTakesItsThresholdFromParams(t *testing.T) {
	s := newTestStore(t)
	three := mustCreate(t, s, "Three labels")
	for _, l := range []string{"alpha", "beta", "gamma"} {
		mustApply(t, s, three.ID, AddLabel{Label: l})
	}
	// Compact cards show three, so nothing is hidden and the rule says nothing.
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    label_order:\n      params:\n        visible: 3\n")

	if f := findingFor(shippedReport(t, s), RuleLabelOrder, three.ID); f != nil {
		t.Errorf("label_order ignored the store's threshold: %q", f.Message)
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
