package layout

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func routingBoard() *Board {
	b := Empty("default")
	b.Pens = map[string]Pen{
		"frontend": {Title: "Frontend", RequiredLabels: []string{"frontend"}},
		"fe-bugs":  {Title: "Frontend bugs", RequiredLabels: []string{"frontend", "bug"}},
		"backend":  {Title: "Backend", RequiredLabels: []string{"backend"}},
	}
	b.RuleOrder = []string{"frontend", "fe-bugs", "backend"}
	b.Inbox = &Point{X: 10, Y: 20}
	return b
}

func outcomes(e Explanation) []Outcome {
	out := make([]Outcome, 0, len(e.Candidates))
	for _, c := range e.Candidates {
		out = append(out, c.Outcome)
	}
	return out
}

// TestRouteIsFirstMatchInRuleOrder holds the contract of plan 12.10: the first
// pen in RuleOrder that the ticket satisfies wins, and a more specific rule
// later in the order does not outrank it. That is the decision recorded on the
// canvas repository's TKT-01M2ND1RJ and it is the opposite of what an earlier
// TypeScript evaluator did, which is why the test says it in its name.
func TestRouteIsFirstMatchInRuleOrder(t *testing.T) {
	b := routingBoard()
	got := Explain(b, RuleTicket{ID: "TKT-A", Labels: []string{"bug", "frontend"}})
	if got.Destination != "frontend" {
		t.Fatalf("destination = %q, want frontend, the earlier rule", got.Destination)
	}
	if want := []Outcome{Winner, LaterRule, MissingLabels}; !reflect.DeepEqual(outcomes(got), want) {
		t.Fatalf("outcomes = %v, want %v", outcomes(got), want)
	}
	if got.Candidates[2].Missing[0] != "backend" {
		t.Fatalf("backend rule missing = %v, want [backend]", got.Candidates[2].Missing)
	}
}

func TestRouteNoMatchGoesToInbox(t *testing.T) {
	got := Explain(routingBoard(), RuleTicket{ID: "TKT-B", Labels: []string{"docs"}})
	if !got.Inbox() || got.Destination != "" {
		t.Fatalf("destination = %q, want the inbox", got.Destination)
	}
	for _, c := range got.Candidates {
		if c.Outcome != MissingLabels {
			t.Fatalf("%s outcome = %s, want missing-labels", c.Pen, c.Outcome)
		}
	}
}

// A pinned card keeps its coordinate and routing is still reported, because
// where the card would go is the question somebody asks before releasing it.
func TestRoutePinnedCardKeepsItsCoordinate(t *testing.T) {
	b := routingBoard()
	b.Cards["TKT-C"] = Card{X: 100, Y: -40}
	got := Explain(b, RuleTicket{ID: "TKT-C", Labels: []string{"backend"}})
	if got.Pinned == nil || got.Pinned.X != 100 || got.Pinned.Y != -40 {
		t.Fatalf("pinned = %+v, want (100, -40)", got.Pinned)
	}
	if got.Destination != "backend" {
		t.Fatalf("destination = %q, want backend reported alongside the pin", got.Destination)
	}
}

func TestRouteLegacyBoardWithoutRoutingSendsEverythingToInbox(t *testing.T) {
	b, err := Parse("default", []byte("schema: 1\nboard: default\ncards:\n  TKT-D: {x: 1, y: 2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := Route(b, []RuleTicket{{ID: "TKT-E"}, {ID: "TKT-D"}})
	if len(got) != 2 || got[0].ID != "TKT-D" || got[1].ID != "TKT-E" {
		t.Fatalf("Route returned %+v, want two explanations sorted by ID", got)
	}
	if got[0].Pinned == nil || got[1].Pinned != nil {
		t.Fatalf("pins = %v/%v, want TKT-D pinned and TKT-E automatic", got[0].Pinned, got[1].Pinned)
	}
	for _, e := range got {
		if !e.Inbox() || len(e.Candidates) != 0 {
			t.Fatalf("%s: destination %q with %d candidates, want the inbox and none", e.ID, e.Destination, len(e.Candidates))
		}
	}
}

func TestMatchReportsMissingInRuleOrder(t *testing.T) {
	pen := Pen{RequiredLabels: []string{"a", "b", "c"}}
	if got := Match(pen, []string{"b"}); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("missing = %v, want [a c]", got)
	}
	if got := Match(pen, []string{"c", "a", "b", "extra"}); len(got) != 0 {
		t.Fatalf("missing = %v, want none; extra labels are allowed", got)
	}
}

// The envelope contract in plan 10.10 promises arrays, so a matched candidate
// carries an empty missingLabels and a pen with no rule carries an empty
// requiredLabels, never null. A test on the Go value would pass with nil.
func TestCandidatesSerializeArraysNotNull(t *testing.T) {
	b := routingBoard()
	b.Pens["open"] = Pen{Title: "Anything"}
	b.RuleOrder = append(b.RuleOrder, "open")
	e := Explain(b, RuleTicket{ID: "T-1", Labels: []string{"frontend", "bug"}})
	data, err := json.Marshal(e.Candidates)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "null") {
		t.Fatalf("candidates carry null:\n%s", data)
	}
}
